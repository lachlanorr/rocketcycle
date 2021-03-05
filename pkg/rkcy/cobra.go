// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"github.com/spf13/cobra"
)

// Cobra sets these values based on command parsing
type Settings struct {
	ConfigFilePath   string
	BootstrapServers string
	HttpAddr         string
	GrpcAddr         string
	AdminAddr        string
	Partition        int32
}

var settings Settings

func GetSettings() Settings {
	return settings
}

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
	getCmd.PersistentFlags().StringVarP(&settings.AdminAddr, "admin_addr", "", "http://localhost:11371", "Address against which to make client requests")
	adminCmd.AddCommand(getCmd)

	getPlatformCmd := &cobra.Command{
		Use:   "platform",
		Short: "get the platform definition",
		Run:   adminGetPlatformCommand,
	}
	getCmd.AddCommand(getPlatformCmd)

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Rocketcycle Admin Server",
		Long:  "Manages topics and provides rest api for admin activities",
		Run:   adminServeCommand,
	}
	serveCmd.PersistentFlags().StringVarP(&settings.BootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read platform config")
	serveCmd.PersistentFlags().StringVarP(&settings.HttpAddr, "http_addr", "", ":11371", "Address to host http api")
	serveCmd.PersistentFlags().StringVarP(&settings.GrpcAddr, "grpc_addr", "", ":11381", "Address to host grpc api")
	adminCmd.AddCommand(serveCmd)
	// admin sub command (END)

	procCmd := &cobra.Command{
		Use:   "process",
		Short: "APECS processing mode",
		Long:  "Runs a proc consumer against the partition specified",
		Run:   procCommand,
	}
	procCmd.PersistentFlags().StringVarP(&settings.BootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read platform config")
	procCmd.PersistentFlags().StringVarP(&settings.BootstrapServers, "parition", "p", "localhost", "Kafka bootstrap servers from which to read platform config")
	procCmd.MarkPersistentFlagRequired("partition")
	rootCmd.AddCommand(procCmd)

	storageCmd := &cobra.Command{
		Use:   "storage platform partition",
		Short: "APECS storage mode",
		Long:  "Runs a storage consumer against the partition specified",
		Run:   storageCommand,
	}
	storageCmd.PersistentFlags().StringVarP(&settings.BootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read platform config")
	storageCmd.PersistentFlags().StringVarP(&settings.BootstrapServers, "parition", "p", "localhost", "Kafka bootstrap servers from which to read platform config")
	storageCmd.MarkPersistentFlagRequired("partition")
	rootCmd.AddCommand(storageCmd)

	runCmd := &cobra.Command{
		Use:   "run platform",
		Short: "Run rocketcycle platform",
		Long:  "Orchestrates sub rcproc and rcstore processes as specified by platform definition",
		Run:   runCommand,
	}
	runCmd.PersistentFlags().StringVarP(&settings.BootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read platform config and begin all other processes")
	rootCmd.AddCommand(runCmd)

	confCmd := &cobra.Command{
		Use:   "config",
		Short: "Update platform config",
		Long:  "Publishes contents of platform config file to platform topic. Creates platform topic if it doesn't already exist.",
		Run:   confCommand,
	}
	confCmd.PersistentFlags().StringVarP(&settings.BootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read metadata and begin all other processes")
	confCmd.PersistentFlags().StringVarP(&settings.ConfigFilePath, "config_file_path", "c", "./platform.json", "Path to json file containing platform configuration")
	rootCmd.AddCommand(confCmd)

	rootCmd.Execute()
}
