package scheduler

import (
	"context"
	"cosmos"
	"errors"
	"log"
	"runtime/debug"
	"sync"
	"time"
)

var _ cosmos.SchedulerService = (*Scheduler)(nil)

type Scheduler struct {
	ctx    context.Context
	cancel context.CancelFunc
	sync.Mutex
	sync.WaitGroup

	*cosmos.App
}

func NewScheduler() *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *Scheduler) Open() error {
	s.Add(1)
	go s.SchedulerLoop(s.ctx)
	return nil
}

func (s *Scheduler) Close() error {
	s.cancel()
	s.Wait()
	return nil
}

func recoverFromPanic() {
	if err := recover(); err != nil {
		log.Printf("scheduler panic: %s", err)
		debug.PrintStack()
	}
}

func (s *Scheduler) SchedulerLoop(ctx context.Context) {
	defer s.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
			s.Schedule(nil, &cosmos.RunOptions{})
		}
	}
}

func (s *Scheduler) Schedule(syncID *int, runOptions *cosmos.RunOptions) error {
	defer recoverFromPanic()

	s.Lock()
	defer s.Unlock()

	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	syncs, _, err := s.App.FindSyncs(ctx, cosmos.SyncFilter{ID: syncID})
	if err != nil {
		log.Printf("scheduler err: %s", err)
		return err
	}

	for _, sync := range syncs {
		run, err := s.App.GetLastRunForSyncID(ctx, sync.ID)
		if err != nil && !errors.Is(err, cosmos.ErrNoPrevRun) {
			log.Printf("scheduler err: %s", err)
			if syncID != nil {
				return err
			}
			continue
		}

		ok, err := okToSchedule(sync, run, syncID != nil)
		if !ok {
			if err != nil && syncID != nil {
				return err
			}
			continue
		}

		run = &cosmos.Run{SyncID: sync.ID, ExecutionDate: time.Now(), Options: *runOptions}
		if err := s.App.CreateRun(ctx, run); err != nil {
			log.Printf("scheduler err: %s", err)
		}
	}

	return nil
}

func okToSchedule(sync *cosmos.Sync, run *cosmos.Run, force bool) (bool, error) {
	if !sync.Enabled && !force {
		return false, cosmos.Errorf(cosmos.ECONFLICT, "Not enabled")
	}
	if run == nil {
		// No previous run.
		return true, nil
	}
	if time.Now().Sub(run.ExecutionDate) < time.Duration(sync.ScheduleInterval)*time.Minute && !force {
		return false, cosmos.Errorf(cosmos.ECONFLICT, "Interval has not elapsed")
	}
	if !run.IsTerminalState() {
		return false, cosmos.Errorf(cosmos.ECONFLICT, "A run is in progress")
	}
	return true, nil
}
