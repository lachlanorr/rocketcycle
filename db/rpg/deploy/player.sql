-- Deploy rpg:player to pg
-- requires: schema

BEGIN;

SET client_min_messages = 'warning';

CREATE TABLE rpg.player (
  id UUID PRIMARY KEY,
  username TEXT UNIQUE NOT NULL,
  active BOOL NOT NULL,

  -- rocketcyle annotations for "most recent offsets"
  mro_process_generation INT NOT NULL,
  mro_process_offset BIGINT NOT NULL,
  mro_storage_generation INT NOT NULL,
  mro_storage_offset BIGINT NOT NULL
);

INSERT INTO rpg.player (
    id,
    username,
    active,
    mro_process_generation,
    mro_process_offset,
    mro_storage_generation,
    mro_storage_offset
) VALUES (
    gen_random_uuid(),
    'sys_holding',
    TRUE,
    0,
    0,
    0,
    0
);

COMMIT;
