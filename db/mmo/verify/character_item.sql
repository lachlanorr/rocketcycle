-- Verify mmo:character_item on pg

SELECT id, character_id, description
  FROM mmo.character_item
WHERE FALSE;
