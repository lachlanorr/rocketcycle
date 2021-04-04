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

INSERT INTO rpg.player (
    id,
    username,
    active,
    mro_generation,
    mro_partition,
    mro_offset
) VALUES (
    '00000000-0000-0000-0000-000000000000',
    'sys_reserved',
    TRUE,
    1,
    0,
    0
);

COMMIT;
