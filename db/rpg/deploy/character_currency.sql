-- Deploy rpg:character_currency to pg
-- requires: character

BEGIN;

SET client_min_messages = 'warning';

CREATE TABLE rpg.character_currency (
  id UUID PRIMARY KEY,
  character_id UUID REFERENCES rpg.character(id) NOT NULL,
  gold INT NOT NULL DEFAULT 0,
  faction_0 INT NOT NULL DEFAULT 0,
  faction_1 INT NOT NULL DEFAULT 0,
  faction_2 INT NOT NULL DEFAULT 0
);

COMMIT;