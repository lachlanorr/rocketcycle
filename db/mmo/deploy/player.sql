-- Deploy mmo:player to pg
-- requires: schema

BEGIN;

SET client_min_messages = 'warning';

CREATE TABLE mmo.player (
  id BIGSERIAL PRIMARY KEY,
  username TEXT UNIQUE NOT NULL,
  active BOOL NOT NULL
);

INSERT INTO mmo.player (id, username, active) VALUES (1, 'sys_holding', TRUE);
ALTER SEQUENCE mmo.player_id_seq RESTART WITH 100001;

COMMIT;
