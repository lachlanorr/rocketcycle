// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"context"

	"github.com/lachlanorr/rocketcycle/pkg/runner"

	"github.com/lachlanorr/rocketcycle/examples/rpg/edge"
	"github.com/lachlanorr/rocketcycle/examples/rpg/sim"

	"github.com/lachlanorr/rocketcycle/examples/rpg/crud_handlers/postgresql"
	"github.com/lachlanorr/rocketcycle/examples/rpg/logic"
)

func main() {
	rkcycmd := runner.NewRkcyCmd(context.Background(), "rpg")

	rkcycmd.AddStorageInit("postgresql", postgresql.InitPostgresqlPool)

	rkcycmd.AddLogicHandler("Player", &logic.Player{})
	rkcycmd.AddLogicHandler("Character", &logic.Character{})

	rkcycmd.AddCrudHandler("Player", "postgresql", &postgresql.Player{})
	rkcycmd.AddCrudHandler("Character", "postgresql", &postgresql.Character{})

	runner.AddCobraEdgeCommandFunc(edge.CobraCommand)
	runner.AddCobraCommandFunc(sim.CobraCommand)

	rkcycmd.Execute()
}
