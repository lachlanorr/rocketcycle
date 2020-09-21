-- Deploy oltp:character to pg
-- requires: player

BEGIN;

SET client_min_messages = 'warning';

CREATE TABLE oltp.character (
  id BIGSERIAL PRIMARY KEY,
  player_id BIGINT REFERENCES oltp.player(id) NOT NULL,
  fullname TEXT NOT NULL,
  active BOOL NOT NULL
);

COMMIT;
