// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "rcsim",
	Short: "Rocketcycle Simulator",
	Long:  "Provides client application simulation for Rocketcycle MMO economy test scenario",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("test\n")
	},
}

func main() {
	rootCmd.Execute()
}
