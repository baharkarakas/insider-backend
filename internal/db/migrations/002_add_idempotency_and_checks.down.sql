-- Rollback: unique index ve kolon/constraint’i geri al

-- Index'i düşür
DROP INDEX IF EXISTS public.ux_transactions_idempotency_key;

-- Sütunu düşür
ALTER TABLE public.transactions
  DROP COLUMN IF EXISTS idempotency_key;

-- CHECK constraint'i düşür
ALTER TABLE public.balances
  DROP CONSTRAINT IF EXISTS balances_amount_nonnegative;
