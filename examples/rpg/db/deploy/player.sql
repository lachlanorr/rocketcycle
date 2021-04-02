-- Deploy rpg:player to pg
-- requires: schema

BEGIN;

SET client_min_messages = 'warning';

CREATE TABLE rpg.player (
  id UUID PRIMARY KEY,
  username TEXT UNIQUE NOT NULL,
  active BOOL NOT NULL,

  -- rocketcyle annotations for "most recent offsets"
  mro_generation INT NOT NULL CHECK (mro_generation >= 1),
  mro_partition INT NOT NULL CHECK (mro_partition >= 0),
  mro_offset BIGINT NOT NULL CHECK (mro_offset >= 0)
);

COMMIT;
