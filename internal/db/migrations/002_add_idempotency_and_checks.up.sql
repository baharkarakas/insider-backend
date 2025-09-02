-- 002_add_idempotency_and_checks.up.sql

-- 1) transactions: idempotency_key + unique partial index
ALTER TABLE public.transactions
  ADD COLUMN IF NOT EXISTS idempotency_key TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS ux_transactions_idempotency_key
  ON public.transactions (idempotency_key)
  WHERE idempotency_key IS NOT NULL;

-- 2) balances: amount >= 0 CHECK (IF NOT EXISTS yok; plpgsql ile koşullu)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conrelid = 'public.balances'::regclass
      AND conname = 'balances_amount_nonnegative'
  ) THEN
    ALTER TABLE public.balances
      ADD CONSTRAINT balances_amount_nonnegative CHECK (amount >= 0);
  END IF;
END$$;
