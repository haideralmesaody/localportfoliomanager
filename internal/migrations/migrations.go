package migrations

import (
	"database/sql"
	"fmt"
	"log"
)

type Migration struct {
	Version     int
	Description string
	Func        func(*sql.DB) error
}

var Migrations = []Migration{
	{
		Version:     1,
		Description: "Add FIFO tracking",
		Func:        AddFIFOTracking,
	},
	// Add future migrations here
}

// CreateMigrationsTable creates the migrations table if it doesn't exist
func CreateMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version INTEGER PRIMARY KEY,
            description TEXT NOT NULL,
            applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        );
    `)
	return err
}

// RunMigrations runs all pending migrations
func RunMigrations(db *sql.DB) error {
	// Create migrations table if it doesn't exist
	if err := CreateMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}

	// Get applied migrations
	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %v", err)
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("failed to scan migration version: %v", err)
		}
		applied[version] = true
	}

	// Run pending migrations
	for _, migration := range Migrations {
		if !applied[migration.Version] {
			log.Printf("Running migration %d: %s", migration.Version, migration.Description)

			if err := migration.Func(db); err != nil {
				return fmt.Errorf("migration %d failed: %v", migration.Version, err)
			}

			// Record successful migration
			_, err := db.Exec(
				"INSERT INTO schema_migrations (version, description) VALUES ($1, $2)",
				migration.Version,
				migration.Description,
			)
			if err != nil {
				return fmt.Errorf("failed to record migration %d: %v", migration.Version, err)
			}

			log.Printf("Migration %d completed successfully", migration.Version)
		}
	}

	return nil
}

// Add rollback function
func RollbackLastMigration(db *sql.DB) error {
	var lastVersion int
	err := db.QueryRow(`
        SELECT version FROM schema_migrations 
        ORDER BY version DESC LIMIT 1
    `).Scan(&lastVersion)
	if err != nil {
		return fmt.Errorf("failed to get last migration: %v", err)
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Remove FIFO tracking
	_, err = tx.Exec(`
        DROP TABLE IF EXISTS portfolio_stock_lots;
        ALTER TABLE portfolio_transactions 
        DROP COLUMN IF EXISTS realized_gain_avg,
        DROP COLUMN IF EXISTS realized_gain_fifo;
        DROP INDEX IF EXISTS idx_unique_transaction;
    `)
	if err != nil {
		return err
	}

	// Remove migration record
	_, err = tx.Exec(`
        DELETE FROM schema_migrations 
        WHERE version = $1
    `, lastVersion)
	if err != nil {
		return err
	}

	return tx.Commit()
}
