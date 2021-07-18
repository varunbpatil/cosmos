package temporal

import (
	"context"
	"cosmos"
	"log"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"go.temporal.io/sdk/client"
)

var _ cosmos.WorkerService = (*Worker)(nil)

type Worker struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	client.Client
	*cosmos.App
}

func NewWorker() *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (w *Worker) Open() error {
	w.wg.Add(1)
	go w.WorkerLoop(w.ctx)
	return nil
}

func (w *Worker) Close() error {
	w.cancel()
	w.wg.Wait()
	return nil
}

func (w *Worker) CancelRun(ctx context.Context, runID int) error {
	run, err := w.App.FindRunByID(ctx, runID)
	if err != nil {
		return err
	}
	return w.Client.CancelWorkflow(ctx, run.TemporalWorkflowID, run.TemporalRunID)
}

func recoverFromPanic() {
	if err := recover(); err != nil {
		log.Printf("worker panic: %s", err)
		debug.PrintStack()
	}
}

func (w *Worker) WorkerLoop(ctx context.Context) {
	defer w.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
			w.DoWork()
		}
	}
}

func (w *Worker) DoWork() {
	defer recoverFromPanic()

	runs, _, err := w.App.FindRuns(w.ctx, cosmos.RunFilter{Status: []string{cosmos.RunStatusQueued}})
	if err != nil {
		log.Printf("worker err: %s", err)
		return
	}

	for _, run := range runs {
		options := client.StartWorkflowOptions{ID: strconv.Itoa(run.SyncID), TaskQueue: cosmos.TemporalTaskQueue}

		// If there is already a workflow running, ExecuteWorkflow() will simply return its run id without creating a new one.
		wr, err := w.Client.ExecuteWorkflow(w.ctx, options, NewWorkflow().IngestionWorkflow, run.ID)
		if err != nil {
			log.Printf("worker failed to start temporal workflow. err: %s", err)
			continue
		}
		temporalWorkflowID := wr.GetID()
		temporalRunID := wr.GetRunID()

		// Don't set the status to "running" here. It will be set in the workflow.
		// Even if this UpdateRun fails, ExecuteWorkflow() will return the same run id next time around.
		run, err = w.App.UpdateRun(
			w.ctx,
			run.ID,
			&cosmos.RunUpdate{
				TemporalWorkflowID: &temporalWorkflowID,
				TemporalRunID:      &temporalRunID,
			},
		)
		if err != nil {
			log.Printf("worker err: %s", err)
		}
	}
}
