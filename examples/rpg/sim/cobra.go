// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package sim

import (
	"time"

	"github.com/spf13/cobra"
)

type Settings struct {
	EdgeGrpcAddr    string
	RunnerCount     uint
	SimulationCount uint
	RandomSeed      int64
}

var (
	settings Settings
)

func CobraCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "sim",
		Short: "RPG Simulator",
		Long:  "Provides client application simulation for RPG example",
	}
	rootCmd.PersistentFlags().StringVar(&settings.EdgeGrpcAddr, "edge_grpc_addr", "localhost:11360", "Address against which to make edge grpc requests")
	rootCmd.PersistentFlags().UintVar(&settings.RunnerCount, "runner_count", 1, "Number of runners to spawn")
	rootCmd.PersistentFlags().UintVar(&settings.SimulationCount, "simulation_count", 1, "Number of overall iterations each runner should complete of simulation")
	rootCmd.PersistentFlags().Int64Var(&settings.RandomSeed, "random_seed", -1, "Seed for run, same seed will run exactly the same simulation")

	createCharactersCmd := &cobra.Command{
		Use:   "create_characters",
		Short: "create characters",
		Run: func(cmd *cobra.Command, args []string) {
			if settings.RandomSeed == -1 {
				settings.RandomSeed = time.Now().UnixNano()
			}

			start(CmdCreateCharacters, &settings)
		},
	}
	rootCmd.AddCommand(createCharactersCmd)

	return rootCmd
}
