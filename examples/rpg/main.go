// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"os"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"

	"github.com/lachlanorr/rocketcycle/examples/rpg/consts"
	"github.com/lachlanorr/rocketcycle/examples/rpg/edge"
	"github.com/lachlanorr/rocketcycle/examples/rpg/sim"

	"github.com/lachlanorr/rocketcycle/examples/rpg/crud_handlers/postgresql"
	"github.com/lachlanorr/rocketcycle/examples/rpg/logic"
)

func main() {
	plat, err := rkcy.NewPlatform(
		consts.Platform,
		os.Getenv("RKCY_ENVIRONMENT"),
	)
	if err != nil {
		panic(err.Error())
	}

	plat.AppendCobraCommand(edge.CobraCommand(plat))
	plat.AppendCobraCommand(sim.CobraCommand())

	plat.SetStorageInit("postgresql", postgresql.InitPostgresqlPool)

	plat.RegisterLogicHandler("Player", &logic.Player{})
	plat.RegisterCrudHandler("postgresql", "Player", &postgresql.Player{})

	plat.RegisterLogicHandler("Character", &logic.Character{})
	plat.RegisterCrudHandler("postgresql", "Character", &postgresql.Character{})

	plat.Start()
}
