package cosmos

import (
	"context"
	"time"
)

// EndPoint represents a "Connector" that has been configured for a particular endpoint.
type Endpoint struct {
	ID             int        `json:"id"`
	Name           string     `json:"name"`
	Type           string     `json:"type"`
	ConnectorID    int        `json:"connectorID"`
	Config         Form       `json:"config"`
	Catalog        Message    `json:"catalog"`
	LastDiscovered time.Time  `json:"lastDiscovered"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	Connector      *Connector `json:"connector"`
}

func (e *Endpoint) Validate() error {
	if e.Name == "" {
		return Errorf(EINVALID, "Endpoint name required")
	} else if e.Type != ConnectorTypeSource && e.Type != ConnectorTypeDestination {
		return Errorf(EINVALID, "Endpoint type must be one of 'source' or 'destination'")
	} else if e.ConnectorID == 0 {
		return Errorf(EINVALID, "A connector must be selected")
	}
	return nil
}

type EndpointUpdate struct {
	Name   *string `json:"name"`
	Config *Form   `json:"config"`
}

type EndpointFilter struct {
	ID   *int    `json:"id"`
	Name *string `json:"name"`
	Type *string `json:"type"`

	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type EndpointService interface {
	FindEndpointByID(ctx context.Context, id int) (*Endpoint, error)
	FindEndpoints(ctx context.Context, filter EndpointFilter) ([]*Endpoint, int, error)
	CreateEndpoint(ctx context.Context, endpoint *Endpoint) error
	UpdateEndpoint(ctx context.Context, id int, endpoint *Endpoint) error
	DeleteEndpoint(ctx context.Context, id int) error
}

func (a *App) CreateEndpoint(ctx context.Context, endpoint *Endpoint) error {
	// Perform basic field validation.
	if err := endpoint.Validate(); err != nil {
		return err
	}

	connector, err := a.FindConnectorByID(ctx, endpoint.ConnectorID)
	if err != nil {
		return err
	}

	config := endpoint.Config.ToSpec()

	if err := a.Validate(ctx, config, &connector.Spec); err != nil {
		return err
	}

	msg, err := a.Check(ctx, connector, config)
	if err != nil {
		return err
	}

	if msg.ConnectionStatus.Status != ConnectionStatusSucceeded {
		var connectionError string
		if msg.ConnectionStatus.Message != nil {
			connectionError = *msg.ConnectionStatus.Message
		}
		return Errorf(EINVALID, "The configuration provided is invalid. %s", connectionError)
	}

	msg, err = a.Discover(ctx, connector, config)
	if err != nil {
		return err
	}
	endpoint.Catalog = *msg

	endpoint.Connector = connector

	return a.DBService.CreateEndpoint(ctx, endpoint)
}

func (a *App) UpdateEndpoint(ctx context.Context, id int, upd *EndpointUpdate) (*Endpoint, error) {
	// Fetch the current endpoint object from the database.
	endpoint, err := a.FindEndpointByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields if set.
	if v := upd.Name; v != nil {
		endpoint.Name = *v
	}
	if v := upd.Config; v != nil {
		endpoint.Config = *v
	}

	// Perform basic validation to make sure that the updates are correct.
	if err := endpoint.Validate(); err != nil {
		return nil, err
	}

	config := endpoint.Config.ToSpec()

	if err := a.Validate(ctx, config, &endpoint.Connector.Spec); err != nil {
		return nil, err
	}

	msg, err := a.Check(ctx, endpoint.Connector, config)
	if err != nil {
		return nil, err
	}

	if msg.ConnectionStatus.Status != ConnectionStatusSucceeded {
		var connectionError string
		if msg.ConnectionStatus.Message != nil {
			connectionError = *msg.ConnectionStatus.Message
		}
		return nil, Errorf(EINVALID, "The configuration provided is invalid. %s", connectionError)
	}

	if err := a.DBService.UpdateEndpoint(ctx, id, endpoint); err != nil {
		return nil, err
	}

	return endpoint, nil
}

func (a *App) RediscoverEndpoint(ctx context.Context, id int) error {
	// Fetch the current endpoint object from the database.
	endpoint, err := a.FindEndpointByID(ctx, id)
	if err != nil {
		return err
	}

	config := endpoint.Config.ToSpec()

	msg, err := a.Discover(ctx, endpoint.Connector, config)
	if err != nil {
		return err
	}
	endpoint.Catalog = *msg

	endpoint.LastDiscovered = time.Now()

	if err := a.DBService.UpdateEndpoint(ctx, id, endpoint); err != nil {
		return err
	}

	return nil
}
