// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rkcy/pkg/rkcy"

	"github.com/lachlanorr/rkcy/examples/rpg/consts"
	"github.com/lachlanorr/rkcy/examples/rpg/edge"
	"github.com/lachlanorr/rkcy/examples/rpg/sim"
	"github.com/lachlanorr/rkcy/examples/rpg/storage"
)

func main() {
	impl := rkcy.PlatformImpl{
		Name: consts.Platform,

		CobraCommands: []*cobra.Command{
			edge.CobraCommand(),
			sim.CobraCommand(),
		},

		StorageHandlers: map[string]rkcy.StorageHandler{
			consts.Player: &storage.Player{},
		},
	}

	rkcy.StartPlatform(&impl)
}
