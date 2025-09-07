
DROP INDEX IF EXISTS public.ux_transactions_idempotency_key;


ALTER TABLE public.transactions
  DROP COLUMN IF EXISTS idempotency_key;


ALTER TABLE public.balances
  DROP CONSTRAINT IF EXISTS balances_amount_nonnegative;
