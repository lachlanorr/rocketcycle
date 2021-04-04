// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"context"

	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/examples/rpg/storage"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

func CharacterValidate(ctx context.Context, stepArgs *rkcy.StepArgs) *rkcy.StepResult {
	rslt := rkcy.StepResult{}

	character := storage.Character{}
	err := proto.Unmarshal(stepArgs.Payload, &character)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_MARSHAL_FAILED
		return &rslt
	}

	if len(character.Fullname) < 2 {
		rslt.LogError("Fullname too short")
		rslt.Code = rkcy.Code_INVALID_ARGUMENT
		return &rslt
	}

	rslt.Code = rkcy.Code_OK
	rslt.Payload = stepArgs.Payload
	return &rslt
}

func CharacterFund(ctx context.Context, stepArgs *rkcy.StepArgs) *rkcy.StepResult {
	rslt := rkcy.StepResult{}

	curr := storage.FundingRequest{}
	err := proto.Unmarshal(stepArgs.Payload, &curr)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_MARSHAL_FAILED
		return &rslt
	}

	if curr.Currency.Gold < 0 || curr.Currency.Faction_0 < 0 || curr.Currency.Faction_1 < 0 || curr.Currency.Faction_2 < 0 {
		rslt.LogError("Cannot fund with negative currency")
		rslt.Code = rkcy.Code_INVALID_ARGUMENT
		return &rslt
	}

	char := storage.Character{}
	err = proto.Unmarshal(stepArgs.Instance, &char)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_MARSHAL_FAILED
		return &rslt
	}

	char.Currency.Gold += curr.Currency.Gold
	char.Currency.Faction_0 += curr.Currency.Faction_0
	char.Currency.Faction_1 += curr.Currency.Faction_1
	char.Currency.Faction_2 += curr.Currency.Faction_2

	charSer, err := proto.Marshal(&char)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_MARSHAL_FAILED
		return &rslt
	}

	rslt.Code = rkcy.Code_OK
	rslt.Instance = charSer
	rslt.Payload = charSer
	return &rslt
}
