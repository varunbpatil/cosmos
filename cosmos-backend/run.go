package cosmos

import (
	"context"
	"errors"
	"time"
)

const (
	RunStatusQueued   = "queued"
	RunStatusRunning  = "running"
	RunStatusSuccess  = "success"
	RunStatusFailed   = "failed"
	RunStatusCanceled = "canceled"
	RunStatusWiped    = "wiped"
)

var (
	ErrNoPrevRun = errors.New("no previous run of this sync")
)

type Run struct {
	ID                 int        `json:"id"`
	SyncID             int        `json:"syncID"`
	ExecutionDate      time.Time  `json:"executionDate"`
	Status             string     `json:"status"`
	Stats              RunStats   `json:"stats"`
	Options            RunOptions `json:"options"`
	TemporalWorkflowID string     `json:"temporalWorkflowID"`
	TemporalRunID      string     `json:"temporalRunID"`
	Sync               *Sync      `json:"sync"`
}

func (r *Run) IsTerminalState() bool {
	switch r.Status {
	case RunStatusSuccess, RunStatusFailed, RunStatusCanceled, RunStatusWiped:
		return true
	default:
		return false
	}
}

type RunStats struct {
	NumRecords     uint64    `json:"numRecords"`
	ExecutionStart time.Time `json:"executionStart"`
	ExecutionEnd   time.Time `json:"executionEnd"`
}

type RunOptions struct {
	WipeDestination bool `json:"wipeDestination"`
}

type RunUpdate struct {
	Status             *string     `json:"status"`
	Retries            *int        `json:"retries"`
	NumRecords         *uint64     `json:"numRecords"`
	ExecutionStart     *time.Time  `json:"executionStart"`
	ExecutionEnd       *time.Time  `json:"executionEnd"`
	Options            *RunOptions `json:"options"`
	TemporalWorkflowID *string     `json:"temporalWorkflowID"`
	TemporalRunID      *string     `json:"temporalRunID"`
}

type RunFilter struct {
	ID        *int     `json:"id"`
	SyncID    *int     `json:"syncID"`
	Status    []string `json:"status"`
	DateRange []string `json:"dateRange"`

	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type RunService interface {
	FindRunByID(ctx context.Context, id int) (*Run, error)
	FindRuns(ctx context.Context, filter RunFilter) ([]*Run, int, error)
	CreateRun(ctx context.Context, run *Run) error
	UpdateRun(ctx context.Context, id int, run *Run) error
	GetLastRunForSyncID(ctx context.Context, syncID int) (*Run, error)
}

func (a *App) UpdateRun(ctx context.Context, id int, upd *RunUpdate) (*Run, error) {
	run, err := a.FindRunByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields if set.
	if v := upd.Status; v != nil {
		run.Status = *v
	}
	if v := upd.NumRecords; v != nil {
		run.Stats.NumRecords = *v
	}
	if v := upd.ExecutionStart; v != nil {
		run.Stats.ExecutionStart = *v
	}
	if v := upd.ExecutionEnd; v != nil {
		run.Stats.ExecutionEnd = *v
	}
	if v := upd.Options; v != nil {
		run.Options = *v
	}
	if v := upd.TemporalWorkflowID; v != nil {
		run.TemporalWorkflowID = *v
	}
	if v := upd.TemporalRunID; v != nil {
		run.TemporalRunID = *v
	}

	if err := a.DBService.UpdateRun(ctx, id, run); err != nil {
		return nil, err
	}

	return run, nil
}
