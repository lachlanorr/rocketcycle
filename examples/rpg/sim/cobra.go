// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package sim

import (
	"fmt"

	"github.com/spf13/cobra"
)

func CobraCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "sim",
		Short: "RPG Simulator",
		Long:  "Provides client application simulation for RPG example",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("test\n")
		},
	}

	return rootCmd
}
