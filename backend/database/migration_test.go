package database

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"rantauplus/backend/database/entities"
)

func TestMigrationHistoryIDs(t *testing.T) {
	want := []string{"000001_initial_schema.up.sql", "000002_transaction_integrity.up.sql", "000003_authentication.up.sql", "000004_financial_setup"}
	if len(migrations) < len(want) {
		t.Fatalf("expected at least %d versions, got %d", len(want), len(migrations))
	}
	for i, version := range want {
		if migrations[i].version != version || migrations[i].apply == nil {
			t.Errorf("invalid migration step for existing history ID %q", version)
		}
	}
}

func TestPostgresDDLFromModels(t *testing.T) {
	var statements bytes.Buffer
	db, err := gorm.Open(postgres.Open("host=127.0.0.1 dbname=unused sslmode=disable"), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
		Logger:               logger.New(log.New(&statements, "", 0), logger.Config{LogLevel: logger.Info}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrator().CreateTable(
		&schemaMigration{}, &entities.User{}, &entities.FinancialPeriod{}, &entities.Category{},
		&entities.Transaction{}, &entities.Obligation{}, &entities.Reminder{}, &entities.RefreshToken{},
		&entities.AuthSession{}, &entities.AuthEmailJob{}, &entities.AuthRateLimit{},
	); err != nil {
		t.Fatal(err)
	}
	ddl := statements.String()
	for _, fragment := range []string{
		`"id" uuid DEFAULT gen_random_uuid()`,
		`"currency" text NOT NULL DEFAULT 'IDR'`,
		`"users_email_idx" ON "users" (lower(email))`,
		`"categories_default_name_idx" ON "categories" ("type",lower(name)) WHERE user_id IS NULL`,
		`"categories_user_name_idx" ON "categories" ("user_id","type",lower(name)) WHERE user_id IS NOT NULL`,
		`"financial_periods_user_id_id_key" ON "financial_periods" ("user_id","id")`,
		`"auth_sessions_id_user_id_key" ON "auth_sessions" ("id","user_id")`,
		`"auth_email_delivery_idx" ON "auth_email_jobs" ("delivery_after") WHERE delivered_at IS NULL AND consumed_at IS NULL`,
		`PRIMARY KEY ("email","purpose")`,
		`PRIMARY KEY ("scope","subject")`,
	} {
		if !strings.Contains(ddl, fragment) {
			t.Errorf("generated DDL is missing %s", fragment)
		}
	}
	for _, deferredColumn := range []string{`"email_verified_at"`, `"ownership_scope"`, `"category_ownership_scope"`, `"paid_transaction_type"`, `"session_id"`, `"setup_period_id"`} {
		if strings.Contains(ddl, deferredColumn) {
			t.Errorf("%s must only be added by its versioned migration step", deferredColumn)
		}
	}
}
