// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package sim

import (
	"fmt"
	"math/rand"

	"github.com/lachlanorr/rocketcycle/examples/rpg/pb"
)

type Player struct {
	Inst *pb.Player
	Rel  *pb.PlayerRelated
}

type Character struct {
	Inst *pb.Character
	Rel  *pb.CharacterRelated
}

type StateDb struct {
	Players    []*Player
	Characters []*Character

	PlayerMap    map[string]*Player
	CharacterMap map[string]*Character
}

func NewStateDb() *StateDb {
	return &StateDb{
		Players:      make([]*Player, 0, 100),
		Characters:   make([]*Character, 0, 100),
		PlayerMap:    make(map[string]*Player),
		CharacterMap: make(map[string]*Character),
	}
}

func (stateDb *StateDb) UpsertPlayer(player *Player) {
	_, ok := stateDb.PlayerMap[player.Inst.Id]
	if !ok {
		stateDb.Players = append(stateDb.Players, player)
	}
	stateDb.PlayerMap[player.Inst.Id] = player
}

func (stateDb *StateDb) RandomPlayer(r *rand.Rand) *Player {
	return stateDb.Players[r.Intn(len(stateDb.Players))]
}

func (stateDb *StateDb) UpsertCharacter(character *Character) {
	_, ok := stateDb.CharacterMap[character.Inst.Id]
	if !ok {
		stateDb.Characters = append(stateDb.Characters, character)
	}

	stateDb.CharacterMap[character.Inst.Id] = character
}

func (stateDb *StateDb) RandomCharacter(r *rand.Rand) *Character {
	return stateDb.Characters[r.Intn(len(stateDb.Characters))]
}

func (stateDb *StateDb) Credit(charId string, funds *pb.Character_Currency) error {
	char, ok := stateDb.CharacterMap[charId]
	if !ok {
		return fmt.Errorf("Credit invalid character %s", charId)
	}
	char.Inst.Currency.Gold += funds.Gold
	char.Inst.Currency.Faction_0 += funds.Faction_0
	char.Inst.Currency.Faction_1 += funds.Faction_1
	char.Inst.Currency.Faction_2 += funds.Faction_2
	return nil
}

func (stateDb *StateDb) Debit(charId string, funds *pb.Character_Currency) error {
	char, ok := stateDb.CharacterMap[charId]
	if !ok {
		return fmt.Errorf("Debit invalid character %s", charId)
	}
	char.Inst.Currency.Gold -= funds.Gold
	char.Inst.Currency.Faction_0 -= funds.Faction_0
	char.Inst.Currency.Faction_1 -= funds.Faction_1
	char.Inst.Currency.Faction_2 -= funds.Faction_2
	return nil
}
