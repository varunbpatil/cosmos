package main

import (
	"cosmos"
	"cosmos/docker"
	"cosmos/filesystem"
	"cosmos/jsonschema"
	"cosmos/postgres"
	"cosmos/temporal"
	"cosmos/zap"
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	client, err := client.NewClient(client.Options{HostPort: "temporal:7233", Logger: zap.NewLogger()})
	if err != nil {
		log.Fatal("Unable to create temporal client. err: " + err.Error())
	}
	defer client.Close()

	db := postgres.NewDB("postgres://postgres:password@postgresql:5432/postgres", false)
	dbService := postgres.NewDBService(db)
	messageService := jsonschema.NewMessageService()
	artifactService := filesystem.NewArtifactService()
	commandService := docker.NewCommandService()
	workflow := temporal.NewWorkflow()
	app := &cosmos.App{
		DBService:       dbService,
		CommandService:  commandService,
		MessageService:  messageService,
		ArtifactService: artifactService,
	}
	commandService.App = app
	workflow.App = app

	if err := db.Open(); err != nil {
		log.Fatal("Unable to connect to db in temporal worker. err: " + err.Error())
	}
	defer db.Close()

	w := worker.New(client, cosmos.TemporalTaskQueue, worker.Options{})
	w.RegisterWorkflow(workflow.IngestionWorkflow)
	w.RegisterActivity(workflow.GetRun)
	w.RegisterActivity(workflow.Initialize)
	w.RegisterActivity(workflow.ReplicationActivity)
	w.RegisterActivity(workflow.NormalizationActivity)
	w.RegisterActivity(workflow.DBUpdateActivity)

	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatal("Unable to start temporal worker. err: " + err.Error())
	}
}
