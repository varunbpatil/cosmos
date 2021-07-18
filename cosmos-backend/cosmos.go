package cosmos

import (
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigDefault

const (
	ScratchSpace      = "/tmp/cosmos/scratch"
	TemporalTaskQueue = "cosmos-task-queue"
)

type DBService interface {
	ConnectorService
	EndpointService
	SyncService
	RunService
}

type App struct {
	DBService
	CommandService
	MessageService
	ArtifactService
	SchedulerService
	WorkerService
	Logger
}
