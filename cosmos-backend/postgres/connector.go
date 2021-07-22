package postgres

import (
	"context"
	"cosmos"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4"
)

func (s *DBService) FindConnectorByID(ctx context.Context, id int) (*cosmos.Connector, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	return findConnectorByID(ctx, tx, id)
}

func (s *DBService) FindConnectors(ctx context.Context, filter cosmos.ConnectorFilter) ([]*cosmos.Connector, int, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback(ctx)
	return findConnectors(ctx, tx, filter)
}

func (s *DBService) CreateConnector(ctx context.Context, connector *cosmos.Connector) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := createConnector(ctx, tx, connector); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *DBService) UpdateConnector(ctx context.Context, id int, connector *cosmos.Connector) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := updateConnector(ctx, tx, id, connector); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *DBService) DeleteConnector(ctx context.Context, id int) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := deleteConnector(ctx, tx, id); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func findConnectorByID(ctx context.Context, tx *Tx, id int) (*cosmos.Connector, error) {
	connectors, totalConnectors, err := findConnectors(ctx, tx, cosmos.ConnectorFilter{ID: &id})
	if err != nil {
		return nil, err
	} else if totalConnectors == 0 {
		return nil, cosmos.Errorf(cosmos.ENOTFOUND, "Connector not found")
	}
	return connectors[0], nil
}

func findConnectors(ctx context.Context, tx *Tx, filter cosmos.ConnectorFilter) ([]*cosmos.Connector, int, error) {
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
			docker_image_name,
			docker_image_tag,
			destination_type,
			spec,
			created_at,
			updated_at,
			COUNT(*) OVER()
		FROM connectors
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY name ASC
		`+FormatLimitOffset(filter.Limit, filter.Offset),
		args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	// Iterate over the returned rows and deserialize into cosmos.Connector objects.
	connectors := []*cosmos.Connector{}
	totalConnectors := 0
	for rows.Next() {
		var connector cosmos.Connector
		if err := rows.Scan(
			&connector.ID,
			&connector.Name,
			&connector.Type,
			&connector.DockerImageName,
			&connector.DockerImageTag,
			&connector.DestinationType,
			(*Message)(&connector.Spec),
			(*NullTime)(&connector.CreatedAt),
			(*NullTime)(&connector.UpdatedAt),
			&totalConnectors,
		); err != nil {
			return nil, 0, err
		}
		connectors = append(connectors, &connector)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return connectors, totalConnectors, nil
}

func createConnector(ctx context.Context, tx *Tx, connector *cosmos.Connector) error {
	// Set timestamps to current time.
	connector.CreatedAt = tx.now
	connector.UpdatedAt = connector.CreatedAt

	// Insert connector into database.
	err := tx.QueryRow(ctx, `
		INSERT INTO connectors (
			name,
			type,
			docker_image_name,
			docker_image_tag,
			destination_type,
			spec,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`,
		connector.Name,
		connector.Type,
		connector.DockerImageName,
		connector.DockerImageTag,
		connector.DestinationType,
		(*Message)(&connector.Spec),
		(*NullTime)(&connector.CreatedAt),
		(*NullTime)(&connector.UpdatedAt),
	).Scan(&connector.ID)

	if err != nil {
		return FormatError(err)
	}

	return nil
}

func updateConnector(ctx context.Context, tx *Tx, id int, connector *cosmos.Connector) error {
	connector.UpdatedAt = tx.now

	// Execute update query.
	if _, err := tx.Exec(ctx, `
		UPDATE connectors
		SET
			name = $1,
			type = $2,
			docker_image_name = $3,
			docker_image_tag = $4,
			destination_type = $5,
			spec = $6,
			updated_at = $7
		WHERE
			id = $8
	`,
		connector.Name,
		connector.Type,
		connector.DockerImageName,
		connector.DockerImageTag,
		connector.DestinationType,
		(*Message)(&connector.Spec),
		(*NullTime)(&connector.UpdatedAt),
		id,
	); err != nil {
		return FormatError(err)
	}

	return nil
}

func deleteConnector(ctx context.Context, tx *Tx, id int) error {
	// Verify that the connector object exists.
	if _, err := findConnectorByID(ctx, tx, id); err != nil {
		return err
	}

	// Remove connector from database.
	if _, err := tx.Exec(ctx, `
		DELETE FROM connectors
		WHERE id = $1
	`,
		id,
	); err != nil {
		return err
	}

	return nil
}
