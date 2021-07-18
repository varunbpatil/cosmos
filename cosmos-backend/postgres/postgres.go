package postgres

import (
	"context"
	"cosmos"
	"database/sql/driver"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigDefault

// DB represents the database connection.
type DB struct {
	db             *pgxpool.Pool
	url            string
	now            func() time.Time
	should_migrate bool
}

// NewDB returns a new instance of DB associated with the given URL.
func NewDB(url string, should_migrate bool) *DB {
	return &DB{
		url:            url,
		now:            time.Now,
		should_migrate: should_migrate,
	}
}

// Open opens the database connection.
func (db *DB) Open() (err error) {
	if db.url == "" {
		return fmt.Errorf("DB url required")
	}

	// Connect to the database.
	if db.db, err = pgxpool.Connect(context.Background(), db.url); err != nil {
		return err
	}

	// Perform database migrations.
	if err := db.migrate(); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	// Close database connection.
	if db.db != nil {
		db.db.Close()
	}
	return nil
}

//go:embed migrations/*.sql
var migrationsFS embed.FS

// migrate performs database migrations.
//
// Migration files are embedded in the postgres/migrations folder and are executed
// in lexicographic order.
//
// Once a migration is run, its name is stored in the 'migrations' table so that
// it is not re-executed.
func (db *DB) migrate() error {
	if !db.should_migrate {
		return nil
	}

	// Create the 'migrations' table if it doesn't already exist.
	if _, err := db.db.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS migrations (name TEXT PRIMARY KEY);`); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Read the migration files from the embedded filesystem.
	// Sort the migrations in lexicographic order.
	names, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)

	// Execute all migrations.
	if err := db.migrateFiles(names); err != nil {
		return fmt.Errorf("migration error: err=%w", err)
	}

	return nil
}

func (db *DB) migrateFiles(names []string) error {
	tx, err := db.db.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	// Acquire a table-level lock which will be release when the transaction ends.
	if _, err := tx.Exec(context.Background(), `LOCK TABLE migrations`); err != nil {
		return err
	}

	for _, name := range names {
		// Make sure that the migration hasn't already been run.
		var n int
		if err := tx.QueryRow(context.Background(), `SELECT COUNT(*) FROM migrations WHERE name = $1`, name).Scan(&n); err != nil {
			return err
		} else if n != 0 {
			// Migration has already been run. Nothing more to do.
			return nil
		}

		// Read and execute the migration file.
		if buf, err := fs.ReadFile(migrationsFS, name); err != nil {
			return err
		} else if _, err := tx.Exec(context.Background(), string(buf)); err != nil {
			return err
		}

		// Insert a record into the migrations table to prevent re-running the migration.
		if _, err := tx.Exec(context.Background(), `INSERT INTO migrations (name) VALUES($1)`, name); err != nil {
			return err
		}
	}

	return tx.Commit(context.Background())
}

// Tx wraps the sql Tx object to provide a transaction start timestamp.
// Useful when populating CreatedAt and UpdatedAt columns.
type Tx struct {
	pgx.Tx
	now time.Time
}

// BeginTx returns a wrapped sql Tx object containing the transaction start timestamp.
func (db *DB) BeginTx(ctx context.Context, opts pgx.TxOptions) (*Tx, error) {
	tx, err := db.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &Tx{
		Tx:  tx,
		now: db.now().UTC().Truncate(time.Second),
	}, nil
}

// NullTime represents a helper wrapper for time.Time.
// It automatically converts to/from RFC 3339 format.
// Also supports NULL for zero time.
type NullTime time.Time

// Scan reads a RFC 3339 formatted time (or NULL) from the database.
func (n *NullTime) Scan(value interface{}) error {
	if value == nil {
		*n = NullTime(time.Time{})
		return nil
	} else if s, ok := value.(string); ok {
		if t, err := time.Parse(time.RFC3339, s); err != nil {
			return fmt.Errorf("NullTime: cannot scan from string value %s", s)
		} else {
			*n = NullTime(t)
			return nil
		}
	}

	return fmt.Errorf("NullTime: cannot scan value of type %T", value)
}

// Value writes a RFC 3339 formatted time (or NULL) to the database.
func (n *NullTime) Value() (driver.Value, error) {
	if n == nil || time.Time(*n).IsZero() {
		return nil, nil
	}

	return time.Time(*n).UTC().Format(time.RFC3339), nil
}

func unmarshal(value, target interface{}) error {
	if s, ok := value.(string); ok {
		if err := json.Unmarshal([]byte(s), target); err != nil {
			return fmt.Errorf("postgres: cannot scan from string value %s", s)
		}
		return nil
	}
	return fmt.Errorf("postgres: cannot scan value of type %T", value)
}

func marshal(value interface{}) (interface{}, error) {
	out, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("postgres: cannot marshal value")
	}
	return string(out), nil
}

// Message represents a helper wrapper for cosmos.Message.
// It automatically converts to/from string.
type Message cosmos.Message

func (m *Message) Scan(value interface{}) error {
	return unmarshal(value, m)
}

func (m *Message) Value() (driver.Value, error) {
	return marshal(m)
}

// Form represents a helper wrapper for cosmos.Form.
// It automatically converts to/from string.
type Form cosmos.Form

func (f *Form) Scan(value interface{}) error {
	return unmarshal(value, f)
}

func (f *Form) Value() (driver.Value, error) {
	return marshal(f)
}

// RunStats represents a helper wrapper for cosmos.RunStats.
// It automatically converts to/from string.
type RunStats cosmos.RunStats

func (r *RunStats) Scan(value interface{}) error {
	return unmarshal(value, r)
}

func (r *RunStats) Value() (driver.Value, error) {
	return marshal(r)
}

// RunOptions represents a helper wrapper for cosmos.RunOptions.
// It automatically converts to/from string.
type RunOptions cosmos.RunOptions

func (r *RunOptions) Scan(value interface{}) error {
	return unmarshal(value, r)
}

func (r *RunOptions) Value() (driver.Value, error) {
	return marshal(r)
}

// Map represents a helper wrapper for map[string]interface{}.
// It automatically converts to/from string.
type Map map[string]interface{}

func (m *Map) Scan(value interface{}) error {
	return unmarshal(value, m)
}

func (m *Map) Value() (driver.Value, error) {
	return marshal(m)
}

// FormatLimitOffset returns a formatted string containing the LIMIT and OFFSET.
func FormatLimitOffset(limit, offset int) string {
	if limit > 0 && offset > 0 {
		return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
	} else if limit > 0 {
		return fmt.Sprintf("LIMIT %d", limit)
	} else if offset > 0 {
		return fmt.Sprintf("OFFSET %d", offset)
	}
	return ""
}

// FormatError converts the error into proper application errors (cosmos.Error) where appropriate.
func FormatError(err error) error {
	errStr := err.Error()
	if strings.Contains(errStr, `violates unique constraint "connectors_name_type_key"`) {
		return cosmos.Errorf(cosmos.ECONFLICT, "Connector already exists")
	} else if strings.Contains(errStr, `violates unique constraint "connectors_docker_image_name_docker_image_tag_key"`) {
		return cosmos.Errorf(cosmos.ECONFLICT, "Connector already exists")
	} else if strings.Contains(errStr, `violates unique constraint "endpoints_name_type_key"`) {
		return cosmos.Errorf(cosmos.ECONFLICT, "Endpoint already exists")
	} else if strings.Contains(errStr, `violates unique constraint "syncs_name_key"`) {
		return cosmos.Errorf(cosmos.ECONFLICT, "Sync already exists")
	} else if strings.Contains(errStr, `violates unique constraint "runs_sync_id_execution_date_key"`) {
		return cosmos.Errorf(cosmos.ECONFLICT, "Run already exists")
	}
	return err
}
