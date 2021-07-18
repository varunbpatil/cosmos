package postgres

import (
	"context"
	"cosmos"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4"
)

func (s *DBService) FindEndpointByID(ctx context.Context, id int) (*cosmos.Endpoint, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	return findEndpointByID(ctx, tx, id)

}

func (s *DBService) FindEndpoints(ctx context.Context, filter cosmos.EndpointFilter) ([]*cosmos.Endpoint, int, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback(ctx)
	return findEndpoints(ctx, tx, filter)
}

func (s *DBService) CreateEndpoint(ctx context.Context, endpoint *cosmos.Endpoint) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := createEndpoint(ctx, tx, endpoint); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *DBService) UpdateEndpoint(ctx context.Context, id int, endpoint *cosmos.Endpoint) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := updateEndpoint(ctx, tx, id, endpoint); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *DBService) DeleteEndpoint(ctx context.Context, id int) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := deleteEndpoint(ctx, tx, id); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func findEndpointByID(ctx context.Context, tx *Tx, id int) (*cosmos.Endpoint, error) {
	endpoints, totalEndpoints, err := findEndpoints(ctx, tx, cosmos.EndpointFilter{ID: &id})
	if err != nil {
		return nil, err
	} else if totalEndpoints == 0 {
		return nil, cosmos.Errorf(cosmos.ENOTFOUND, "Endpoint not found")
	}
	return endpoints[0], nil
}

func findEndpoints(ctx context.Context, tx *Tx, filter cosmos.EndpointFilter) ([]*cosmos.Endpoint, int, error) {
	// Build the WHERE clause.
	where, args, i := []string{"1 = 1"}, []interface{}{}, 1
	if v := filter.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", i)), append(args, *v)
		i++
	}
	if v := filter.Name; v != nil {
		where, args = append(where, fmt.Sprintf("name = $%d", i)), append(args, *v)
		i++
	}
	if v := filter.Type; v != nil {
		where, args = append(where, fmt.Sprintf("type = $%d", i)), append(args, *v)
		i++
	}

	rows, err := tx.Query(ctx, `
		SELECT
			id,
			name,
			type,
			connector_id,
			config,
			catalog,
			last_discovered,
			created_at,
			updated_at,
			COUNT(*) OVER()
		FROM endpoints
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY id ASC
		`+FormatLimitOffset(filter.Limit, filter.Offset),
		args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	// Iterate over the returned rows and deserialize into cosmos.Endpoint objects.
	endpoints := []*cosmos.Endpoint{}
	totalEndpoints := 0
	for rows.Next() {
		var endpoint cosmos.Endpoint

		if err := rows.Scan(
			&endpoint.ID,
			&endpoint.Name,
			&endpoint.Type,
			&endpoint.ConnectorID,
			(*Form)(&endpoint.Config),
			(*Message)(&endpoint.Catalog),
			(*NullTime)(&endpoint.LastDiscovered),
			(*NullTime)(&endpoint.CreatedAt),
			(*NullTime)(&endpoint.UpdatedAt),
			&totalEndpoints,
		); err != nil {
			return nil, 0, err
		}

		endpoints = append(endpoints, &endpoint)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// associate each endpoint with its connector.
	for _, endpoint := range endpoints {
		if err := attachConnector(ctx, tx, endpoint); err != nil {
			return nil, 0, err
		}
	}

	return endpoints, totalEndpoints, nil
}

func createEndpoint(ctx context.Context, tx *Tx, endpoint *cosmos.Endpoint) error {
	// Set timestamps to current time.
	endpoint.CreatedAt = tx.now
	endpoint.UpdatedAt = tx.now
	endpoint.LastDiscovered = tx.now

	// Insert endpoint into database.
	err := tx.QueryRow(ctx, `
		INSERT INTO endpoints (
			name,
			type,
			connector_id,
			config,
			catalog,
			last_discovered,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`,
		endpoint.Name,
		endpoint.Type,
		endpoint.ConnectorID,
		(*Form)(&endpoint.Config),
		(*Message)(&endpoint.Catalog),
		(*NullTime)(&endpoint.LastDiscovered),
		(*NullTime)(&endpoint.CreatedAt),
		(*NullTime)(&endpoint.UpdatedAt),
	).Scan(&endpoint.ID)

	if err != nil {
		return FormatError(err)
	}

	return nil
}

func updateEndpoint(ctx context.Context, tx *Tx, id int, endpoint *cosmos.Endpoint) error {
	endpoint.UpdatedAt = tx.now

	// Execute update query.
	if _, err := tx.Exec(ctx, `
		UPDATE endpoints
		SET
			name = $1,
			type = $2,
			connector_id = $3,
			config = $4,
			catalog = $5,
			last_discovered = $6,
			updated_at = $7
		WHERE
			id = $8
	`,
		endpoint.Name,
		endpoint.Type,
		endpoint.ConnectorID,
		(*Form)(&endpoint.Config),
		(*Message)(&endpoint.Catalog),
		(*NullTime)(&endpoint.LastDiscovered),
		(*NullTime)(&endpoint.UpdatedAt),
		id,
	); err != nil {
		return FormatError(err)
	}

	return nil
}

func deleteEndpoint(ctx context.Context, tx *Tx, id int) error {
	// Verify that the endpoint object exists.
	if _, err := findEndpointByID(ctx, tx, id); err != nil {
		return err
	}

	// Remove endpoint from database.
	if _, err := tx.Exec(ctx, `
		DELETE FROM endpoints
		WHERE id = $1
	`,
		id,
	); err != nil {
		return err
	}

	return nil
}

func attachConnector(ctx context.Context, tx *Tx, endpoint *cosmos.Endpoint) error {
	connector, err := findConnectorByID(ctx, tx, endpoint.ConnectorID)
	if err != nil {
		return err
	}
	endpoint.Connector = connector
	return nil
}
