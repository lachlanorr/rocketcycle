// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"context"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"

	"github.com/lachlanorr/rocketcycle/examples/rpg/storage"
)

func PlayerValidate(ctx context.Context, stepArgs *rkcy.StepArgs) *rkcy.StepResult {
	rslt := rkcy.StepResult{}

	mdl, err := storage.Unmarshal(stepArgs.Payload)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_MARSHAL_FAILED
		return &rslt
	}
	player, ok := mdl.(*storage.Player)
	if !ok {
		rslt.LogError("Unmarshal returned wrong type")
		rslt.Code = rkcy.Code_MARSHAL_FAILED
		return &rslt
	}

	if stepArgs.Instance != nil {
		mdl, err := storage.Unmarshal(stepArgs.Instance)
		if err != nil {
			rslt.LogError(err.Error())
			rslt.Code = rkcy.Code_MARSHAL_FAILED
			return &rslt
		}
		playerInst, ok := mdl.(*storage.Player)
		if !ok {
			rslt.LogError("Unmarshal returned wrong type")
			rslt.Code = rkcy.Code_MARSHAL_FAILED
			return &rslt
		}

		if player.Username != playerInst.Username {
			rslt.LogError("Username may not be changed")
			rslt.Code = rkcy.Code_INVALID_ARGUMENT
			return &rslt
		}
	}

	if len(player.Username) < 4 {
		rslt.LogError("Username too short")
		rslt.Code = rkcy.Code_INVALID_ARGUMENT
		return &rslt
	}

	rslt.Code = rkcy.Code_OK
	rslt.Payload = stepArgs.Payload
	return &rslt
}
