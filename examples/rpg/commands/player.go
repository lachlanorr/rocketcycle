// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"context"

	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"

	"github.com/lachlanorr/rocketcycle/examples/rpg/storage"
)

func PlayerValidate(ctx context.Context, stepArgs *rkcy.StepArgs) *rkcy.StepResult {
	rslt := rkcy.StepResult{}

	player := storage.Player{}
	err := proto.Unmarshal(stepArgs.Payload, &player)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_MARSHAL_FAILED
		return &rslt
	}

	if stepArgs.Instance != nil {
		playerInst := storage.Player{}
		err := proto.Unmarshal(stepArgs.Instance, &playerInst)
		if err != nil {
			rslt.LogError(err.Error())
			rslt.Code = rkcy.Code_MARSHAL_FAILED
			return &rslt
		}

		if player.Username != playerInst.Username {
			rslt.LogError("Username may not be changed")
			rslt.Code = rkcy.Code_FAILED_CONSTRAINT
			return &rslt
		}
	}

	if len(player.Username) < 4 {
		rslt.LogError("Username too short")
		rslt.Code = rkcy.Code_FAILED_CONSTRAINT
		return &rslt
	}

	rslt.Code = rkcy.Code_OK
	rslt.Payload = stepArgs.Payload
	return &rslt
}
