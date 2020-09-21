-- Deploy oltp:character_item to pg
-- requires: character

BEGIN;

SET client_min_messages = 'warning';

CREATE TABLE oltp.character_item (
  id BIGSERIAL PRIMARY KEY,
  character_id BIGINT REFERENCES oltp.character(id) NOT NULL,
  description TEXT NOT NULL
);

COMMIT;
