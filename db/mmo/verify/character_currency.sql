-- Verify mmo:character_currency on pg

SELECT id, character_id, gold, faction_0, faction_1, faction_2
  FROM mmo.character_currency
WHERE FALSE;
