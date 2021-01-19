// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"flag"
	"os"
	"os/signal"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rocketcycle/version"
)

var rootCmd = &cobra.Command{
	Use:   "rcedge",
	Short: "Rocketcycle Edge Api",
	Long:  "Rest api the runs on the 'edge' of the rest of the system. Provides server api for Rocketcycle MMO economy test scenario",
	Run: func(cmd *cobra.Command, args []string) {
		//fmt.Printf("test\n")
	},
}

func main() {
	//	rootCmd.Execute()

	flag.Parse()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("admin started")

	go serve(ctx)

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	select {
	case <-interruptCh:
		cancel()
		return
	}
}
