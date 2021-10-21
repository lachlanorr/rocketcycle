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
	rkcy.RegisterProcessCommands("Player", &Player{})
}

type Player struct{}

func (*Player) ValidateCreate(ctx context.Context, inst *pb.Player) (*pb.Player, error) {
	if len(inst.Username) < 4 {
		return nil, rkcy.NewError(rkcy.Code_INVALID_ARGUMENT, "Username too short")
	}
	return inst, nil
}

func (p *Player) ValidateUpdate(ctx context.Context, original *pb.Player, updated *pb.Player) (*pb.Player, error) {
	if original.Username != updated.Username {
		return nil, rkcy.NewError(rkcy.Code_INVALID_ARGUMENT, "Username may not be changed")
	}
	return p.ValidateCreate(ctx, updated)
}
