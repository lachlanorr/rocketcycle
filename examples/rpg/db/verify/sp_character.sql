DO $$
BEGIN

ASSERT(
SELECT EXISTS (
  SELECT *
  FROM pg_catalog.pg_proc
  JOIN pg_namespace ON pg_catalog.pg_proc.pronamespace = pg_namespace.oid
  WHERE proname = 'sp_upsert_character'
    AND pg_namespace.nspname = 'rpg'
));

ASSERT(
SELECT EXISTS (
  SELECT *
  FROM pg_catalog.pg_proc
  JOIN pg_namespace ON pg_catalog.pg_proc.pronamespace = pg_namespace.oid
  WHERE proname = 'sp_delete_character'
    AND pg_namespace.nspname = 'rpg'
));

ASSERT(
SELECT EXISTS (
  SELECT *
  FROM pg_catalog.pg_proc
  JOIN pg_namespace ON pg_catalog.pg_proc.pronamespace = pg_namespace.oid
  WHERE proname = 'sp_upsert_character_currency'
    AND pg_namespace.nspname = 'rpg'
));

ASSERT(
SELECT EXISTS (
  SELECT *
  FROM pg_catalog.pg_proc
  JOIN pg_namespace ON pg_catalog.pg_proc.pronamespace = pg_namespace.oid
  WHERE proname = 'sp_upsert_character_item'
    AND pg_namespace.nspname = 'rpg'
));

END $$;
