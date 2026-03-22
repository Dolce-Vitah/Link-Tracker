package migrations

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // Enables file:// migration source.
)

func Run(db *sql.DB, migrationsPath string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create migrate postgres driver: %w", err)
	}

	pathForURI := filepath.ToSlash(filepath.Clean(migrationsPath))

	m, migErr := migrate.NewWithDatabaseInstance("file://"+pathForURI, "postgres", driver)
	if migErr != nil {
		return fmt.Errorf("create migrator: %w", migErr)
	}

	upErr := m.Up()
	if upErr == nil || errors.Is(upErr, migrate.ErrNoChange) {
		return nil
	}
	return fmt.Errorf("run up migrations: %w", upErr)
}
