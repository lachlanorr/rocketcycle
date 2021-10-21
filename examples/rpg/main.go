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

	// Make sure we run the inits in these so command handlers get registered
	_ "github.com/lachlanorr/rocketcycle/examples/rpg/process"
	_ "github.com/lachlanorr/rocketcycle/examples/rpg/storage/postgresql"
)

func main() {
	impl := rkcy.PlatformImpl{
		Name: consts.Platform,

		CobraCommands: []*cobra.Command{
			edge.CobraCommand(),
			sim.CobraCommand(),
		},
	}

	rkcy.StartPlatform(&impl)
}
