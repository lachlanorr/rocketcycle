// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"os"
	"strconv"

	"github.com/lachlanorr/rocketcycle/pkg/offline"
	"github.com/lachlanorr/rocketcycle/pkg/platform"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcycmd"

	"github.com/lachlanorr/rocketcycle/examples/rpg/consts"
	"github.com/lachlanorr/rocketcycle/examples/rpg/edge"
	"github.com/lachlanorr/rocketcycle/examples/rpg/sim"

	"github.com/lachlanorr/rocketcycle/examples/rpg/crud_handlers/postgresql"
	"github.com/lachlanorr/rocketcycle/examples/rpg/logic"
)

func main() {
	rkcyEnvironment := os.Getenv("RKCY_ENVIRONMENT")
	if rkcyEnvironment == "" {
		panic("RKCY_ENVIRONMENT not defined")
	}
	rkcyOffline, _ := strconv.ParseBool(os.Getenv("RKCY_OFFLINE"))

	var plat rkcy.Platform

	if !rkcyOffline {
		var err error
		plat, err = platform.NewKafkaPlatform(
			consts.Platform,
			rkcyEnvironment,
		)
		if err != nil {
			panic(err.Error())
		}
	} else {
		plat = offline.NewOfflinePlatform(
			consts.Platform,
			rkcyEnvironment,
		)
	}

	plat.SetStorageInit("postgresql", postgresql.InitPostgresqlPool)

	plat.RegisterLogicHandler("Player", &logic.Player{})
	plat.RegisterCrudHandler("postgresql", "Player", &postgresql.Player{})

	plat.RegisterLogicHandler("Character", &logic.Character{})
	plat.RegisterCrudHandler("postgresql", "Character", &postgresql.Character{})

	rkcyCmd := rkcycmd.NewRkcyCmd(plat)
	rkcyCmd.AppendCobraCommand(edge.CobraCommand(plat))
	rkcyCmd.AppendCobraCommand(sim.CobraCommand())
	rkcyCmd.Start()
}
