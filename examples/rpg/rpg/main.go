// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rkcy/examples/rpg/lib"
	"github.com/lachlanorr/rkcy/pkg/rkcy"

	"github.com/lachlanorr/rkcy/examples/rpg/lib/edge"
	"github.com/lachlanorr/rkcy/examples/rpg/lib/sim"
)

func main() {
	impl := rkcy.PlatformImpl{
		Name: lib.PlatformName,

		CobraCommands: []*cobra.Command{
			edge.CobraCommand(),
			sim.CobraCommand(),
		},
	}

	rkcy.StartPlatform(&impl)
}
