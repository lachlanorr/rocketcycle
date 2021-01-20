-- Deploy mmo:character to pg
-- requires: player

BEGIN;

SET client_min_messages = 'warning';

CREATE TABLE mmo.character (
  id BIGSERIAL PRIMARY KEY,
  player_id BIGINT REFERENCES mmo.player(id) NOT NULL,
  fullname TEXT NOT NULL,
  active BOOL NOT NULL
);


DO $$
DECLARE
    i int := 1;
    end_id int := 1001;
BEGIN
    LOOP
        EXIT WHEN i = end_id;

        INSERT INTO mmo.character (id, player_id, fullname, active) VALUES (i, 1, 'sys_holding_' || i, TRUE);
        i := i + 1;
    END LOOP;
    ALTER SEQUENCE mmo.character_id_seq RESTART WITH 100001;
END $$;

COMMIT;
