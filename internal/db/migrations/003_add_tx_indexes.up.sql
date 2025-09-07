
CREATE INDEX IF NOT EXISTS ix_tx_from_user_created_at
  ON public.transactions (from_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS ix_tx_to_user_created_at
  ON public.transactions (to_user_id, created_at DESC);
