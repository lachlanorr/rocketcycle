BEGIN;

CREATE OR REPLACE PROCEDURE rpg.sp_upsert_player(
  _id UUID,
  _username TEXT,
  _active BOOL,
  _related TEXT,

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
  FROM rpg.player
  WHERE id = _id;

  IF _mro_offset_existing IS NOT NULL THEN
    IF _mro_generation = _mro_generation_existing THEN
      IF _mro_partition != _mro_partition_existing THEN
        RAISE EXCEPTION 'mro_partition (%) != existing (%) for the same generation (%)', _mro_partition, _mro_partition_existing, _mro_generation;
      END IF;
      IF _mro_offset < _mro_offset_existing THEN
        RETURN; -- this can happen and isn't that unusual, but we should never update with old data
      END IF;
    ELSIF _mro_generation < _mro_generation_existing THEN
      RETURN; -- this can happen and isn't that unusual, but we should never update with old data
    END IF;

    UPDATE rpg.player
       SET username = _username,
           active = _active,
           related = _related,
           mro_generation = _mro_generation,
           mro_partition = _mro_partition,
           mro_offset = _mro_offset
     WHERE id = _id;
  ELSE
    INSERT INTO rpg.player
      (id,
       username,
       active,
       related,
       mro_generation,
       mro_partition,
       mro_offset)
    VALUES
      (_id,
       _username,
       _active,
       _related,
       _mro_generation,
       _mro_partition,
       _mro_offset);
  END IF;

  COMMIT;
END $$;


CREATE OR REPLACE PROCEDURE rpg.sp_delete_player(
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
  FROM rpg.player
  WHERE id = _id;

  IF _mro_offset_existing IS NOT NULL THEN
    IF _mro_generation = _mro_generation_existing THEN
      IF _mro_partition != _mro_partition_existing THEN
        RAISE EXCEPTION 'mro_partition (%) != existing (%) for the same generation (%)', _mro_partition, _mro_partition_existing, _mro_generation;
      END IF;
      IF _mro_offset < _mro_offset_existing THEN
        RETURN; -- this can happen and isn't that unusual, but we should never update with old data
      END IF;
    ELSIF _mro_generation < _mro_generation_existing THEN
      RETURN; -- this can happen and isn't that unusual, but we should never update with old data
    END IF;

    DELETE FROM rpg.player WHERE id = _id;
  END IF;

  COMMIT;
END $$;

COMMIT;
