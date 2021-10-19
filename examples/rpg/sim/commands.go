// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package sim

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/lachlanorr/rocketcycle/examples/rpg/edge"
	"github.com/lachlanorr/rocketcycle/examples/rpg/pb"
)

func cmdCreateCharacter(ctx context.Context, client edge.RpgServiceClient, r *rand.Rand, stateDb *StateDb) (string, error) {
	player, err := client.CreatePlayer(
		ctx,
		&pb.Player{
			Username: fmt.Sprintf("User_%d", r.Int31()),
			Active:   true,
		})
	if err != nil {
		return "", err
	}
	stateDb.UpsertPlayer(player)

	currency := pb.Character_Currency{
		Gold:      1000 + int32(r.Intn(10000)),
		Faction_0: 1000 + int32(r.Intn(10000)),
		Faction_1: 1000 + int32(r.Intn(10000)),
		Faction_2: 1000 + int32(r.Intn(10000)),
	}

	character, err := client.CreateCharacter(
		ctx,
		&pb.Character{
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
		"Create %s:%s(%d/%d/%d/%d)",
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
	funds := &pb.Character_Currency{
		Gold:      int32(r.Intn(100)),
		Faction_0: int32(r.Intn(100)),
		Faction_1: int32(r.Intn(100)),
		Faction_2: int32(r.Intn(100)),
	}
	character, err := client.FundCharacter(
		ctx,
		&pb.FundingRequest{
			CharacterId: character.Id,
			Currency:    funds,
		},
	)
	if err != nil {
		return "", err
	}

	stateDb.Credit(character.Id, funds)

	return fmt.Sprintf(
		"Fund %s(%d/%d/%d/%d)",
		character.Id,
		funds.Gold,
		funds.Faction_0,
		funds.Faction_1,
		funds.Faction_2,
	), nil
}

func maxi(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func cmdTrade(ctx context.Context, client edge.RpgServiceClient, r *rand.Rand, stateDb *StateDb) (string, error) {
	charLhs := stateDb.RandomCharacter(r)
	var charRhs *pb.Character
	for {
		charRhs = stateDb.RandomCharacter(r)
		if charRhs.Id != charLhs.Id {
			break
		}
	}

	fundsLhs := &pb.Character_Currency{
		Gold:      int32(r.Intn(maxi(1, int(charLhs.Currency.Gold)/100))),
		Faction_0: int32(r.Intn(maxi(1, int(charLhs.Currency.Faction_0)/100))),
		Faction_1: int32(r.Intn(maxi(1, int(charLhs.Currency.Faction_1)/100))),
		Faction_2: int32(r.Intn(maxi(1, int(charLhs.Currency.Faction_2)/100))),
	}
	fundsRhs := &pb.Character_Currency{
		Gold:      int32(r.Intn(maxi(1, int(charRhs.Currency.Gold)/100))),
		Faction_0: int32(r.Intn(maxi(1, int(charRhs.Currency.Faction_0)/100))),
		Faction_1: int32(r.Intn(maxi(1, int(charRhs.Currency.Faction_1)/100))),
		Faction_2: int32(r.Intn(maxi(1, int(charRhs.Currency.Faction_2)/100))),
	}

	_, err := client.ConductTrade(
		ctx,
		&edge.TradeRequest{
			Lhs: &pb.FundingRequest{
				CharacterId: charLhs.Id,
				Currency:    fundsLhs,
			},
			Rhs: &pb.FundingRequest{
				CharacterId: charRhs.Id,
				Currency:    fundsRhs,
			},
		},
	)
	if err != nil {
		return "", err
	}

	stateDb.Debit(charLhs.Id, fundsLhs)
	stateDb.Debit(charRhs.Id, fundsRhs)
	stateDb.Credit(charLhs.Id, fundsRhs)
	stateDb.Credit(charRhs.Id, fundsLhs)

	return fmt.Sprintf(
		"Trade %s(%d/%d/%d/%d) %s(%d/%d/%d/%d)",
		charLhs.Id,
		fundsLhs.Gold,
		fundsLhs.Faction_0,
		fundsLhs.Faction_1,
		fundsLhs.Faction_2,
		charRhs.Id,
		fundsRhs.Gold,
		fundsRhs.Faction_0,
		fundsRhs.Faction_1,
		fundsRhs.Faction_2,
	), nil
}

func cmdReadPlayer(ctx context.Context, client edge.RpgServiceClient, r *rand.Rand, stateDb *StateDb) (string, error) {
	player := stateDb.RandomPlayer(r)
	_, err := client.ReadPlayer(ctx, &edge.RpgRequest{Id: player.Id})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("ReadPlayer %s", player.Id), nil
}

func cmdReadCharacter(ctx context.Context, client edge.RpgServiceClient, r *rand.Rand, stateDb *StateDb) (string, error) {
	char := stateDb.RandomCharacter(r)
	_, err := client.ReadCharacter(ctx, &edge.RpgRequest{Id: char.Id})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("ReadCharacter %s", char.Id), nil
}
