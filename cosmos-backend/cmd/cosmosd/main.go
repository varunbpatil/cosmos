package main

import (
	"cosmos"
	"cosmos/docker"
	"cosmos/filesystem"
	"cosmos/http"
	"cosmos/jsonschema"
	"cosmos/postgres"
	"cosmos/scheduler"
	"cosmos/temporal"
	"cosmos/zap"
	"fmt"
	"log"
	"os"
	"os/signal"

	"go.temporal.io/sdk/client"
)

type OpenCloser interface {
	Open() error
	Close() error
}

// Main represents the application.
type Main struct {
	db         OpenCloser
	httpServer OpenCloser
	scheduler  OpenCloser
	worker     OpenCloser
	client     client.Client
}

// NewMain returns a new instance of Main.
func NewMain() *Main {
	// TODO: NewMain() should accept a config parameter
	// which contains all the configuration that has been
	// parsed from a TOML file.
	db := postgres.NewDB("postgres://postgres:password@postgresql:5432/postgres", true)

	dbService := postgres.NewDBService(db)
	messageService := jsonschema.NewMessageService()
	artifactService := filesystem.NewArtifactService()
	commandService := docker.NewCommandService()
	worker := temporal.NewWorker()
	scheduler := scheduler.NewScheduler()
	logger := zap.NewLogger()
	httpServer := http.NewServer(":5000")

	app := &cosmos.App{
		DBService:        dbService,
		CommandService:   commandService,
		MessageService:   messageService,
		ArtifactService:  artifactService,
		SchedulerService: scheduler,
		WorkerService:    worker,
		Logger:           logger,
	}
	commandService.App = app
	worker.App = app
	scheduler.App = app
	httpServer.App = app

	client, err := client.NewClient(client.Options{HostPort: "temporal:7233", Logger: logger})
	if err != nil {
		log.Fatal("Unable to create temporal client. err: " + err.Error())
	}
	worker.Client = client

	return &Main{
		db:         db,
		httpServer: httpServer,
		scheduler:  scheduler,
		worker:     worker,
		client:     client,
	}
}

func (m *Main) startup() error {
	if err := m.db.Open(); err != nil {
		return fmt.Errorf("cannot open db: %w", err)
	}
	if err := m.httpServer.Open(); err != nil {
		return fmt.Errorf("cannot start http server: %w", err)
	}
	if err := m.worker.Open(); err != nil {
		return fmt.Errorf("cannot start worker: %w", err)
	}
	if err := m.scheduler.Open(); err != nil {
		return fmt.Errorf("cannot start scheduler: %w", err)
	}

	fmt.Printf(`

	_________
	\_   ___ \   ____    ______  _____    ____    ______
	/    \  \/  /  _ \  /  ___/ /     \  /  _ \  /  ___/
	\     \____(  <_> ) \___ \ |  Y Y  \(  <_> ) \___ \
	 \______  / \____/ /____  >|__|_|  / \____/ /____  >
	        \/              \/       \/              \/

	is now accepting connections at %s

	`, "http://localhost:5000")

	return nil
}

func (m *Main) shutdown() error {
	if err := m.scheduler.Close(); err != nil {
		return err
	}
	if err := m.worker.Close(); err != nil {
		return err
	}
	if err := m.httpServer.Close(); err != nil {
		return err
	}
	if err := m.db.Close(); err != nil {
		return err
	}
	m.client.Close()
	return nil
}

func main() {
	// Setup SIGINT (Ctrl-C) handler.
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, os.Interrupt)

	m := NewMain()

	// Start the application.
	if err := m.startup(); err != nil {
		m.shutdown()
		log.Fatalf("cosmos: application startup failed: %s", err)
	}

	// Wait for Ctrl-C.
	<-interruptChannel

	// Shutdown the application.
	if err := m.shutdown(); err != nil {
		log.Fatalf("cosmos: application shutdown failed: %s", err)
	}
}
