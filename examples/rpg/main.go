// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"

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
				Handlers: map[rkcy.Command]rkcy.Handler{
					rkcy.Command_VALIDATE_NEW: {
						Do: commands.PlayerValidate,
					},
					rkcy.Command_VALIDATE_EXISTING: {
						Do: commands.PlayerValidate,
					},
				},
				CrudHandlers: &storage.Player{},
			},
			consts.Character: {
				Handlers: map[rkcy.Command]rkcy.Handler{
					rkcy.Command_VALIDATE_NEW: {
						Do: commands.CharacterValidate,
					},
					rkcy.Command_VALIDATE_EXISTING: {
						Do: commands.CharacterValidate,
					},
				},
				CrudHandlers: &storage.Character{},
			},
		},
	}

	rkcy.StartPlatform(&impl)
}
