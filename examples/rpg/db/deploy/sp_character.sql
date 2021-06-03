BEGIN;

--------------------------------------------------------------------------------

CREATE OR REPLACE PROCEDURE rpg.sp_upsert_character(
  _id UUID,
  _player_id UUID,
  _fullname TEXT,
  _active BOOL,

  _mro_generation INT,
  _mro_partition INT,
  _mro_offset BIGINT
)
LANGUAGE plpgsql
AS $$
DECLARE
  _mro_generation_existing INT;
  _mro_partition_existing INT;
  _mro_offset_existing BIGINT;
BEGIN
  SELECT mro_generation, mro_partition, mro_offset
  INTO _mro_generation_existing, _mro_partition_existing, _mro_offset_existing
  FROM rpg.character
  WHERE id = _id;

  IF _mro_offset_existing IS NOT NULL THEN
    IF _mro_generation = _mro_generation_existing THEN
      IF _mro_partition != _mro_partition_existing THEN
        RAISE EXCEPTION 'mro_partition (%) != existing (%) for the same generation (%)', _mro_partition, _mro_partition_existing, _mro_generation;
      END IF;
      IF _mro_offset < _mro_offset_existing THEN
        RAISE EXCEPTION 'mro_offset (%) < existing (%)', _mro_offset, _mro_offset_existing;
      END IF;
    ELSIF _mro_generation < _mro_generation_existing THEN
      RAISE EXCEPTION 'mro_generation (%) < existing (%)', _mro_generation, _mro_generation_existing;
    END IF;

    UPDATE rpg.character
       SET player_id = _player_id,
           fullname = _fullname,
           active = _active,
           mro_generation = _mro_generation,
           mro_partition = _mro_partition,
           mro_offset = _mro_offset
     WHERE id = _id;
  ELSE
    INSERT INTO rpg.character
      (id,
       player_id,
       fullname,
       active,
       mro_generation,
       mro_partition,
       mro_offset)
    VALUES
      (_id,
       _player_id,
       _fullname,
       _active,
       _mro_generation,
       _mro_partition,
       _mro_offset);
  END IF;

  COMMIT;
END $$;

--------------------------------------------------------------------------------

CREATE OR REPLACE PROCEDURE rpg.sp_delete_character(
  _id UUID,

  _mro_generation INT,
  _mro_partition INT,
  _mro_offset BIGINT
)
LANGUAGE plpgsql
AS $$
DECLARE
  _mro_generation_existing INT;
  _mro_partition_existing INT;
  _mro_offset_existing BIGINT;
BEGIN
  SELECT mro_generation, mro_partition, mro_offset
  INTO _mro_generation_existing, _mro_partition_existing, _mro_offset_existing
  FROM rpg.character
  WHERE id = _id;

  IF _mro_offset_existing IS NOT NULL THEN
    IF _mro_generation = _mro_generation_existing THEN
      IF _mro_partition != _mro_partition_existing THEN
        RAISE EXCEPTION 'mro_partition (%) != existing (%) for the same generation (%)', _mro_partition, _mro_partition_existing, _mro_generation;
      END IF;
      IF _mro_offset < _mro_offset_existing THEN
        RAISE EXCEPTION 'mro_offset (%) < existing (%)', _mro_offset, _mro_offset_existing;
      END IF;
    ELSIF _mro_generation < _mro_generation_existing THEN
      RAISE EXCEPTION 'mro_generation (%) < existing (%)', _mro_generation, _mro_generation_existing;
    END IF;

    UPDATE rpg.character_item SET character_id = '00000000-0000-0000-0000-000000000000' WHERE character_id = _id;
    DELETE FROM rpg.character_currency WHERE character_id = _id;
    DELETE FROM rpg.character WHERE id = _id;
  END IF;

  COMMIT;
END $$;

--------------------------------------------------------------------------------

CREATE OR REPLACE PROCEDURE rpg.sp_upsert_character_currency(
  _character_id UUID,
  _gold INT,
  _faction_0 INT,
  _faction_1 INT,
  _faction_2 INT,

  _mro_generation INT,
  _mro_partition INT,
  _mro_offset BIGINT
)
LANGUAGE plpgsql
AS $$
DECLARE
  _mro_generation_existing INT;
  _mro_partition_existing INT;
  _mro_offset_existing BIGINT;
BEGIN
  SELECT mro_generation, mro_partition, mro_offset
  INTO _mro_generation_existing, _mro_partition_existing, _mro_offset_existing
  FROM rpg.character_currency
  WHERE character_id = _character_id;

  IF _mro_offset_existing IS NOT NULL THEN
    IF _mro_generation = _mro_generation_existing THEN
      IF _mro_partition != _mro_partition_existing THEN
        RAISE EXCEPTION 'mro_partition (%) != existing (%) for the same generation (%)', _mro_partition, _mro_partition_existing, _mro_generation;
      END IF;
      IF _mro_offset <= _mro_offset_existing THEN
        RAISE EXCEPTION 'mro_offset (%) < existing (%)', _mro_offset, _mro_offset_existing;
      END IF;
    ELSIF _mro_generation < _mro_generation_existing THEN
      RAISE EXCEPTION 'mro_generation (%) < existing (%)', _mro_generation, _mro_generation_existing;
    END IF;

    UPDATE rpg.character_currency
       SET gold = _gold,
           faction_0 = _faction_0,
           faction_1 = _faction_1,
           faction_2 = _faction_2,
           mro_generation = _mro_generation,
           mro_partition = _mro_partition,
           mro_offset = _mro_offset
     WHERE character_id = _character_id;
  ELSE
    INSERT INTO rpg.character_currency
      (character_id,
       gold,
       faction_0,
       faction_1,
       faction_2,
       mro_generation,
       mro_partition,
       mro_offset)
    VALUES
      (_character_id,
       _gold,
       _faction_0,
       _faction_1,
       _faction_2,
       _mro_generation,
       _mro_partition,
       _mro_offset);
  END IF;

  COMMIT;
END $$;

--------------------------------------------------------------------------------

CREATE OR REPLACE PROCEDURE rpg.sp_upsert_character_item(
  _id UUID,
  _character_id UUID,
  _description TEXT,

  _mro_generation INT,
  _mro_partition INT,
  _mro_offset BIGINT
)
LANGUAGE plpgsql
AS $$
DECLARE
  _mro_generation_existing INT;
  _mro_partition_existing INT;
  _mro_offset_existing BIGINT;
BEGIN
  SELECT mro_generation, mro_partition, mro_offset
  INTO _mro_generation_existing, _mro_partition_existing, _mro_offset_existing
  FROM rpg.character_item
  WHERE id = _id;

  IF _mro_offset_existing IS NOT NULL THEN
    IF _mro_generation = _mro_generation_existing THEN
      IF _mro_partition != _mro_partition_existing THEN
        RAISE EXCEPTION 'mro_partition (%) != existing (%) for the same generation (%)', _mro_partition, _mro_partition_existing, _mro_generation;
      END IF;
      IF _mro_offset < _mro_offset_existing THEN
        RAISE EXCEPTION 'mro_offset (%) < existing (%)', _mro_offset, _mro_offset_existing;
      END IF;
    ELSIF _mro_generation < _mro_generation_existing THEN
      RAISE EXCEPTION 'mro_generation (%) < existing (%)', _mro_generation, _mro_generation_existing;
    END IF;

    UPDATE rpg.character_item
       SET character_id = _character_id,
           description = description,
           mro_generation = _mro_generation,
           mro_partition = _mro_partition,
           mro_offset = _mro_offset
     WHERE id = _id;
  ELSE
    INSERT INTO rpg.character
      (id,
       character_id,
       description,
       mro_generation,
       mro_partition,
       mro_offset)
    VALUES
      (_id,
       _character_id,
       _description,
       _mro_generation,
       _mro_partition,
       _mro_offset);
  END IF;

  COMMIT;
END $$;

--------------------------------------------------------------------------------

COMMIT;
