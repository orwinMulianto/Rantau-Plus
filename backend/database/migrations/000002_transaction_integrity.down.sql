BEGIN;

SELECT pg_advisory_xact_lock(90440440496469);

ALTER TABLE obligations
    ADD CONSTRAINT obligations_user_id_paid_transaction_id_fkey
        FOREIGN KEY (user_id, paid_transaction_id) REFERENCES transactions (user_id, id),
    DROP CONSTRAINT obligations_paid_expense_fkey,
    DROP COLUMN paid_transaction_type;

ALTER TABLE categories ADD CONSTRAINT categories_id_type_key UNIQUE (id, type);

ALTER TABLE transactions
    ADD CONSTRAINT transactions_category_id_type_fkey
        FOREIGN KEY (category_id, type) REFERENCES categories (id, type),
    DROP CONSTRAINT transactions_category_ownership_fkey,
    DROP CONSTRAINT transactions_user_id_id_type_key,
    DROP COLUMN category_ownership_scope,
    DROP COLUMN category_is_shared;

ALTER TABLE categories
    DROP CONSTRAINT categories_id_type_ownership_scope_key,
    DROP COLUMN ownership_scope;

DO $$
BEGIN
    IF to_regclass('schema_migrations') IS NOT NULL THEN
        DELETE FROM schema_migrations WHERE version = '000002_transaction_integrity.up.sql';
    END IF;
END
$$;

COMMIT;
