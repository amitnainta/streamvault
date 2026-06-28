package db

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/amitnainta/streamvault/internal/config"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open connects to SQLite or PostgreSQL based on config.
func Open(cfg config.DatabaseConfig) (*sql.DB, error) {
	switch cfg.Type {
	case "sqlite":
		db, err := sql.Open("sqlite3", cfg.URL+"?_journal_mode=WAL&_foreign_keys=on")
		if err != nil {
			return nil, fmt.Errorf("open sqlite: %w", err)
		}
		// SQLite: single writer, multiple readers
		db.SetMaxOpenConns(1)
		return db, nil

	case "postgres":
		db, err := sql.Open("postgres", cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("open postgres: %w", err)
		}
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		return db, nil

	default:
		return nil, fmt.Errorf("unknown database type: %q", cfg.Type)
	}
}

// Migrate runs all pending up-migrations automatically at startup.
func Migrate(db *sql.DB, dbType string) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}

	var m *migrate.Migrate
	switch dbType {
	case "sqlite":
		driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
		if err != nil {
			return fmt.Errorf("sqlite migrate driver: %w", err)
		}
		m, err = migrate.NewWithInstance("iofs", src, "sqlite3", driver)
	case "postgres":
		driver, err := postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			return fmt.Errorf("postgres migrate driver: %w", err)
		}
		m, err = migrate.NewWithInstance("iofs", src, "postgres", driver)
	}

	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
