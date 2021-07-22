package cosmos

import (
	"context"
	"time"
)

type Sync struct {
	ID                    int                    `json:"id"`
	Name                  string                 `json:"name"`
	SourceEndpointID      int                    `json:"sourceEndpointID"`
	DestinationEndpointID int                    `json:"destinationEndpointID"`
	ScheduleInterval      int                    `json:"scheduleInterval"`
	Enabled               bool                   `json:"enabled"`
	BasicNormalization    bool                   `json:"basicNormalization"`
	State                 map[string]interface{} `json:"state"`
	Config                Form                   `json:"config"`
	ConfiguredCatalog     Message                `json:"configuredCatalog"`
	CreatedAt             time.Time              `json:"createdAt"`
	UpdatedAt             time.Time              `json:"updatedAt"`
	SourceEndpoint        *Endpoint              `json:"sourceEndpoint"`
	DestinationEndpoint   *Endpoint              `json:"destinationEndpoint"`
	LastRun               *Run                   `json:"lastRun"`
	LastSuccessfulRun     *Run                   `json:"lastSuccessfulRun"`
}

func (s *Sync) Validate() error {
	if s.Name == "" {
		return Errorf(EINVALID, "Sync name required")
	} else if s.SourceEndpointID == 0 {
		return Errorf(EINVALID, "A source endpoint must be selected")
	} else if s.DestinationEndpointID == 0 {
		return Errorf(EINVALID, "A destination endpoint must be selected")
	} else if s.ScheduleInterval < 0 {
		return Errorf(EINVALID, "Schedule interval must be greater than or equal to 0")
	}
	return nil
}

type SyncUpdate struct {
	Name               *string                 `json:"name"`
	Config             *Form                   `json:"config"`
	ScheduleInterval   *int                    `json:"scheduleInterval"`
	Enabled            *bool                   `json:"enabled"`
	BasicNormalization *bool                   `json:"basicNormalization"`
	State              *map[string]interface{} `json:"state"`
}

type SyncFilter struct {
	ID   *int    `json:"id"`
	Name *string `json:"name"`

	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type SyncService interface {
	FindSyncByID(ctx context.Context, id int) (*Sync, error)
	FindSyncs(ctx context.Context, filter SyncFilter) ([]*Sync, int, error)
	CreateSync(ctx context.Context, sync *Sync) error
	UpdateSync(ctx context.Context, id int, sync *Sync) error
	DeleteSync(ctx context.Context, id int) error
}

func (a *App) CreateSync(ctx context.Context, sync *Sync) error {
	// Perform basic field validation.
	if err := sync.Validate(); err != nil {
		return err
	}

	sync.Enabled = false

	config, err := json.Marshal(sync.Config.ToConfiguredCatalog())
	if err != nil {
		return err
	}
	msg, err := a.CreateMessage(ctx, config)
	if err != nil {
		return err
	}
	sync.ConfiguredCatalog = *msg

	return a.DBService.CreateSync(ctx, sync)
}

func (a *App) UpdateSync(ctx context.Context, id int, upd *SyncUpdate) (*Sync, error) {
	// Fetch the current sync object from the database.
	sync, err := a.FindSyncByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields if set.
	if v := upd.Name; v != nil {
		sync.Name = *v
	}
	if v := upd.ScheduleInterval; v != nil {
		sync.ScheduleInterval = *v
	}
	if v := upd.Enabled; v != nil {
		sync.Enabled = *v
	}
	if v := upd.BasicNormalization; v != nil {
		sync.BasicNormalization = *v
	}
	if v := upd.Config; v != nil {
		sync.Config = *v
	}
	if v := upd.State; v != nil {
		sync.State = *v
		if len(sync.State) == 0 {
			sync.State = nil
		}
	}

	// Perform basic validation to make sure that the updates are correct.
	if err := sync.Validate(); err != nil {
		return nil, err
	}

	config, err := json.Marshal(sync.Config.ToConfiguredCatalog())
	if err != nil {
		return nil, err
	}
	msg, err := a.CreateMessage(ctx, config)
	if err != nil {
		return nil, err
	}
	sync.ConfiguredCatalog = *msg

	if err := a.DBService.UpdateSync(ctx, id, sync); err != nil {
		return nil, err
	}

	return sync, nil
}
