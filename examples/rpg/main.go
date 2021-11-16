// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"

	"github.com/lachlanorr/rocketcycle/examples/rpg/consts"
	"github.com/lachlanorr/rocketcycle/examples/rpg/edge"
	"github.com/lachlanorr/rocketcycle/examples/rpg/sim"

	"github.com/lachlanorr/rocketcycle/examples/rpg/crud_handlers/postgresql"
	"github.com/lachlanorr/rocketcycle/examples/rpg/logic"
)

func main() {
	rkcy.RegisterLogicHandler("Player", &logic.Player{})
	rkcy.RegisterCrudHandler("postgresql", "Player", &postgresql.Player{})

	rkcy.RegisterLogicHandler("Character", &logic.Character{})
	rkcy.RegisterCrudHandler("postgresql", "Character", &postgresql.Character{})

	impl := rkcy.PlatformImpl{
		Name: consts.Platform,

		CobraCommands: []*cobra.Command{
			edge.CobraCommand(),
			sim.CobraCommand(),
		},

		StorageInits: map[string]rkcy.StorageInit{
			"postgresql": postgresql.InitPostgresqlPool,
		},
	}

	rkcy.StartPlatform(&impl)
}
