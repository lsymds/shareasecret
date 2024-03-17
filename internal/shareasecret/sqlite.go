package shareasecret

import (
	"database/sql"
	"fmt"
)

type database struct {
	db *sql.DB
}

// NewDatabase creates a SQLite database connection, configures it (WAL etc) and migrates it to the latest available
// migration embedded within this executable.
func newSqliteDatabase(conStr string) (*database, error) {
	c, err := sql.Open("sqlite3", conStr)
	if err != nil {
		return nil, fmt.Errorf("db open: %w", err)
	}

	return &database{
		db: c,
	}, nil
}
