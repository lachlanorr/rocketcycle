// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package edge

import (
	"github.com/spf13/cobra"
)

// Cobra sets these values based on command parsing
type Settings struct {
	BootstrapServers string
	HttpAddr         string
	GrpcAddr         string
	EdgeHttpAddr     string

	Topic     string
	Partition int32

	TimeoutSecs int
}

var (
	settings Settings = Settings{Partition: -1}
)

func CobraCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "edge",
		Short: "RPG Edge Rest Api",
		Long:  "Client interaction rest api that provides synchronous access over http",
	}
	rootCmd.PersistentFlags().StringVar(&settings.EdgeHttpAddr, "edge_http_addr", "http://localhost:11350", "Address against which to make edge http requests")

	readCmd := &cobra.Command{
		Use:       "read resource id",
		Short:     "read a specific resource from rest api",
		Run:       cobraReadResource,
		Args:      cobra.ExactArgs(2),
		ValidArgs: []string{"resource", "id"},
	}
	rootCmd.AddCommand(readCmd)

	createCmd := &cobra.Command{
		Use:       "create resource [key1=val1] [key2=val2]",
		Short:     "create a resource from provided arguments",
		Run:       cobraCreateResource,
		Args:      cobra.MinimumNArgs(2),
		ValidArgs: []string{"resource"},
	}
	rootCmd.AddCommand(createCmd)

	updateCmd := &cobra.Command{
		Use:       "update resource id [key1=val1] [key2=val2]",
		Short:     "updates specified fields in already existing resource",
		Run:       cobraUpdateResource,
		Args:      cobra.MinimumNArgs(3),
		ValidArgs: []string{"resource", "id"},
	}
	rootCmd.AddCommand(updateCmd)

	deleteCmd := &cobra.Command{
		Use:       "delete resource id",
		Short:     "delete a specific resource from rest api",
		Run:       cobraDeleteResource,
		Args:      cobra.ExactArgs(2),
		ValidArgs: []string{"resource", "id"},
	}
	rootCmd.AddCommand(deleteCmd)

	fundCmd := &cobra.Command{
		Use:       "fund character_id [key1=val1] [key2=val2]",
		Short:     "funds character with specified currency values",
		Run:       cobraFundCharacter,
		Args:      cobra.MinimumNArgs(2),
		ValidArgs: []string{"character_id"},
	}
	rootCmd.AddCommand(fundCmd)

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Rocketcycle Edge Api Server",
		Long:  "Provides rest entrypoints into application",
		Run:   cobraServe,
	}
	serveCmd.PersistentFlags().StringVarP(&settings.BootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read platform config")
	serveCmd.PersistentFlags().StringVarP(&settings.HttpAddr, "http_addr", "", ":11350", "Address to host http api")
	serveCmd.PersistentFlags().StringVarP(&settings.GrpcAddr, "grpc_addr", "", ":11360", "Address to host grpc api")
	serveCmd.PersistentFlags().StringVarP(&settings.Topic, "topic", "t", "", "Topic to consume")
	serveCmd.MarkPersistentFlagRequired("topic")
	serveCmd.PersistentFlags().Int32VarP(&settings.Partition, "partition", "p", -1, "Partition index to consume")
	serveCmd.MarkPersistentFlagRequired("partition")
	serveCmd.PersistentFlags().IntVar(&settings.TimeoutSecs, "timeout", 30, "Client timeout")
	rootCmd.AddCommand(serveCmd)

	return rootCmd
}
