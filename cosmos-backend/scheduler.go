package cosmos

type SchedulerService interface {
	Schedule(syncID *int, runOptions *RunOptions) error
}
