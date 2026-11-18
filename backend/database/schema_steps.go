package database

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"rantauplus/backend/database/entities"
)

func migrateInitial(tx *gorm.DB) error {
	if err := tx.AutoMigrate(&entities.User{}, &entities.FinancialPeriod{}, &entities.Category{},
		&entities.Transaction{}, &entities.Obligation{}, &entities.Reminder{}, &entities.RefreshToken{}); err != nil {
		return err
	}
	if err := addConstraints(tx, []constraint{
		{"financial_periods", "financial_periods_user_id_fkey", "FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE"},
		{"categories", "categories_user_id_fkey", "FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE"},
		{"categories", "categories_id_type_key", "UNIQUE (id, type)"},
		{"transactions", "transactions_user_id_fkey", "FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE"},
		{"transactions", "transactions_user_id_period_id_fkey", "FOREIGN KEY (user_id, period_id) REFERENCES financial_periods (user_id, id)"},
		{"transactions", "transactions_category_id_type_fkey", "FOREIGN KEY (category_id, type) REFERENCES categories (id, type)"},
		{"obligations", "obligations_user_id_fkey", "FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE"},
		{"obligations", "obligations_user_id_paid_transaction_id_fkey", "FOREIGN KEY (user_id, paid_transaction_id) REFERENCES transactions (user_id, id)"},
		{"reminders", "reminders_user_id_fkey", "FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE"},
		{"refresh_tokens", "refresh_tokens_user_id_fkey", "FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE"},
	}); err != nil {
		return err
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&[]entities.Category{
		{Name: "Uang saku", Type: entities.TransactionTypeIncome},
		{Name: "Makan", Type: entities.TransactionTypeExpense},
		{Name: "Transportasi", Type: entities.TransactionTypeExpense},
		{Name: "Tagihan", Type: entities.TransactionTypeExpense},
	}).Error
}

func migrateTransactionIntegrity(tx *gorm.DB) error {
	if err := executeDDL(tx, []string{
		`ALTER TABLE categories ADD COLUMN IF NOT EXISTS ownership_scope TEXT GENERATED ALWAYS AS
			(CASE WHEN user_id IS NULL THEN 'shared' ELSE user_id::text END) STORED`,
		`ALTER TABLE transactions ADD COLUMN IF NOT EXISTS category_is_shared BOOLEAN NOT NULL DEFAULT true`,
		`UPDATE transactions AS txn SET category_is_shared = false FROM categories AS category
			WHERE txn.category_id = category.id AND category.user_id IS NOT NULL AND txn.category_is_shared`,
		`ALTER TABLE transactions ADD COLUMN IF NOT EXISTS category_ownership_scope TEXT GENERATED ALWAYS AS
			(CASE WHEN category_is_shared THEN 'shared' ELSE user_id::text END) STORED`,
		`ALTER TABLE obligations ADD COLUMN IF NOT EXISTS paid_transaction_type TEXT GENERATED ALWAYS AS ('EXPENSE'::text) STORED`,
	}); err != nil {
		return err
	}
	if err := addConstraints(tx, []constraint{
		{"categories", "categories_id_type_ownership_scope_key", "UNIQUE (id, type, ownership_scope)"},
		{"transactions", "transactions_user_id_id_type_key", "UNIQUE (user_id, id, type)"},
		{"transactions", "transactions_category_ownership_fkey", "FOREIGN KEY (category_id, type, category_ownership_scope) REFERENCES categories (id, type, ownership_scope)"},
		{"obligations", "obligations_paid_expense_fkey", "FOREIGN KEY (user_id, paid_transaction_id, paid_transaction_type) REFERENCES transactions (user_id, id, type)"},
	}); err != nil {
		return err
	}
	return executeDDL(tx, []string{
		"ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_category_id_type_fkey",
		"ALTER TABLE obligations DROP CONSTRAINT IF EXISTS obligations_user_id_paid_transaction_id_fkey",
		"ALTER TABLE categories DROP CONSTRAINT IF EXISTS categories_id_type_key",
	})
}

func migrateAuthentication(tx *gorm.DB) error {
	if err := tx.AutoMigrate(&entities.AuthSession{}, &entities.AuthEmailJob{}, &entities.AuthRateLimit{}); err != nil {
		return err
	}
	if err := executeDDL(tx, []string{
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ",
		"ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS session_id UUID, ADD COLUMN IF NOT EXISTS used_at TIMESTAMPTZ",
		"CREATE INDEX IF NOT EXISTS refresh_tokens_session_idx ON refresh_tokens (session_id)",
	}); err != nil {
		return err
	}
	return addConstraints(tx, []constraint{
		{"auth_sessions", "auth_sessions_user_id_fkey", "FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE"},
		{"refresh_tokens", "refresh_tokens_session_fkey", "FOREIGN KEY (session_id, user_id) REFERENCES auth_sessions (id, user_id) ON DELETE CASCADE"},
	})
}

func migrateFinancialSetup(tx *gorm.DB) error {
	if err := executeDDL(tx, []string{
		"ALTER TABLE obligations ADD COLUMN IF NOT EXISTS setup_period_id UUID",
		"CREATE INDEX IF NOT EXISTS obligations_setup_period_idx ON obligations (user_id, setup_period_id) WHERE setup_period_id IS NOT NULL",
	}); err != nil {
		return err
	}
	return addConstraints(tx, []constraint{
		{"obligations", "obligations_setup_period_fkey", "FOREIGN KEY (user_id, setup_period_id) REFERENCES financial_periods (user_id, id)"},
	})
}

type constraint struct {
	table      string
	name       string
	definition string
}

// Definitions are fixed PostgreSQL DDL supplied by the migration steps.
func addConstraints(tx *gorm.DB, constraints []constraint) error {
	for _, definition := range constraints {
		if tx.Migrator().HasConstraint(definition.table, definition.name) {
			continue
		}
		statement := "ALTER TABLE " + definition.table + " ADD CONSTRAINT " + definition.name + " " + definition.definition
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("constraint %s: %w", definition.name, err)
		}
	}
	return nil
}

func executeDDL(tx *gorm.DB, statements []string) error {
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
