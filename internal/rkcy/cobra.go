// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"github.com/spf13/cobra"
)

// Cobra sets these values based on command parsing
type Flags struct {
	configFilePath   string
	bootstrapServers string
	httpAddr         string
	grpcAddr         string
	adminAddr        string

	// positional args, get set here for convenience
	platformName string
}

var flags Flags

func runCobra(name string) {
	rootCmd := &cobra.Command{
		Use:   name,
		Short: "Rocketcycle Platform - " + name,
	}

	// admin sub command
	adminCmd := &cobra.Command{
		Use:  "admin",
		Long: "Runs admin server and also allows for client calls through the rest api",
	}
	rootCmd.AddCommand(adminCmd)

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "get a specific resource from rest api",
	}
	getCmd.PersistentFlags().StringVarP(&flags.adminAddr, "admin_addr", "", "http://localhost:11371", "Address against which to make client requests")
	adminCmd.AddCommand(getCmd)

	getPlatformCmd := &cobra.Command{
		Use:   "platform",
		Short: "get the platform definition",
		Run:   adminGetPlatformCommand,
	}
	getCmd.AddCommand(getPlatformCmd)

	serveCmd := &cobra.Command{
		Use:       "serve platform",
		Short:     "Rocketcycle Admin Server",
		Long:      "Manages topics and provides rest api for admin activities",
		Run:       adminServeCommand,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"platform"},
	}
	serveCmd.PersistentFlags().StringVarP(&flags.bootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read platform config")
	serveCmd.PersistentFlags().StringVarP(&flags.httpAddr, "http_addr", "", ":11371", "Address to host http api")
	serveCmd.PersistentFlags().StringVarP(&flags.grpcAddr, "grpc_addr", "", ":11381", "Address to host grpc api")
	adminCmd.AddCommand(serveCmd)
	// admin sub command (END)

	procCmd := &cobra.Command{
		Use:       "proc platform partition",
		Short:     "APECS processing mode",
		Long:      "Runs a proc consumer against the partition specified",
		Run:       procCommand,
		Args:      cobra.ExactArgs(2),
		ValidArgs: []string{"platform", "partition"},
	}
	rootCmd.AddCommand(procCmd)

	storageCmd := &cobra.Command{
		Use:       "store platform partition",
		Short:     "APECS storage mode",
		Long:      "Runs a storage consumer against the partition specified",
		Run:       storageCommand,
		Args:      cobra.ExactArgs(2),
		ValidArgs: []string{"platform", "partition"},
	}
	rootCmd.AddCommand(storageCmd)

	runCmd := &cobra.Command{
		Use:       "run platform",
		Short:     "Run rocketcycle platform",
		Long:      "Orchestrates sub rcproc and rcstore processes as specified by platform definition",
		Run:       runCommand,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"platform"},
	}
	runCmd.PersistentFlags().StringVarP(&flags.bootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read platform config and begin all other processes")
	rootCmd.AddCommand(runCmd)

	confCmd := &cobra.Command{
		Use:       "conf",
		Short:     "Update platform config",
		Long:      "Publishes contents of platform config file to platform topic. Creates platform topic if it doesn't already exist.",
		Run:       confCommand,
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"config_path"},
	}
	confCmd.PersistentFlags().StringVarP(&flags.bootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read metadata and begin all other processes")
	confCmd.PersistentFlags().StringVarP(&flags.configFilePath, "config_file_path", "c", "./platform.json", "Path to json file containing platform configuration")
	rootCmd.AddCommand(confCmd)

	rootCmd.Execute()
}

func Run(name string) {
	PrepLogging()
	runCobra(name)
}
