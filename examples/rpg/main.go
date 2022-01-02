// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"context"

	"github.com/lachlanorr/rocketcycle/pkg/rkcycmd"

	"github.com/lachlanorr/rocketcycle/examples/rpg/edge"
	"github.com/lachlanorr/rocketcycle/examples/rpg/sim"

	"github.com/lachlanorr/rocketcycle/examples/rpg/crud_handlers/postgresql"
	"github.com/lachlanorr/rocketcycle/examples/rpg/logic"
)

func main() {
	rkcyCmd := rkcycmd.NewRkcyCmd(context.Background(), "rpg")

	rkcyCmd.AddStorageInit("postgresql", postgresql.InitPostgresqlPool)

	rkcyCmd.AddLogicHandler("Player", &logic.Player{})
	rkcyCmd.AddLogicHandler("Character", &logic.Character{})

	rkcyCmd.AddCrudHandler("Player", "postgresql", &postgresql.Player{})
	rkcyCmd.AddCrudHandler("Character", "postgresql", &postgresql.Character{})

	rkcyCmd.AddCobraEdgeCommand(edge.CobraCommand(rkcyCmd.PlatformFunc()))
	rkcyCmd.AddCobraCommand(sim.CobraCommand())

	rkcyCmd.Start()
}
