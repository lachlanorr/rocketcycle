-- Verify oltp:character_item on pg

SELECT id, character_id, description
  FROM oltp.character_item
WHERE FALSE;
