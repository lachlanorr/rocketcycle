-- Verify mmo:character on pg

SELECT id, player_id, fullname, active
  FROM mmo.character
WHERE FALSE;
