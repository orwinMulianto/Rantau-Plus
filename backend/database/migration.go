package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	// These IDs match the names already stored by the previous runner.
	initialVersion   = "000001_initial_schema.up.sql"
	integrityVersion = "000002_transaction_integrity.up.sql"
	authVersion      = "000003_authentication.up.sql"
	setupVersion     = "000004_financial_setup"
	lockID           = int64(0x52414e544155)
)

type schemaMigration struct {
	Version   string    `gorm:"type:text;primaryKey"`
	AppliedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (schemaMigration) TableName() string { return "schema_migrations" }

var migrations = []struct {
	version string
	apply   func(*gorm.DB) error
}{
	{initialVersion, migrateInitial},
	{integrityVersion, migrateTransactionIntegrity},
	{authVersion, migrateAuthentication},
	{setupVersion, migrateFinancialSetup},
}

// Apply runs pending migrations atomically and serializes schema changes in PostgreSQL.
func Apply(ctx context.Context, db *gorm.DB, adoptInitial bool) ([]string, error) {
	var applied []string
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", lockID).Error; err != nil {
			return fmt.Errorf("mengunci migrasi: %w", err)
		}
		if err := tx.AutoMigrate(&schemaMigration{}); err != nil {
			return fmt.Errorf("menyiapkan riwayat migrasi: %w", err)
		}
		if adoptInitial {
			if err := recordExistingInitial(tx); err != nil {
				return err
			}
		}
		for _, migration := range migrations {
			var count int64
			if err := tx.Model(&schemaMigration{}).Where("version = ?", migration.version).Count(&count).Error; err != nil {
				return err
			}
			if count != 0 {
				continue
			}
			if err := migration.apply(tx); err != nil {
				return fmt.Errorf("migrasi %s: %w", migration.version, err)
			}
			if err := tx.Create(&schemaMigration{Version: migration.version}).Error; err != nil {
				return err
			}
			applied = append(applied, migration.version)
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	return applied, nil
}

func recordExistingInitial(tx *gorm.DB) error {
	for _, table := range []string{"users", "financial_periods", "categories", "transactions", "obligations", "reminders", "refresh_tokens"} {
		if !tx.Migrator().HasTable(table) {
			return errors.New("skema awal tidak lengkap; jalankan migrasi biasa pada database baru")
		}
	}
	return tx.Exec("INSERT INTO schema_migrations (version) VALUES (?) ON CONFLICT DO NOTHING", initialVersion).Error
}

func CheckReady(ctx context.Context, db *gorm.DB) error {
	versions := make([]string, 0, len(migrations))
	for _, migration := range migrations {
		versions = append(versions, migration.version)
	}
	var count int64
	if err := db.WithContext(ctx).Model(&schemaMigration{}).Where("version IN ?", versions).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(versions)) {
		return errors.New("migrasi database belum lengkap")
	}
	return nil
}
