// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package sim

import (
	"time"

	"github.com/spf13/cobra"
)

type Settings struct {
	EdgeGrpcAddr       string
	RunnerCount        uint
	SimulationCount    uint
	RandomSeed         int64
	Ratios             string
	InitCharacterCount uint
	PreSleepSecs       uint
	DiffWaitSecs       uint
}

var (
	settings Settings
)

func CobraCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "sim",
		Short: "RPG Simulator",
		Long:  "Provides client application simulation for RPG example",
		Run: func(cmd *cobra.Command, args []string) {
			if settings.RandomSeed == -1 {
				settings.RandomSeed = time.Now().UnixNano()
			}

			start(&settings)
		},
	}
	rootCmd.PersistentFlags().StringVar(&settings.EdgeGrpcAddr, "edge_grpc_addr", "localhost:11360", "Address against which to make edge grpc requests")
	rootCmd.PersistentFlags().UintVar(&settings.RunnerCount, "runner_count", 1, "Number of runners to spawn")
	rootCmd.PersistentFlags().UintVar(&settings.SimulationCount, "simulation_count", 10, "Number of overall iterations each runner should complete of simulation")
	rootCmd.PersistentFlags().Int64Var(&settings.RandomSeed, "random_seed", -1, "Seed for run, same seed will run exactly the same simulation")
	rootCmd.PersistentFlags().StringVar(&settings.Ratios, "ratios", "", "Command ratios used for simulation")
	rootCmd.PersistentFlags().UintVar(&settings.InitCharacterCount, "init_character_count", 10, "Number characters to create before random simulation begins")
	rootCmd.PersistentFlags().UintVar(&settings.PreSleepSecs, "pre_sleep_secs", 0, "Number of seconds to sleep between initial character creation and simulator start")
	rootCmd.PersistentFlags().UintVar(&settings.DiffWaitSecs, "diff_wait_secs", 120, "Max time to wait for storage diffs to normalize and catch up")

	return rootCmd
}
