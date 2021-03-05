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

	"github.com/lachlanorr/rkcy/examples/rpg/lib"
	"github.com/lachlanorr/rkcy/pkg/rkcy"
	"github.com/lachlanorr/rkcy/version"
)

// Cobra sets these values based on command parsing
var (
	bootstrapServers string
	httpAddr         string
	grpcAddr         string
	edgeAddr         string
)

func runCobra() {
	rootCmd := &cobra.Command{
		Use:   "rpg_edge",
		Short: "Rocketcycle Edge Rest Api",
		Long:  "Client interaction rest api that provides synchronous access over http",
	}

	getCmd := &cobra.Command{
		Use:       "get resource id",
		Short:     "get a specific resource from rest api",
		Run:       cmdGetResource,
		Args:      cobra.ExactArgs(2),
		ValidArgs: []string{"resource", "id"},
	}
	getCmd.PersistentFlags().StringVarP(&edgeAddr, "rcedge_addr", "", "http://localhost:11372", "Address against which to make client requests")
	rootCmd.AddCommand(getCmd)

	createCmd := &cobra.Command{
		Use:       "create resource [key1=val1] [key2=val2]",
		Short:     "create a resource from provided arguments",
		Run:       cmdCreateResource,
		Args:      cobra.MinimumNArgs(2),
		ValidArgs: []string{"resource"},
	}
	createCmd.PersistentFlags().StringVarP(&edgeAddr, "rcedge_addr", "", "http://localhost:11372", "Address against which to make client requests")
	rootCmd.AddCommand(createCmd)

	serveCmd := &cobra.Command{
		Use:       "serve platform",
		Short:     "Rocketcycle Edge Api Server",
		Long:      "Provides rest entrypoints into application",
		Run:       cmdServe,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"platform"},
	}
	serveCmd.PersistentFlags().StringVarP(&bootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read platform config")
	serveCmd.PersistentFlags().StringVarP(&httpAddr, "http_addr", "", ":11372", "Address to host http api")
	serveCmd.PersistentFlags().StringVarP(&grpcAddr, "grpc_addr", "", ":11382", "Address to host grpc api")
	rootCmd.AddCommand(serveCmd)

	rootCmd.Execute()
}

func cmdServe(cmd *cobra.Command, args []string) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("rcedge started")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	platformName := args[0]

	go serve(ctx, httpAddr, grpcAddr, platformName)

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	select {
	case <-interruptCh:
		cancel()
		return
	}
}

func main() {
	rkcy.InitAncillary(lib.PlatformName)
	runCobra()
}
