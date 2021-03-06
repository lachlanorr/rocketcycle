// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rkcy/examples/rpg/lib"
	"github.com/lachlanorr/rkcy/pkg/rkcy"
)

// Cobra sets these values based on command parsing
var (
	bootstrapServers string
	httpAddr         string
	grpcAddr         string
	edgeAddr         string

	topic     string
	partition int32
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
		Run:       cobraGetResource,
		Args:      cobra.ExactArgs(2),
		ValidArgs: []string{"resource", "id"},
	}
	getCmd.PersistentFlags().StringVarP(&edgeAddr, "rcedge_addr", "", "http://localhost:11372", "Address against which to make client requests")
	rootCmd.AddCommand(getCmd)

	createCmd := &cobra.Command{
		Use:       "create resource [key1=val1] [key2=val2]",
		Short:     "create a resource from provided arguments",
		Run:       cobraCreateResource,
		Args:      cobra.MinimumNArgs(2),
		ValidArgs: []string{"resource"},
	}
	createCmd.PersistentFlags().StringVarP(&edgeAddr, "rcedge_addr", "", "http://localhost:11372", "Address against which to make client requests")
	rootCmd.AddCommand(createCmd)

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Rocketcycle Edge Api Server",
		Long:  "Provides rest entrypoints into application",
		Run:   cobraServe,
	}
	serveCmd.PersistentFlags().StringVarP(&bootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read platform config")
	serveCmd.PersistentFlags().StringVarP(&httpAddr, "http_addr", "", ":11372", "Address to host http api")
	serveCmd.PersistentFlags().StringVarP(&grpcAddr, "grpc_addr", "", ":11382", "Address to host grpc api")
	serveCmd.PersistentFlags().StringVarP(&topic, "topic", "t", "", "Topic to consume")
	serveCmd.MarkPersistentFlagRequired("topic")
	serveCmd.PersistentFlags().Int32VarP(&partition, "parition", "p", -1, "Partition index to consume")
	serveCmd.MarkPersistentFlagRequired("partition")
	rootCmd.AddCommand(serveCmd)

	rootCmd.Execute()
}

func prepLogging() {
	if topic != "" {
		log.Logger = log.With().
			Str("Topic", topic).
			Logger()
	}
	if partition != -1 {
		log.Logger = log.With().
			Int32("Partition", partition).
			Logger()
	}
}

func main() {
	rkcy.InitAncillary(lib.PlatformName)
	runCobra()
}
