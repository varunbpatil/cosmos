package cosmos

import (
	"context"
	"time"
)

// Connector types.
const (
	ConnectorTypeSource      = "source"
	ConnectorTypeDestination = "destination"
)

var DestinationTypes = []string{
	"postgres",
	"bigquery",
	"redshift",
	"snowflake",
	"mysql",
	"other",
}

// Connector represents a source or destination connector.
type Connector struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"`
	DockerImageName string    `json:"dockerImageName"`
	DockerImageTag  string    `json:"dockerImageTag"`
	DestinationType string    `json:"destinationType"`
	Spec            Message   `json:"spec"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func (c *Connector) HasValidDestinationType() bool {
	switch c.Type {
	case ConnectorTypeSource:
		if c.DestinationType == "" {
			return true
		}
	case ConnectorTypeDestination:
		for _, t := range DestinationTypes {
			if t == c.DestinationType {
				return true
			}
		}
	}
	return false
}

// Validate performs some basic validation on the connector object during create and update.
func (c *Connector) Validate() error {
	if c.Name == "" {
		return Errorf(EINVALID, "Connector name required")
	} else if c.Type != ConnectorTypeSource && c.Type != ConnectorTypeDestination {
		return Errorf(EINVALID, "Connector type must be one of 'source' or 'destination'")
	} else if c.DockerImageName == "" || c.DockerImageTag == "" {
		return Errorf(EINVALID, "Docker image name and tag are required")
	} else if !c.HasValidDestinationType() {
		return Errorf(EINVALID, "Invalid destination type")
	}
	return nil
}

// ConnectorFilter represents a connector search filter.
type ConnectorFilter struct {
	ID   *int    `json:"id"`
	Name *string `json:"name"`
	Type *string `json:"type"`

	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// ConnectorUpdate represent connector fields that can be updated.
type ConnectorUpdate struct {
	Name            *string `json:"name"`
	DockerImageName *string `json:"dockerImageName"`
	DockerImageTag  *string `json:"dockerImageTag"`
	DestinationType *string `json:"destinationType"`
}

type ConnectorService interface {
	FindConnectorByID(ctx context.Context, id int) (*Connector, error)
	FindConnectors(ctx context.Context, filter ConnectorFilter) ([]*Connector, int, error)
	CreateConnector(ctx context.Context, connector *Connector) error
	UpdateConnector(ctx context.Context, id int, connector *Connector) error
	DeleteConnector(ctx context.Context, id int) error
}

func (a *App) CreateConnector(ctx context.Context, connector *Connector) error {
	// Perform basic field validation.
	if err := connector.Validate(); err != nil {
		return err
	}

	// Get connections specification for the connector and update it in the connector object.
	msg, err := a.Spec(ctx, connector)
	if err != nil {
		return err
	}
	connector.Spec = *msg

	return a.DBService.CreateConnector(ctx, connector)
}

func (a *App) UpdateConnector(ctx context.Context, id int, upd *ConnectorUpdate) (*Connector, error) {
	// Fetch the current connector object from the database.
	connector, err := a.FindConnectorByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields if set.
	if v := upd.Name; v != nil {
		connector.Name = *v
	}
	if v := upd.DockerImageName; v != nil {
		connector.DockerImageName = *v
	}
	if v := upd.DockerImageTag; v != nil {
		connector.DockerImageTag = *v
	}
	if v := upd.DestinationType; v != nil {
		connector.DestinationType = *v
	}

	// Perform basic validation to make sure that the updates are correct.
	if err := connector.Validate(); err != nil {
		return nil, err
	}

	// Get connections specification for the connector and update it in the connector object.
	msg, err := a.Spec(ctx, connector)
	if err != nil {
		return nil, err
	}
	connector.Spec = *msg

	if err := a.DBService.UpdateConnector(ctx, id, connector); err != nil {
		return nil, err
	}

	return connector, nil
}
