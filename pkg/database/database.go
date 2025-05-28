package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"log/slog"

	"github.com/XSAM/otelsql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations
var migrations embed.FS

type Config struct {
	URI          string
	MaxOpenConns int
}

func Connect(cfg Config) (*sql.DB, error) {
	db, err := otelsql.Open("pgx", cfg.URI)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)

	return db, nil
}

func Migrate(ctx context.Context, db *sql.DB, logger *slog.Logger) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}

	driver, err := postgres.WithConnection(ctx, conn, &postgres.Config{
		MigrationsTable: postgres.DefaultMigrationsTable,
	})
	if err != nil {
		return err
	}

	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return err
	}

	migrator, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return err
	}

	version, dirty, err := migrator.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return err
	}

	logger.InfoContext(ctx, "before migration", slog.Uint64("version", uint64(version)), slog.Bool("dirty", dirty))

	err = migrator.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	version, dirty, err = migrator.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return err
	}

	logger.InfoContext(ctx, "after migration", slog.Uint64("version", uint64(version)), slog.Bool("dirty", dirty))

	return nil
}
