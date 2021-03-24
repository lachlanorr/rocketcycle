// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	rkcy_pb "github.com/lachlanorr/rocketcycle/pkg/rkcy/pb"

	"github.com/lachlanorr/rocketcycle/examples/rpg/commands"
	"github.com/lachlanorr/rocketcycle/examples/rpg/consts"
	"github.com/lachlanorr/rocketcycle/examples/rpg/edge"
	"github.com/lachlanorr/rocketcycle/examples/rpg/sim"
	"github.com/lachlanorr/rocketcycle/examples/rpg/storage"
)

func main() {
	impl := rkcy.PlatformImpl{
		Name: consts.Platform,

		CobraCommands: []*cobra.Command{
			edge.CobraCommand(),
			sim.CobraCommand(),
		},

		Handlers: map[string]rkcy.ConcernHandlers{
			consts.Player: {
				Handlers: map[rkcy_pb.Command]rkcy.Handler{
					rkcy_pb.Command_VALIDATE: commands.PlayerValidate,
				},
				CrudHandlers: &storage.Player{},
			},
		},
	}

	rkcy.StartPlatform(&impl)
}
