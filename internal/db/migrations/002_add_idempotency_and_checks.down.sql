-- Rollback: unique index ve kolon/constraint’i geri al
DROP INDEX IF EXISTS ux_transactions_idempotency_key;

ALTER TABLE transactions
  DROP COLUMN IF EXISTS idempotency_key;

ALTER TABLE balances
  DROP CONSTRAINT IF EXISTS balances_amount_nonnegative;
