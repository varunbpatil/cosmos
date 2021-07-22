package temporal

import (
	"context"
	"cosmos"
	"errors"
	"log"
	"sync"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigDefault

type Workflow struct {
	*cosmos.App
}

func NewWorkflow() *Workflow {
	return &Workflow{}
}

// A wrapper for the run object with a mutex so that the object can be modified concurrently inside an activity.
type RunWrapper struct {
	sync.Mutex
	*cosmos.Run
}

// DeepCopy deepcopies a to b using json marshaling.
func DeepCopy(a, b interface{}) {
	byt, _ := json.Marshal(a)
	json.Unmarshal(byt, b)
}

func withActivityOptions(ctx workflow.Context, queue string, maxAttempts int32) workflow.Context {
	ao := workflow.ActivityOptions{
		TaskQueue:           queue,
		WaitForCancellation: true,
		StartToCloseTimeout: 3 * 24 * time.Hour,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Minute,
			BackoffCoefficient: 2.0,
			MaximumInterval:    10 * time.Minute,
			MaximumAttempts:    maxAttempts,
		},
	}
	ctxOut := workflow.WithActivityOptions(ctx, ao)
	return ctxOut
}

// Cancellation is only delivered to activities that heartbeat.
func (w *Workflow) StartHeartbeat(ctx context.Context, period time.Duration, run *RunWrapper) chan<- struct{} {
	ch := make(chan struct{})

	go func() {
		recordHeartbeat := func() {
			if run != nil {
				// Make a copy of the run so that we don't hold the lock while recording activity heartbeat.
				runCopy := &RunWrapper{}
				run.Lock()
				DeepCopy(run, runCopy)
				run.Unlock()

				// Record activity heartbeat.
				activity.RecordHeartbeat(ctx, runCopy)

				// Best effort stats updation. Errors are ignored.
				numRecords := runCopy.Stats.NumRecords
				executionStart := runCopy.Stats.ExecutionStart
				executionEnd := time.Now()
				w.App.UpdateRun(ctx, run.ID, &cosmos.RunUpdate{
					NumRecords:     &numRecords,
					ExecutionStart: &executionStart,
					ExecutionEnd:   &executionEnd,
				})
			} else {
				activity.RecordHeartbeat(ctx)
			}
		}

		for {
			select {
			case <-ch:
				return
			case <-time.After(period):
				recordHeartbeat()
			}
		}
	}()

	return ch
}

func (w *Workflow) IngestionWorkflow(ctx workflow.Context, runID int) error {
	ctx = withActivityOptions(ctx, cosmos.TemporalTaskQueue, 5)

	run := &cosmos.Run{}
	err := workflow.ExecuteActivity(ctx, w.GetRun, runID).Get(ctx, run)
	if err != nil {
		return err
	}

	err = workflow.ExecuteActivity(ctx, w.Initialize, run).Get(ctx, run)
	if err != nil {
		w.UpdateDB(ctx, run, err)
		return err
	}

	err = workflow.ExecuteActivity(ctx, w.ReplicationActivity, run).Get(ctx, run)
	if err != nil {
		w.UpdateDB(ctx, run, err)
		return err
	}

	// Normalization will be skipped if the workflow was cancelled while replication was running.
	err = workflow.ExecuteActivity(ctx, w.NormalizationActivity, run).Get(ctx, run)
	if err != nil {
		w.UpdateDB(ctx, run, err)
		return err
	}

	return w.UpdateDB(ctx, run, err)
}

func (w *Workflow) GetRun(ctx context.Context, runID int) (*cosmos.Run, error) {
	defer close(w.StartHeartbeat(ctx, 5*time.Second, nil))

	// Mark the workflow as running.
	status := cosmos.RunStatusRunning
	run, err := w.App.UpdateRun(ctx, runID, &cosmos.RunUpdate{Status: &status})
	if err != nil {
		return nil, err
	}

	// Set the execution start time.
	run.Stats.ExecutionStart = time.Now()

	// Return a run object which will used in the rest of the workflow.
	// This run object will be immune to changes made to the backend database
	// while the workflow is running.
	return run, nil
}

func (w *Workflow) Initialize(ctx context.Context, run *cosmos.Run) (*cosmos.Run, error) {
	defer close(w.StartHeartbeat(ctx, 5*time.Second, &RunWrapper{Run: run}))

	state := run.Sync.State
	srcConfig := run.Sync.SourceEndpoint.Config.ToSpec()
	dstConfig := run.Sync.DestinationEndpoint.Config.ToSpec()
	configuredCatalog := run.Sync.ConfiguredCatalog.ConfiguredCatalog

	// In order to wipe the destination clean, we do a "full_refresh - overwrite" sync
	// with the only difference being that no records are read from the source and
	// therefore no records are sent to the destination.
	if run.Options.WipeDestination {
		syncMode := cosmos.SyncModeFullRefresh
		dstSyncMode := cosmos.DestinationSyncModeOverwrite
		for i := range configuredCatalog.Streams {
			configuredCatalog.Streams[i].SyncMode = &syncMode
			configuredCatalog.Streams[i].DestinationSyncMode = &dstSyncMode
		}
	}

	artifactory, err := w.App.GetArtifactory(run.SyncID, run.ExecutionDate)
	if err != nil {
		return nil, err
	}

	defer w.App.CloseArtifactory(artifactory)

	if err := w.App.WriteArtifact(artifactory, cosmos.ArtifactBeforeState, state); err != nil {
		return nil, err
	}
	if err := w.App.WriteArtifact(artifactory, cosmos.ArtifactSrcConfig, srcConfig); err != nil {
		return nil, err
	}
	if err := w.App.WriteArtifact(artifactory, cosmos.ArtifactDstConfig, dstConfig); err != nil {
		return nil, err
	}
	if err := w.App.WriteArtifact(artifactory, cosmos.ArtifactCatalog, configuredCatalog); err != nil {
		return nil, err
	}

	return run, nil
}

func (w *Workflow) ReplicationActivity(ctx context.Context, run *cosmos.Run) (*cosmos.Run, error) {
	// Get heartbeat details from a previous attempt (if any).
	if activity.HasHeartbeatDetails(ctx) {
		if err := activity.GetHeartbeatDetails(ctx, run); err != nil {
			log.Printf("replication activity failed to get heartbeat details on retry. err: %s", err)
		}
	}

	runWrapper := &RunWrapper{Run: run}
	defer close(w.StartHeartbeat(ctx, 5*time.Second, runWrapper))

	// Current attempt number.
	attempt := activity.GetInfo(ctx).Attempt

	artifactory, err := w.App.GetArtifactory(run.SyncID, run.ExecutionDate)
	if err != nil {
		return nil, err
	}

	defer w.App.CloseArtifactory(artifactory)

	// State might have changed in the previous attempt. Write it out again.
	if err := w.App.WriteArtifact(artifactory, cosmos.ArtifactBeforeState, run.Sync.State); err != nil {
		return nil, err
	}

	workerArtifact, err := w.App.GetArtifactRef(artifactory, cosmos.ArtifactWorker, attempt)
	if err != nil {
		return nil, err
	}

	srcConnector := run.Sync.SourceEndpoint.Connector
	dstConnector := run.Sync.DestinationEndpoint.Connector

	runctx, cancel := context.WithCancel(ctx)
	runctx = cosmos.NewArtifactoryContext(runctx, artifactory)
	defer cancel()

	s1out, s1errc := w.App.Read(runctx, srcConnector, run.Options.WipeDestination)
	s2out, s2errc := w.ProcessSourceConnectorOutput(runctx, s1out, runWrapper, attempt)
	s3out, s3errc := w.App.Write(runctx, dstConnector, s2out)
	s4errc := w.ProcessDestinationConnectorOutput(runctx, s3out, runWrapper, attempt)

	cancel()

	var finalErr error
	for _, errc := range []<-chan error{s1errc, s2errc, s3errc, s4errc} {
		if err := <-errc; err != nil {
			workerArtifact.Println(err)
			finalErr = err
		}
	}

	// If you want the workflow to get partial results even on error, you should send them via ApplicationError.
	if finalErr != nil {
		return nil, temporal.NewApplicationErrorWithCause("replication activity failed", "", finalErr, run)
	}

	// TODO: Return NewCanceledError() when the activity is successfully canceled.
	//       See https://github.com/temporalio/sdk-go/blob/3f172f50c54b65639fe3265c6c28fea4cff22c5e/temporal/error.go#L177

	return run, nil
}

func (w *Workflow) NormalizationActivity(ctx context.Context, run *cosmos.Run) (*cosmos.Run, error) {
	defer close(w.StartHeartbeat(ctx, 5*time.Second, &RunWrapper{Run: run}))

	// Current attempt number.
	attempt := activity.GetInfo(ctx).Attempt

	artifactory, err := w.App.GetArtifactory(run.SyncID, run.ExecutionDate)
	if err != nil {
		return nil, err
	}

	defer w.App.CloseArtifactory(artifactory)

	workerArtifact, err := w.App.GetArtifactRef(artifactory, cosmos.ArtifactWorker, attempt)
	if err != nil {
		return nil, err
	}

	dstConnector := run.Sync.DestinationEndpoint.Connector
	basicNormalization := run.Sync.BasicNormalization

	runctx, cancel := context.WithCancel(ctx)
	runctx = cosmos.NewArtifactoryContext(runctx, artifactory)
	defer cancel()

	s1out, s1errc := w.App.Normalize(runctx, dstConnector, basicNormalization)
	s2errc := w.ProcessNormalizationOutput(runctx, s1out, attempt)

	cancel()

	var finalErr error
	for _, errc := range []<-chan error{s1errc, s2errc} {
		if err := <-errc; err != nil {
			workerArtifact.Println(err)
			finalErr = err
		}
	}

	return run, finalErr
}

func (w *Workflow) UpdateDB(ctx workflow.Context, run *cosmos.Run, err error) error {
	// If there is an error from an activity, temporal doesn't extract the
	// result from the activity. Hence, if you need partial results from
	// activities even on error, you should send them via the details field of
	// ApplicationError.
	var applicationErr *temporal.ApplicationError
	if errors.As(err, &applicationErr) {
		applicationErr.Details(run)
	}

	// Set the run status.
	run.Status = cosmos.RunStatusSuccess
	if run.Options.WipeDestination {
		run.Status = cosmos.RunStatusWiped
	}
	if err != nil {
		run.Status = cosmos.RunStatusFailed
	}
	if temporal.IsCanceledError(ctx.Err()) {
		run.Status = cosmos.RunStatusCanceled
	}

	// Set the execution end time.
	run.Stats.ExecutionEnd = time.Now()

	// Get a new disconnected context so that DB is updated even if the workflow was cancelled.
	ctx, cancel := workflow.NewDisconnectedContext(ctx)
	defer cancel()

	return workflow.ExecuteActivity(ctx, w.DBUpdateActivity, run).Get(ctx, nil)
}

func (w *Workflow) DBUpdateActivity(ctx context.Context, run *cosmos.Run) error {
	defer close(w.StartHeartbeat(ctx, 5*time.Second, &RunWrapper{Run: run}))

	// State must be updated in the sync before setting the run status to a terminal state.
	// Otherwise, cosmos scheduler may create a new run with the old state.
	if _, err := w.App.UpdateSync(ctx, run.SyncID, &cosmos.SyncUpdate{State: &run.Sync.State}); err != nil {
		return err
	}

	// Log the new state.
	artifactory, err := w.App.GetArtifactory(run.SyncID, run.ExecutionDate)
	if err != nil {
		return err
	}

	defer w.App.CloseArtifactory(artifactory)

	if err := w.App.WriteArtifact(artifactory, cosmos.ArtifactAfterState, run.Sync.State); err != nil {
		return err
	}

	// Update the run in the DB.
	_, err = w.App.UpdateRun(ctx, run.ID, &cosmos.RunUpdate{
		Status:         &run.Status,
		NumRecords:     &run.Stats.NumRecords,
		ExecutionStart: &run.Stats.ExecutionStart,
		ExecutionEnd:   &run.Stats.ExecutionEnd,
	})

	return err
}

func (w *Workflow) ProcessSourceConnectorOutput(ctx context.Context, in <-chan interface{}, run *RunWrapper, attempt int32) (<-chan *cosmos.Message, <-chan error) {
	out := make(chan *cosmos.Message, 100)
	errc := make(chan error, 1)

	go func() {
		defer close(out)

		artifactory := cosmos.ArtifactoryFromContext(ctx)
		sourceArtifact, err := w.App.GetArtifactRef(artifactory, cosmos.ArtifactSource, attempt)
		if err != nil {
			errc <- err
			return
		}

		for line := range in {
			if msg, ok := line.(*cosmos.Message); ok {
				if msg.Type == cosmos.MessageTypeRecord || msg.Type == cosmos.MessageTypeState {
					if err := sendMsgOnChannel(ctx, msg, out); err != nil {
						break
					}
					if msg.Type == cosmos.MessageTypeRecord {
						run.Lock()
						run.Stats.NumRecords++
						run.Unlock()
					}
				} else {
					b, err := json.Marshal(msg)
					if err != nil {
						sourceArtifact.Println(string(b))
					}
				}
			} else {
				sourceArtifact.Println(line)
			}
		}

		errc <- nil
	}()

	return out, errc
}

func (w *Workflow) ProcessDestinationConnectorOutput(ctx context.Context, in <-chan interface{}, run *RunWrapper, attempt int32) <-chan error {
	errc := make(chan error, 1)

	artifactory := cosmos.ArtifactoryFromContext(ctx)
	destinationArtifact, err := w.App.GetArtifactRef(artifactory, cosmos.ArtifactDestination, attempt)
	if err != nil {
		errc <- err
		return errc
	}

	for line := range in {
		if msg, ok := line.(*cosmos.Message); ok {
			if msg.Type == cosmos.MessageTypeState {
				run.Lock()
				run.Sync.State = msg.State.Data
				run.Unlock()
			}
		} else {
			destinationArtifact.Println(line)
		}
	}

	errc <- nil
	return errc
}

func (w *Workflow) ProcessNormalizationOutput(ctx context.Context, in <-chan interface{}, attempt int32) <-chan error {
	errc := make(chan error, 1)

	artifactory := cosmos.ArtifactoryFromContext(ctx)
	normalizationArtifact, err := w.App.GetArtifactRef(artifactory, cosmos.ArtifactNormalization, attempt)
	if err != nil {
		errc <- err
		return errc
	}

	for line := range in {
		if msg, ok := line.(*cosmos.Message); ok {
			_ = msg
		} else {
			normalizationArtifact.Println(line)
		}
	}

	errc <- nil
	return errc
}

func sendMsgOnChannel(ctx context.Context, msg *cosmos.Message, ch chan<- *cosmos.Message) error {
	select {
	case ch <- msg:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}
