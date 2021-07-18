package cosmos

import "context"

type WorkerService interface {
	CancelRun(ctx context.Context, runID int) error
}
