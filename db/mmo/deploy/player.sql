-- Deploy oltp:player to pg
-- requires: schema

BEGIN;

SET client_min_messages = 'warning';

CREATE TABLE oltp.player (
  id BIGSERIAL PRIMARY KEY,
  username TEXT UNIQUE NOT NULL,
  active BOOL NOT NULL
);

INSERT INTO oltp.player (id, username, active) VALUES (1, 'sys_holding', TRUE);
ALTER SEQUENCE oltp.player_id_seq RESTART WITH 100001;

COMMIT;
