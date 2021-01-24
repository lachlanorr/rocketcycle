// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rocketcycle/internal/utils"
	"github.com/lachlanorr/rocketcycle/version"
)

// Cobra sets these values based on command parsing
var (
	platform         string
	bootstrapServers string
	httpAddr         string
	grpcAddr         string
)

func runCobra() {
	rootCmd := &cobra.Command{
		Use:       "rcedge platform",
		Short:     "Rocketcycle Edge Api Server",
		Long:      "Provides rest entrypoints into application",
		Run:       rcedge,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"platform"},
	}
	rootCmd.PersistentFlags().StringVarP(&bootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read platform config")
	rootCmd.PersistentFlags().StringVarP(&httpAddr, "http_addr", "", ":11372", "Address to host http api")
	rootCmd.PersistentFlags().StringVarP(&grpcAddr, "grpc_addr", "", ":11382", "Address to host grpc api")

	rootCmd.Execute()
}

func rcedge(cmd *cobra.Command, args []string) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("rcedge started")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go serve(ctx, httpAddr, grpcAddr)

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	select {
	case <-interruptCh:
		cancel()
		return
	}
}

func main() {
	utils.PrepLogging()
	runCobra()
}
