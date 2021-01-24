-- Deploy rpg:player to pg
-- requires: schema

BEGIN;

SET client_min_messages = 'warning';

CREATE TABLE rpg.player (
  id BIGSERIAL PRIMARY KEY,
  username TEXT UNIQUE NOT NULL,
  active BOOL NOT NULL,

  -- rocketcyle annotations for "most recent offsets"
  mro_process_topic BIGINT NOT NULL,
  mro_process_offset BIGINT NOT NULL,
  mro_storage_topic BIGINT NOT NULL,
  mro_storage_offset BIGINT NOT NULL
);

INSERT INTO rpg.player (
    id,
    username,
    active,
    mro_process_topic,
    mro_process_offset,
    mro_storage_topic,
    mro_storage_offset
) VALUES (
    1,
    'sys_holding',
    TRUE,
    0,
    0,
    0,
    0
);
ALTER SEQUENCE rpg.player_id_seq RESTART WITH 100001;

COMMIT;
