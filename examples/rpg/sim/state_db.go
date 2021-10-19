// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package sim

import (
	"fmt"
	"math/rand"

	"github.com/lachlanorr/rocketcycle/examples/rpg/pb"
)

type StateDb struct {
	Players    []*pb.Player
	Characters []*pb.Character

	PlayerMap    map[string]*pb.Player
	CharacterMap map[string]*pb.Character
}

func NewStateDb() *StateDb {
	return &StateDb{
		Players:      make([]*pb.Player, 0, 100),
		Characters:   make([]*pb.Character, 0, 100),
		PlayerMap:    make(map[string]*pb.Player),
		CharacterMap: make(map[string]*pb.Character),
	}
}

func (stateDb *StateDb) UpsertPlayer(player *pb.Player) {
	_, ok := stateDb.PlayerMap[player.Id]
	if !ok {
		stateDb.Players = append(stateDb.Players, player)
	}
	stateDb.PlayerMap[player.Id] = player
}

func (stateDb *StateDb) RandomPlayer(r *rand.Rand) *pb.Player {
	return stateDb.Players[r.Intn(len(stateDb.Players))]
}

func (stateDb *StateDb) UpsertCharacter(character *pb.Character) {
	_, ok := stateDb.CharacterMap[character.Id]
	if !ok {
		stateDb.Characters = append(stateDb.Characters, character)
	}

	stateDb.CharacterMap[character.Id] = character
}

func (stateDb *StateDb) RandomCharacter(r *rand.Rand) *pb.Character {
	return stateDb.Characters[r.Intn(len(stateDb.Characters))]
}

func (stateDb *StateDb) Credit(charId string, funds *pb.Character_Currency) error {
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

func (stateDb *StateDb) Debit(charId string, funds *pb.Character_Currency) error {
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
