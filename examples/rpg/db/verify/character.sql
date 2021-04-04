-- Verify rpg:character on pg

SELECT id, player_id, fullname, active
  FROM rpg.character
WHERE FALSE;

SELECT character_id, gold, faction_0, faction_1, faction_2
  FROM rpg.character_currency
WHERE FALSE;

SELECT id, character_id, description
  FROM rpg.character_item
WHERE FALSE;
