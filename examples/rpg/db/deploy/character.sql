-- Deploy rpg:character to pg
-- requires: player

BEGIN;

SET client_min_messages = 'warning';

CREATE TABLE rpg.character (
  id UUID PRIMARY KEY,
  player_id UUID REFERENCES rpg.player(id) NOT NULL,
  fullname TEXT NOT NULL,
  active BOOL NOT NULL,

  -- rocketcyle annotations for "most recent offsets"
  mro_generation INT NOT NULL CHECK (mro_generation >= 1),
  mro_partition INT NOT NULL CHECK (mro_partition >= 0),
  mro_offset BIGINT NOT NULL CHECK (mro_offset >= 0)
);

COMMIT;
