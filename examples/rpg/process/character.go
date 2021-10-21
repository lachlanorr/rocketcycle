// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package process

import (
	"context"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"

	"github.com/lachlanorr/rocketcycle/examples/rpg/pb"
)

func init() {
	rkcy.RegisterProcessCommands("Character", &Character{})
}

type Character struct{}

func (*Character) ValidateCreate(ctx context.Context, inst *pb.Character) (*pb.Character, error) {
	if len(inst.Fullname) < 2 {
		return nil, rkcy.NewError(rkcy.Code_INVALID_ARGUMENT, "Fullname too short")
	}
	return inst, nil
}

func (c *Character) ValidateUpdate(ctx context.Context, original *pb.Character, updated *pb.Character) (*pb.Character, error) {
	return c.ValidateCreate(ctx, updated)
}

func (Character) Fund(ctx context.Context, inst *pb.Character, payload *pb.FundingRequest) (*pb.Character, error) {
	if payload.Currency.Gold < 0 || payload.Currency.Faction_0 < 0 || payload.Currency.Faction_1 < 0 || payload.Currency.Faction_2 < 0 {
		return nil, rkcy.NewError(rkcy.Code_INVALID_ARGUMENT, "Cannot fund with negative currency")
	}

	if inst.Currency == nil {
		inst.Currency = &pb.Character_Currency{}
	}

	inst.Currency.Gold += payload.Currency.Gold
	inst.Currency.Faction_0 += payload.Currency.Faction_0
	inst.Currency.Faction_1 += payload.Currency.Faction_1
	inst.Currency.Faction_2 += payload.Currency.Faction_2

	return inst, nil
}

func (*Character) DebitFunds(ctx context.Context, inst *pb.Character, payload *pb.FundingRequest) (*pb.Character, error) {
	if payload.Currency.Gold < 0 || payload.Currency.Faction_0 < 0 || payload.Currency.Faction_1 < 0 || payload.Currency.Faction_2 < 0 {
		return nil, rkcy.NewError(rkcy.Code_INVALID_ARGUMENT, "Cannot debit with negative currency")
	}

	if payload.Currency.Gold > inst.Currency.Gold ||
		payload.Currency.Faction_0 > inst.Currency.Faction_0 ||
		payload.Currency.Faction_1 > inst.Currency.Faction_0 ||
		payload.Currency.Faction_2 > inst.Currency.Faction_0 {

		return nil, rkcy.NewError(rkcy.Code_INVALID_ARGUMENT, "Insufficient funds")
	}

	inst.Currency.Gold -= payload.Currency.Gold
	inst.Currency.Faction_0 -= payload.Currency.Faction_0
	inst.Currency.Faction_1 -= payload.Currency.Faction_1
	inst.Currency.Faction_2 -= payload.Currency.Faction_2

	return inst, nil
}

func (*Character) CreditFunds(ctx context.Context, inst *pb.Character, payload *pb.FundingRequest) (*pb.Character, error) {
	if payload.Currency.Gold < 0 || payload.Currency.Faction_0 < 0 || payload.Currency.Faction_1 < 0 || payload.Currency.Faction_2 < 0 {
		return nil, rkcy.NewError(rkcy.Code_INVALID_ARGUMENT, "Cannot credit with negative currency")
	}

	inst.Currency.Gold += payload.Currency.Gold
	inst.Currency.Faction_0 += payload.Currency.Faction_0
	inst.Currency.Faction_1 += payload.Currency.Faction_1
	inst.Currency.Faction_2 += payload.Currency.Faction_2

	return inst, nil
}
