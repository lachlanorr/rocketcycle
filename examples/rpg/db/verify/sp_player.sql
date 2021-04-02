DO $$
BEGIN

ASSERT(
SELECT EXISTS (
  SELECT *
  FROM pg_catalog.pg_proc
  JOIN pg_namespace ON pg_catalog.pg_proc.pronamespace = pg_namespace.oid
  WHERE proname = 'sp_upsert_player'
    AND pg_namespace.nspname = 'rpg'
));

ASSERT(
SELECT EXISTS (
  SELECT *
  FROM pg_catalog.pg_proc
  JOIN pg_namespace ON pg_catalog.pg_proc.pronamespace = pg_namespace.oid
  WHERE proname = 'sp_delete_player'
    AND pg_namespace.nspname = 'rpg'
));

END $$;
