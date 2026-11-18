package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.up.sql
var scripts embed.FS

const (
	initialVersion = "000001_initial_schema.up.sql"
	lockID         = int64(0x52414e544155)
)

// Apply runs pending migrations atomically. Only one process can change the schema at a time.
func Apply(ctx context.Context, pool *pgxpool.Pool, adoptInitial bool) ([]string, error) {
	versions, err := migrationVersions()
	if err != nil {
		return nil, err
	}

	var applied []string
	err = pgx.BeginTxFunc(ctx, pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockID); err != nil {
			return fmt.Errorf("mengunci migrasi: %w", err)
		}
		if _, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
			return fmt.Errorf("menyiapkan riwayat migrasi: %w", err)
		}
		if adoptInitial {
			if err := recordExistingInitial(ctx, tx); err != nil {
				return err
			}
		}
		for _, version := range versions {
			var exists bool
			if err := tx.QueryRow(ctx,
				"SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)", version,
			).Scan(&exists); err != nil {
				return err
			}
			if exists {
				continue
			}
			sql, err := scripts.ReadFile("migrations/" + version)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, string(sql)); err != nil {
				return fmt.Errorf("migrasi %s: %w", version, err)
			}
			if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
				return err
			}
			applied = append(applied, version)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return applied, nil
}

func recordExistingInitial(ctx context.Context, tx pgx.Tx) error {
	initialTables := []string{
		"users", "financial_periods", "categories", "transactions", "obligations", "reminders", "refresh_tokens",
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*)
		FROM unnest($1::text[]) AS initial(table_name)
		WHERE to_regclass(table_name) IS NOT NULL`, initialTables).Scan(&count); err != nil {
		return err
	}
	if count != len(initialTables) {
		return errors.New("skema awal tidak lengkap; jalankan migrasi biasa pada database baru")
	}
	_, err := tx.Exec(ctx,
		"INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING", initialVersion)
	return err
}

func CheckReady(ctx context.Context, pool *pgxpool.Pool) error {
	versions, err := migrationVersions()
	if err != nil {
		return err
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM schema_migrations WHERE version = ANY($1::text[])",
		versions).Scan(&count); err != nil {
		return err
	}
	if count != len(versions) {
		return errors.New("migrasi database belum lengkap")
	}
	return nil
}

func migrationVersions() ([]string, error) {
	paths, err := fs.Glob(scripts, "migrations/*.up.sql")
	if err != nil {
		return nil, err
	}
	// The migration history stores filenames without their directory.
	for i, scriptPath := range paths {
		paths[i] = path.Base(scriptPath)
	}
	return paths, nil
}
