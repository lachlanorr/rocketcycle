-- Verify oltp:player on pg

SELECT id, username, active
  FROM oltp.player
WHERE FALSE;
