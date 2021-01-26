-- Deploy rpg:character to pg
-- requires: player

BEGIN;

SET client_min_messages = 'warning';

CREATE TABLE rpg.character (
  id UUID PRIMARY KEY,
  player_id UUID REFERENCES rpg.player(id) NOT NULL,
  fullname TEXT NOT NULL,
  active BOOL NOT NULL
);


DO $$
DECLARE
    i int := 1;
    end_id int := 1001;
    sh_player_id uuid;
BEGIN
    sh_player_id := (select id from rpg.player where username = 'sys_holding');

    LOOP
        EXIT WHEN i = end_id;

        INSERT INTO rpg.character (id, player_id, fullname, active) VALUES (gen_random_uuid(), sh_player_id, 'sys_holding_' || i, TRUE);
        i := i + 1;
    END LOOP;
END $$;

COMMIT;
