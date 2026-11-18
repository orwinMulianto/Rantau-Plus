-- Default categories are shared. A private category belongs to the same user as its transaction.
ALTER TABLE categories
    ADD COLUMN ownership_scope TEXT GENERATED ALWAYS AS (
        CASE WHEN user_id IS NULL THEN 'shared' ELSE user_id::text END
    ) STORED,
    ADD CONSTRAINT categories_id_type_ownership_scope_key UNIQUE (id, type, ownership_scope);

ALTER TABLE transactions
    ADD COLUMN category_is_shared BOOLEAN NOT NULL DEFAULT true;

UPDATE transactions AS txn
SET category_is_shared = false
FROM categories AS category
WHERE txn.category_id = category.id AND category.user_id IS NOT NULL;

ALTER TABLE transactions
    ADD COLUMN category_ownership_scope TEXT GENERATED ALWAYS AS (
        CASE WHEN category_is_shared THEN 'shared' ELSE user_id::text END
    ) STORED,
    ADD CONSTRAINT transactions_category_ownership_fkey
        FOREIGN KEY (category_id, type, category_ownership_scope)
        REFERENCES categories (id, type, ownership_scope),
    DROP CONSTRAINT transactions_category_id_type_fkey,
    ADD CONSTRAINT transactions_user_id_id_type_key UNIQUE (user_id, id, type);

ALTER TABLE categories DROP CONSTRAINT categories_id_type_key;

-- Marking an obligation paid must link it to an expense owned by the same user.
ALTER TABLE obligations
    ADD COLUMN paid_transaction_type TEXT GENERATED ALWAYS AS ('EXPENSE'::text) STORED,
    ADD CONSTRAINT obligations_paid_expense_fkey
        FOREIGN KEY (user_id, paid_transaction_id, paid_transaction_type)
        REFERENCES transactions (user_id, id, type),
    DROP CONSTRAINT obligations_user_id_paid_transaction_id_fkey;
