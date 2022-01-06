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
	runner := runner.NewRunner(context.Background(), "rpg")

	runner.AddStorageInit("postgresql", postgresql.InitPostgresqlPool)

	runner.AddLogicHandler("Player", &logic.Player{})
	runner.AddLogicHandler("Character", &logic.Character{})

	runner.AddCrudHandler("Player", "postgresql", &postgresql.Player{})
	runner.AddCrudHandler("Character", "postgresql", &postgresql.Character{})

	runner.AddCobraEdgeCommand(edge.CobraCommand(runner.PlatformFunc()))
	runner.AddCobraCommand(sim.CobraCommand())

	runner.Execute()
}
