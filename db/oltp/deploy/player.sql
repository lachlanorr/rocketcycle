-- Deploy oltp:player to pg
-- requires: schema

BEGIN;

SET client_min_messages = 'warning';

CREATE TABLE oltp.player (
  id BIGSERIAL PRIMARY KEY,
  username TEXT UNIQUE NOT NULL,
  active BOOL NOT NULL
);

COMMIT;
