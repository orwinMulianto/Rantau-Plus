CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'Asia/Jakarta',
    currency TEXT NOT NULL DEFAULT 'IDR' CHECK (currency = 'IDR'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_email_idx ON users (lower(email));

CREATE TABLE financial_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    start_date DATE NOT NULL,
    next_income_date DATE NOT NULL CHECK (next_income_date > start_date),
    opening_balance BIGINT NOT NULL CHECK (opening_balance >= 0),
    reserve_amount BIGINT NOT NULL DEFAULT 0 CHECK (reserve_amount >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, id),
    UNIQUE (user_id, start_date)
);

CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('INCOME', 'EXPENSE')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, type)
);
CREATE UNIQUE INDEX categories_default_name_idx ON categories (type, lower(name)) WHERE user_id IS NULL;
CREATE UNIQUE INDEX categories_user_name_idx ON categories (user_id, type, lower(name)) WHERE user_id IS NOT NULL;

INSERT INTO categories (name, type) VALUES
    ('Uang saku', 'INCOME'), ('Makan', 'EXPENSE'), ('Transportasi', 'EXPENSE'), ('Tagihan', 'EXPENSE');

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    period_id UUID NOT NULL,
    category_id UUID NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('INCOME', 'EXPENSE')),
    amount BIGINT NOT NULL CHECK (amount > 0),
    transaction_date DATE NOT NULL,
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, id),
    FOREIGN KEY (user_id, period_id) REFERENCES financial_periods (user_id, id),
    FOREIGN KEY (category_id, type) REFERENCES categories (id, type)
);
CREATE INDEX transactions_period_date_idx ON transactions (user_id, period_id, transaction_date);

CREATE TABLE obligations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    due_date DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'PAID')),
    frequency TEXT,
    paid_at TIMESTAMPTZ,
    paid_transaction_id UUID UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (user_id, paid_transaction_id) REFERENCES transactions (user_id, id),
    CHECK (
        (status = 'PENDING' AND paid_at IS NULL AND paid_transaction_id IS NULL) OR
        (status = 'PAID' AND paid_at IS NOT NULL AND paid_transaction_id IS NOT NULL)
    )
);
CREATE INDEX obligations_pending_due_idx ON obligations (user_id, due_date) WHERE status = 'PENDING';

CREATE TABLE reminders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    due_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'COMPLETED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX reminders_user_due_idx ON reminders (user_id, due_at);

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL CHECK (expires_at > created_at),
    revoked_at TIMESTAMPTZ
);
CREATE INDEX refresh_tokens_user_idx ON refresh_tokens (user_id);
