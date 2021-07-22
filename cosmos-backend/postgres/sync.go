package postgres

import (
	"context"
	"cosmos"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4"
)

func (s *DBService) FindSyncByID(ctx context.Context, id int) (*cosmos.Sync, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	return findSyncByID(ctx, tx, id)
}

func (s *DBService) FindSyncs(ctx context.Context, filter cosmos.SyncFilter) ([]*cosmos.Sync, int, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback(ctx)
	return findSyncs(ctx, tx, filter)
}

func (s *DBService) CreateSync(ctx context.Context, sync *cosmos.Sync) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := createSync(ctx, tx, sync); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *DBService) UpdateSync(ctx context.Context, id int, sync *cosmos.Sync) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := updateSync(ctx, tx, id, sync); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *DBService) DeleteSync(ctx context.Context, id int) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := deleteSync(ctx, tx, id); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func findSyncByID(ctx context.Context, tx *Tx, id int) (*cosmos.Sync, error) {
	syncs, totalSyncs, err := findSyncs(ctx, tx, cosmos.SyncFilter{ID: &id})
	if err != nil {
		return nil, err
	} else if totalSyncs == 0 {
		return nil, cosmos.Errorf(cosmos.ENOTFOUND, "Sync not found")
	}
	return syncs[0], nil
}

func findSyncs(ctx context.Context, tx *Tx, filter cosmos.SyncFilter) ([]*cosmos.Sync, int, error) {
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

	rows, err := tx.Query(ctx, `
		SELECT
			id,
			name,
			source_endpoint_id,
			destination_endpoint_id,
			schedule_interval,
			enabled,
			state,
			config,
			configured_catalog,
			created_at,
			updated_at,
			COUNT(*) OVER()
		FROM syncs
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY name ASC
		`+FormatLimitOffset(filter.Limit, filter.Offset),
		args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	// Iterate over the returned rows and deserialize into cosmos.Sync objects.
	syncs := []*cosmos.Sync{}
	totalSyncs := 0
	for rows.Next() {
		var sync cosmos.Sync

		if err := rows.Scan(
			&sync.ID,
			&sync.Name,
			&sync.SourceEndpointID,
			&sync.DestinationEndpointID,
			&sync.ScheduleInterval,
			&sync.Enabled,
			(*Map)(&sync.State),
			(*Form)(&sync.Config),
			(*Message)(&sync.ConfiguredCatalog),
			(*NullTime)(&sync.CreatedAt),
			(*NullTime)(&sync.UpdatedAt),
			&totalSyncs,
		); err != nil {
			return nil, 0, err
		}

		syncs = append(syncs, &sync)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	for _, sync := range syncs {
		// associate each sync with its endpoints.
		if err := attachEndpoints(ctx, tx, sync); err != nil {
			return nil, 0, err
		}

		// associate each sync with its last run.
		if err := attachLastRun(ctx, tx, sync); err != nil {
			return nil, 0, err
		}

		// associate each sync with its last successful run.
		if err := attachLastSuccessfulRun(ctx, tx, sync); err != nil {
			return nil, 0, err
		}
	}

	return syncs, totalSyncs, nil
}

func createSync(ctx context.Context, tx *Tx, sync *cosmos.Sync) error {
	// Set timestamps to current time.
	sync.CreatedAt = tx.now
	sync.UpdatedAt = sync.CreatedAt

	// Insert sync into database.
	err := tx.QueryRow(ctx, `
		INSERT INTO syncs (
			name,
			source_endpoint_id,
			destination_endpoint_id,
			schedule_interval,
			enabled,
			state,
			config,
			configured_catalog,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`,
		sync.Name,
		sync.SourceEndpointID,
		sync.DestinationEndpointID,
		sync.ScheduleInterval,
		sync.Enabled,
		(*Map)(&sync.State),
		(*Form)(&sync.Config),
		(*Message)(&sync.ConfiguredCatalog),
		(*NullTime)(&sync.CreatedAt),
		(*NullTime)(&sync.UpdatedAt),
	).Scan(&sync.ID)

	if err != nil {
		return FormatError(err)
	}

	return nil
}

func updateSync(ctx context.Context, tx *Tx, id int, sync *cosmos.Sync) error {
	sync.UpdatedAt = tx.now

	// Execute update query.
	if _, err := tx.Exec(ctx, `
		UPDATE syncs
		SET
			name = $1,
			source_endpoint_id = $2,
			destination_endpoint_id = $3,
			schedule_interval = $4,
			enabled = $5,
			state = $6,
			config = $7,
			configured_catalog = $8,
			updated_at = $9
		WHERE
			id = $10
	`,
		sync.Name,
		sync.SourceEndpointID,
		sync.DestinationEndpointID,
		sync.ScheduleInterval,
		sync.Enabled,
		(*Map)(&sync.State),
		(*Form)(&sync.Config),
		(*Message)(&sync.ConfiguredCatalog),
		(*NullTime)(&sync.UpdatedAt),
		id,
	); err != nil {
		return FormatError(err)
	}

	return nil
}

func deleteSync(ctx context.Context, tx *Tx, id int) error {
	// Verify that the sync object exists.
	if _, err := findSyncByID(ctx, tx, id); err != nil {
		return err
	}

	// Remove sync from database.
	if _, err := tx.Exec(ctx, `
		DELETE FROM syncs
		WHERE id = $1
	`,
		id,
	); err != nil {
		return err
	}

	return nil
}

func attachEndpoints(ctx context.Context, tx *Tx, sync *cosmos.Sync) error {
	endpoint, err := findEndpointByID(ctx, tx, sync.SourceEndpointID)
	if err != nil {
		return err
	}
	sync.SourceEndpoint = endpoint

	endpoint, err = findEndpointByID(ctx, tx, sync.DestinationEndpointID)
	if err != nil {
		return err
	}
	sync.DestinationEndpoint = endpoint

	return nil
}

func attachLastRun(ctx context.Context, tx *Tx, sync *cosmos.Sync) error {
	run, err := getLastRunForSyncID(ctx, tx, sync.ID, nil, false)
	if err != nil && !errors.Is(err, cosmos.ErrNoPrevRun) {
		return err
	}
	sync.LastRun = run
	return nil
}

func attachLastSuccessfulRun(ctx context.Context, tx *Tx, sync *cosmos.Sync) error {
	run, err := getLastRunForSyncID(ctx, tx, sync.ID, []string{cosmos.RunStatusSuccess}, false)
	if err != nil && !errors.Is(err, cosmos.ErrNoPrevRun) {
		return err
	}
	sync.LastSuccessfulRun = run
	return nil
}
