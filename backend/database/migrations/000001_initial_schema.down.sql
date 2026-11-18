BEGIN;

SELECT pg_advisory_xact_lock(90440440496469);

DROP TABLE refresh_tokens;
DROP TABLE reminders;
DROP TABLE obligations;
DROP TABLE transactions;
DROP TABLE categories;
DROP TABLE financial_periods;
DROP TABLE users;
DROP TABLE IF EXISTS schema_migrations;

COMMIT;
