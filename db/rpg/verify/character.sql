-- Verify rpg:character on pg

SELECT id, player_id, fullname, active
  FROM rpg.character
WHERE FALSE;
