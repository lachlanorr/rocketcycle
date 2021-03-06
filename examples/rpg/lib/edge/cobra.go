// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package edge

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// Cobra sets these values based on command parsing
type Settings struct {
	BootstrapServers string
	HttpAddr         string
	GrpcAddr         string
	EdgeAddr         string

	Topic     string
	Partition int32
}

var (
	settings Settings = Settings{Partition: -1}
)

func preRunCobra(cmd *cobra.Command, args []string) {
	if settings.BootstrapServers != "" {
		log.Logger = log.With().
			Str("BootstrapServers", settings.BootstrapServers).
			Logger()
	}
	if settings.Topic != "" {
		log.Logger = log.With().
			Str("Topic", settings.Topic).
			Logger()
	}
	if settings.Partition != -1 {
		log.Logger = log.With().
			Int32("Partition", settings.Partition).
			Logger()
	}
}

func CobraCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:              "edge",
		Short:            "RPG Edge Rest Api",
		Long:             "Client interaction rest api that provides synchronous access over http",
		PersistentPreRun: preRunCobra,
	}

	getCmd := &cobra.Command{
		Use:       "get resource id",
		Short:     "get a specific resource from rest api",
		Run:       cobraGetResource,
		Args:      cobra.ExactArgs(2),
		ValidArgs: []string{"resource", "id"},
	}
	getCmd.PersistentFlags().StringVarP(&settings.EdgeAddr, "rcedge_addr", "", "http://localhost:11372", "Address against which to make client requests")
	rootCmd.AddCommand(getCmd)

	createCmd := &cobra.Command{
		Use:       "create resource [key1=val1] [key2=val2]",
		Short:     "create a resource from provided arguments",
		Run:       cobraCreateResource,
		Args:      cobra.MinimumNArgs(2),
		ValidArgs: []string{"resource"},
	}
	createCmd.PersistentFlags().StringVarP(&settings.EdgeAddr, "rcedge_addr", "", "http://localhost:11372", "Address against which to make client requests")
	rootCmd.AddCommand(createCmd)

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Rocketcycle Edge Api Server",
		Long:  "Provides rest entrypoints into application",
		Run:   cobraServe,
	}
	serveCmd.PersistentFlags().StringVarP(&settings.BootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read platform config")
	serveCmd.PersistentFlags().StringVarP(&settings.HttpAddr, "http_addr", "", ":11372", "Address to host http api")
	serveCmd.PersistentFlags().StringVarP(&settings.GrpcAddr, "grpc_addr", "", ":11382", "Address to host grpc api")
	serveCmd.PersistentFlags().StringVarP(&settings.Topic, "topic", "t", "", "Topic to consume")
	serveCmd.MarkPersistentFlagRequired("topic")
	serveCmd.PersistentFlags().Int32VarP(&settings.Partition, "partition", "p", -1, "Partition index to consume")
	serveCmd.MarkPersistentFlagRequired("partition")
	rootCmd.AddCommand(serveCmd)

	return rootCmd
}
