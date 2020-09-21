-- Verify oltp:character on pg

SELECT id, player_id, fullname, active
  FROM oltp.character
WHERE FALSE;
