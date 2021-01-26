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
	bootstrapServers string
	httpAddr         string
	grpcAddr         string
	adminAddr        string
)

func runCobra() {
	rootCmd := &cobra.Command{
		Use:   "rcadmin",
		Short: "Rocketcycle Admin",
		Long:  "Runs admin server and also allows for client calls through the rest api",
	}

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "get a specific resource from rest api",
	}
	getCmd.PersistentFlags().StringVarP(&adminAddr, "rcadmin_addr", "", "http://localhost:11371", "Address against which to make client requests")
	rootCmd.AddCommand(getCmd)

	getPlatformCmd := &cobra.Command{
		Use:   "platform",
		Short: "get the platform definition",
		Run:   rcadminGetPlatform,
	}
	getCmd.AddCommand(getPlatformCmd)

	serveCmd := &cobra.Command{
		Use:       "serve platform",
		Short:     "Rocketcycle Admin Server",
		Long:      "Manages topics and provides rest api for admin activities",
		Run:       rcadminServe,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"platform"},
	}
	serveCmd.PersistentFlags().StringVarP(&bootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read platform config")
	serveCmd.PersistentFlags().StringVarP(&httpAddr, "http_addr", "", ":11371", "Address to host http api")
	serveCmd.PersistentFlags().StringVarP(&grpcAddr, "grpc_addr", "", ":11381", "Address to host grpc api")
	rootCmd.AddCommand(serveCmd)

	rootCmd.Execute()
}

func rcadminServe(cmd *cobra.Command, args []string) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("rcadmin started")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	platformName := args[0]
	go manageTopics(ctx, bootstrapServers, platformName)
	go serve(ctx, httpAddr, grpcAddr)

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	select {
	case <-interruptCh:
		return
	}
}

func main() {
	utils.PrepLogging()
	runCobra()
}
