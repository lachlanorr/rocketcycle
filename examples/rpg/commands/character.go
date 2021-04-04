// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"context"

	"github.com/lachlanorr/rocketcycle/examples/rpg/storage"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

func CharacterValidate(ctx context.Context, stepArgs *rkcy.StepArgs) *rkcy.StepResult {
	rslt := rkcy.StepResult{}

	mdl, err := storage.Unmarshal(stepArgs.Payload)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_MARSHAL_FAILED
		return &rslt
	}
	char, ok := mdl.(*storage.Character)
	if !ok {
		rslt.LogError("Unmarshal returned wrong type")
		rslt.Code = rkcy.Code_MARSHAL_FAILED
		return &rslt
	}

	if len(char.Fullname) < 2 {
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

	mdl, err := storage.Unmarshal(stepArgs.Payload)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_MARSHAL_FAILED
		return &rslt
	}
	fr, ok := mdl.(*storage.FundingRequest)
	if !ok {
		rslt.LogError("Unmarshal returned wrong type")
		rslt.Code = rkcy.Code_MARSHAL_FAILED
		return &rslt
	}

	if fr.Currency.Gold < 0 || fr.Currency.Faction_0 < 0 || fr.Currency.Faction_1 < 0 || fr.Currency.Faction_2 < 0 {
		rslt.LogError("Cannot fund with negative currency")
		rslt.Code = rkcy.Code_INVALID_ARGUMENT
		return &rslt
	}

	mdl, err = storage.Unmarshal(stepArgs.Instance)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_MARSHAL_FAILED
		return &rslt
	}
	char, ok := mdl.(*storage.Character)
	if !ok {
		rslt.LogError("Unmarshal returned wrong type")
		rslt.Code = rkcy.Code_MARSHAL_FAILED
		return &rslt
	}

	char.Currency.Gold += fr.Currency.Gold
	char.Currency.Faction_0 += fr.Currency.Faction_0
	char.Currency.Faction_1 += fr.Currency.Faction_1
	char.Currency.Faction_2 += fr.Currency.Faction_2

	charBuf, err := storage.Marshal(int32(storage.ResourceType_CHARACTER), char)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_MARSHAL_FAILED
		return &rslt
	}

	rslt.Code = rkcy.Code_OK
	rslt.Instance = charBuf
	rslt.Payload = charBuf
	return &rslt
}
