// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package sim

import (
	"fmt"
	"math/rand"

	"github.com/lachlanorr/rocketcycle/examples/rpg/concerns"
)

type StateDb struct {
	Players    []*concerns.Player
	Characters []*concerns.Character

	PlayerMap    map[string]*concerns.Player
	CharacterMap map[string]*concerns.Character
}

func NewStateDb() *StateDb {
	return &StateDb{
		Players:      make([]*concerns.Player, 0, 100),
		Characters:   make([]*concerns.Character, 0, 100),
		PlayerMap:    make(map[string]*concerns.Player),
		CharacterMap: make(map[string]*concerns.Character),
	}
}

func (stateDb *StateDb) UpsertPlayer(player *concerns.Player) {
	_, ok := stateDb.PlayerMap[player.Id]
	if !ok {
		stateDb.Players = append(stateDb.Players, player)
	}
	stateDb.PlayerMap[player.Id] = player
}

func (stateDb *StateDb) RandomPlayer(r *rand.Rand) *concerns.Player {
	return stateDb.Players[r.Intn(len(stateDb.Players))]
}

func (stateDb *StateDb) UpsertCharacter(character *concerns.Character) {
	_, ok := stateDb.CharacterMap[character.Id]
	if !ok {
		stateDb.Characters = append(stateDb.Characters, character)
	}

	stateDb.CharacterMap[character.Id] = character
}

func (stateDb *StateDb) RandomCharacter(r *rand.Rand) *concerns.Character {
	return stateDb.Characters[r.Intn(len(stateDb.Characters))]
}

func (stateDb *StateDb) Credit(charId string, funds *concerns.Character_Currency) error {
	char, ok := stateDb.CharacterMap[charId]
	if !ok {
		return fmt.Errorf("Credit invalid character %s", charId)
	}
	char.Currency.Gold += funds.Gold
	char.Currency.Faction_0 += funds.Faction_0
	char.Currency.Faction_1 += funds.Faction_1
	char.Currency.Faction_2 += funds.Faction_2
	return nil
}

func (stateDb *StateDb) Debit(charId string, funds *concerns.Character_Currency) error {
	char, ok := stateDb.CharacterMap[charId]
	if !ok {
		return fmt.Errorf("Debit invalid character %s", charId)
	}
	char.Currency.Gold -= funds.Gold
	char.Currency.Faction_0 -= funds.Faction_0
	char.Currency.Faction_1 -= funds.Faction_1
	char.Currency.Faction_2 -= funds.Faction_2
	return nil
}
