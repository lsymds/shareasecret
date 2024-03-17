package shareasecret

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// database represents a SQLite database.
type database struct {
	db *sql.DB
}

// newDatabase creates a SQLite connection and then runs any applicable migrations or seeders.
func newDatabase(connectionString string) (*database, error) {
	con, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		return nil, err
	}

	db := &database{
		db: con,
	}

	// Run the migrations if they have been enabled.
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// migrate migrates the database within a transaction, rolling it back and returning the error
// should any occurr.
func (d *database) migrate() error {
	// You have to enable WAL outside of a transaction.
	if _, err := d.db.Exec("PRAGMA journal_mode = wal;"); err != nil {
		return fmt.Errorf("unable to enable wal: %w", err)
	}

	// You have to enable foreign key checks outside of a transaction.
	if _, err := d.db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return fmt.Errorf("unable to enable foreign keys: %w", err)
	}

	// Create the migrations table if it doesn't yet exist.
	if _, err := d.db.Exec("CREATE TABLE IF NOT EXISTS migrations (name TEXT PRIMARY KEY);"); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	// Retrieve a list of migration files to execute.
	fileNames, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("globbing migration files: %w", err)
	}
	sort.Strings(fileNames)

	// Begin a transaction, rolling it back by default.
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("unable to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Then execute them all.
	for _, fileName := range fileNames {
		if err = d.migrateFile(fileName, tx); err != nil {
			return err
		}
	}

	// Commit the transaction if we got this far.
	tx.Commit()

	return nil
}

// migrateFile runs a migration file if it hasn't been ran already.
func (d *database) migrateFile(fileName string, tx *sql.Tx) error {
	// Check if the migration has been ran before and, if it has, return early.
	var c int
	if err := tx.QueryRow("SELECT COUNT(*) FROM migrations WHERE name = ?", fileName).Scan(&c); err != nil {
		return err
	} else if c != 0 {
		return nil
	}

	// Read the file and execute it against the database.
	if buf, err := fs.ReadFile(migrationFS, fileName); err != nil {
		return err
	} else if _, err := tx.Exec(string(buf)); err != nil {
		return err
	}

	// Insert the record into the table.
	if _, err := tx.Exec("INSERT INTO migrations (name) VALUES (?)", fileName); err != nil {
		return err
	}

	return nil
}
