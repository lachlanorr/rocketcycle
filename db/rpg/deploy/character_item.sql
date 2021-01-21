-- Deploy rpg:character_item to pg
-- requires: character

BEGIN;

SET client_min_messages = 'warning';

CREATE TABLE rpg.character_item (
  id BIGSERIAL PRIMARY KEY,
  character_id BIGINT REFERENCES rpg.character(id) NOT NULL,
  description TEXT NOT NULL
);

ALTER SEQUENCE rpg.character_item_id_seq RESTART WITH 100001;

COMMIT;
