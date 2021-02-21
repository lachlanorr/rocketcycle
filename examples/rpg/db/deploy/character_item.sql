-- Deploy rpg:character_item to pg
-- requires: character

BEGIN;

SET client_min_messages = 'warning';

CREATE TABLE rpg.character_item (
  id UUID PRIMARY KEY,
  character_id UUID REFERENCES rpg.character(id) NOT NULL,
  description TEXT NOT NULL
);

COMMIT;
