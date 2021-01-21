-- Verify rpg:character_item on pg

SELECT id, character_id, description
  FROM rpg.character_item
WHERE FALSE;
