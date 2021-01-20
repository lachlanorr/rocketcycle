-- Deploy mmo:character_currency to pg
-- requires: character

BEGIN;

SET client_min_messages = 'warning';

CREATE TABLE mmo.character_currency (
  id BIGSERIAL PRIMARY KEY,
  character_id BIGINT REFERENCES mmo.character(id) NOT NULL,
  gold INT NOT NULL DEFAULT 0,
  faction_0 INT NOT NULL DEFAULT 0,
  faction_1 INT NOT NULL DEFAULT 0,
  faction_2 INT NOT NULL DEFAULT 0
);

ALTER SEQUENCE mmo.character_currency_id_seq RESTART WITH 100001;

COMMIT;
