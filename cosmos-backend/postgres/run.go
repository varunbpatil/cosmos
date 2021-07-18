package postgres

import (
	"context"
	"cosmos"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4"
)

func (s *DBService) FindRunByID(ctx context.Context, id int) (*cosmos.Run, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	return findRunByID(ctx, tx, id)
}

func (s *DBService) FindRuns(ctx context.Context, filter cosmos.RunFilter) ([]*cosmos.Run, int, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback(ctx)
	return findRuns(ctx, tx, filter, true)
}

func (s *DBService) CreateRun(ctx context.Context, run *cosmos.Run) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := createRun(ctx, tx, run); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *DBService) UpdateRun(ctx context.Context, id int, run *cosmos.Run) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := updateRun(ctx, tx, id, run); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *DBService) GetLastRunForSyncID(ctx context.Context, syncID int) (*cosmos.Run, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	return getLastRunForSyncID(ctx, tx, syncID, nil, true)
}

func findRunByID(ctx context.Context, tx *Tx, id int) (*cosmos.Run, error) {
	runs, totalRuns, err := findRuns(ctx, tx, cosmos.RunFilter{ID: &id}, true)
	if err != nil {
		return nil, err
	} else if totalRuns == 0 {
		return nil, cosmos.Errorf(cosmos.ENOTFOUND, "Run not found")
	}
	return runs[0], nil
}

func findRuns(ctx context.Context, tx *Tx, filter cosmos.RunFilter, wantSync bool) ([]*cosmos.Run, int, error) {
	// Build the WHERE clause.
	where, args, i := []string{"1 = 1"}, []interface{}{}, 1
	if v := filter.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", i)), append(args, *v)
		i++
	}
	if v := filter.SyncID; v != nil {
		where, args = append(where, fmt.Sprintf("sync_id = $%d", i)), append(args, *v)
		i++
	}
	if v := filter.Status; v != nil {
		tmp := []string{}
		for _, s := range v {
			tmp = append(tmp, fmt.Sprintf("$%d", i))
			args = append(args, s)
			i++
		}
		where = append(where, fmt.Sprintf("status IN (%s)", strings.Join(tmp, ", ")))
	}
	if v := filter.DateRange; v != nil {
		where, args = append(where, fmt.Sprintf("execution_date >= $%d", i)), append(args, v[0])
		i++
		where, args = append(where, fmt.Sprintf("execution_date <= $%d", i)), append(args, v[1]+"T23:59:59Z")
		i++
	}

	rows, err := tx.Query(ctx, `
		SELECT
			id,
			sync_id,
			execution_date,
			status,
			stats,
			options,
			temporal_workflow_id,
			temporal_run_id,
			COUNT(*) OVER()
		FROM runs
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY execution_date DESC
		`+FormatLimitOffset(filter.Limit, filter.Offset),
		args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	// Iterate over the returned rows and deserialize into cosmos.Run objects.
	runs := []*cosmos.Run{}
	totalRuns := 0
	for rows.Next() {
		var run cosmos.Run

		if err := rows.Scan(
			&run.ID,
			&run.SyncID,
			(*NullTime)(&run.ExecutionDate),
			&run.Status,
			(*RunStats)(&run.Stats),
			(*RunOptions)(&run.Options),
			&run.TemporalWorkflowID,
			&run.TemporalRunID,
			&totalRuns,
		); err != nil {
			return nil, 0, err
		}

		runs = append(runs, &run)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	if wantSync {
		for _, run := range runs {
			// associate each run with its sync.
			if err := attachSync(ctx, tx, run); err != nil {
				return nil, 0, err
			}
		}
	}

	return runs, totalRuns, nil
}

func createRun(ctx context.Context, tx *Tx, run *cosmos.Run) error {
	run.Status = cosmos.RunStatusQueued
	run.TemporalWorkflowID = ""
	run.TemporalRunID = ""

	err := tx.QueryRow(ctx, `
		INSERT INTO runs (
			sync_id,
			execution_date,
			status,
			stats,
			options,
			temporal_workflow_id,
			temporal_run_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`,
		run.SyncID,
		(*NullTime)(&run.ExecutionDate),
		run.Status,
		(*RunStats)(&run.Stats),
		(*RunOptions)(&run.Options),
		run.TemporalWorkflowID,
		run.TemporalRunID,
	).Scan(&run.ID)

	if err != nil {
		return FormatError(err)
	}

	return nil
}

func updateRun(ctx context.Context, tx *Tx, id int, run *cosmos.Run) error {
	if _, err := tx.Exec(ctx, `
		UPDATE runs
		SET
			sync_id = $1,
			execution_date = $2,
			status = $3,
			stats = $4,
			options = $5,
			temporal_workflow_id = $6,
			temporal_run_id = $7
		WHERE
			id = $8
	`,
		run.SyncID,
		(*NullTime)(&run.ExecutionDate),
		run.Status,
		(*RunStats)(&run.Stats),
		(*RunOptions)(&run.Options),
		run.TemporalWorkflowID,
		run.TemporalRunID,
		id,
	); err != nil {
		return FormatError(err)
	}

	return nil
}

func getLastRunForSyncID(ctx context.Context, tx *Tx, syncID int, status []string, wantSync bool) (*cosmos.Run, error) {
	runs, totalRuns, err := findRuns(ctx, tx, cosmos.RunFilter{SyncID: &syncID, Status: status, Limit: 1}, wantSync)
	if err != nil {
		return nil, err
	} else if totalRuns == 0 {
		return nil, cosmos.ErrNoPrevRun
	}
	return runs[0], nil
}

func attachSync(ctx context.Context, tx *Tx, run *cosmos.Run) error {
	sync, err := findSyncByID(ctx, tx, run.SyncID)
	if err != nil {
		return err
	}
	run.Sync = sync
	return nil
}
