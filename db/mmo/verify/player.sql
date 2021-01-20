-- Verify mmo:player on pg

SELECT id, username, active
  FROM mmo.player
WHERE FALSE;
