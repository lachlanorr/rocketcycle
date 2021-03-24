// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	rkcy_pb "github.com/lachlanorr/rocketcycle/pkg/rkcy/pb"
	//"github.com/lachlanorr/rocketcycle/examples/rpg/codes"
)

func PlayerValidate(ctx context.Context, stepArgs *rkcy.StepArgs) *rkcy.StepResult {
	rslt := rkcy.StepResult{}
	log.Info().Msgf("PlayerValidate ReqId=%s Key=%s", stepArgs.ReqId, stepArgs.Key)

	rslt.Code = rkcy_pb.Code_OK
	rslt.Payload = stepArgs.Payload
	return &rslt
}
