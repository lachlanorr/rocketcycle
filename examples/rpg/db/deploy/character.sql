-- Deploy rpg:character to pg
-- requires: player

BEGIN;

SET client_min_messages = 'warning';

CREATE TABLE rpg.character (
  id UUID PRIMARY KEY,
  player_id UUID REFERENCES rpg.player(id) NOT NULL,
  fullname TEXT NOT NULL,
  active BOOL NOT NULL,
  related TEXT,

  -- rocketcyle annotations for "most recent offsets"
  mro_generation INT NOT NULL CHECK (mro_generation >= 1),
  mro_partition INT NOT NULL CHECK (mro_partition >= 0),
  mro_offset BIGINT NOT NULL CHECK (mro_offset >= 0)
);

INSERT INTO rpg.character (
    id,
    player_id,
    fullname,
    active,
    mro_generation,
    mro_partition,
    mro_offset
) VALUES (
    '00000000-0000-0000-0000-000000000000',
    '00000000-0000-0000-0000-000000000000',
    'sys_reserved',
    TRUE,
    1,
    0,
    0
);

CREATE TABLE rpg.character_currency (
  character_id UUID PRIMARY KEY REFERENCES rpg.character(id),
  gold INT NOT NULL DEFAULT 0,
  faction_0 INT NOT NULL DEFAULT 0,
  faction_1 INT NOT NULL DEFAULT 0,
  faction_2 INT NOT NULL DEFAULT 0,

  -- rocketcyle annotations for "most recent offsets"
  mro_generation INT NOT NULL CHECK (mro_generation >= 1),
  mro_partition INT NOT NULL CHECK (mro_partition >= 0),
  mro_offset BIGINT NOT NULL CHECK (mro_offset >= 0)
);

CREATE TABLE rpg.character_item (
  id UUID PRIMARY KEY,
  character_id UUID NOT NULL REFERENCES rpg.character(id),
  description TEXT NOT NULL,

  -- rocketcyle annotations for "most recent offsets"
  mro_generation INT NOT NULL CHECK (mro_generation >= 1),
  mro_partition INT NOT NULL CHECK (mro_partition >= 0),
  mro_offset BIGINT NOT NULL CHECK (mro_offset >= 0)
);

COMMIT;
