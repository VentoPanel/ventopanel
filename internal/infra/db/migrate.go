package db

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies all pending up-migrations from the given directory.
// It is safe to call on every startup — already-applied migrations are skipped.
func RunMigrations(databaseURL, migrationsPath string) error {
	// golang-migrate needs the pgx5 scheme
	pgxURL := "pgx5://" + stripScheme(databaseURL)

	m, err := migrate.New("file://"+migrationsPath, pgxURL)
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

// stripScheme removes "postgres://" or "postgresql://" prefix so we can
// replace it with the pgx5:// scheme that golang-migrate expects.
func stripScheme(dsn string) string {
	for _, prefix := range []string{"postgresql://", "postgres://"} {
		if len(dsn) > len(prefix) && dsn[:len(prefix)] == prefix {
			return dsn[len(prefix):]
		}
	}
	return dsn
}
