// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package sim

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/lachlanorr/rocketcycle/examples/rpg/concerns"
	"github.com/lachlanorr/rocketcycle/examples/rpg/edge"
)

func cmdCreateCharacter(ctx context.Context, client edge.RpgServiceClient, r *rand.Rand, stateDb *StateDb) (string, error) {
	player, err := client.CreatePlayer(
		ctx,
		&concerns.Player{
			Username: fmt.Sprintf("User_%d", r.Int31()),
			Active:   true,
		})
	if err != nil {
		return "", err
	}
	stateDb.UpsertPlayer(player)

	currency := concerns.Character_Currency{
		Gold:      1000 + int32(r.Intn(10000)),
		Faction_0: 1000 + int32(r.Intn(10000)),
		Faction_1: 1000 + int32(r.Intn(10000)),
		Faction_2: 1000 + int32(r.Intn(10000)),
	}

	character, err := client.CreateCharacter(
		ctx,
		&concerns.Character{
			PlayerId: player.Id,
			Fullname: fmt.Sprintf("Fullname_%d", r.Int31()),
			Active:   true,
			Currency: &currency,
		},
	)
	if err != nil {
		return "", err
	}
	stateDb.UpsertCharacter(character)

	return fmt.Sprintf(
		"Created Player(%s) and Character(%s) %d/%d/%d/%d",
		player.Id,
		character.Id,
		currency.Gold,
		currency.Faction_0,
		currency.Faction_1,
		currency.Faction_2,
	), nil
}

func cmdFund(ctx context.Context, client edge.RpgServiceClient, r *rand.Rand, stateDb *StateDb) (string, error) {
	character := stateDb.RandomCharacter(r)
	currency := concerns.Character_Currency{
		Gold:      int32(r.Intn(100)),
		Faction_0: int32(r.Intn(100)),
		Faction_1: int32(r.Intn(100)),
		Faction_2: int32(r.Intn(100)),
	}
	character, err := client.FundCharacter(
		ctx,
		&concerns.FundingRequest{
			CharacterId: character.Id,
			Currency:    &currency,
		},
	)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"Funded %s %d/%d/%d/%d",
		character.Id,
		currency.Gold,
		currency.Faction_0,
		currency.Faction_1,
		currency.Faction_2,
	), nil
}

func cmdTrade(ctx context.Context, client edge.RpgServiceClient, r *rand.Rand, stateDb *StateDb) (string, error) {
	/*
		charLhs := stateDb.RandomCharacter(r)
		var charRhs *concerns.Character
		for {
			charRhs = stateDb.RandomCharacter(r)
			if charRhs.Id != charLhs.Id {
				break
			}
		}
	*/
	return fmt.Sprintf("Trade conducted NOT IMPLEMENTED"), nil
}
