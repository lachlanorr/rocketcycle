// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy/pb"
)

// Cobra sets these values based on command parsing
type Settings struct {
	ConfigFilePath   string
	BootstrapServers string

	HttpAddr  string
	GrpcAddr  string
	AdminAddr string

	Topic     string
	Partition int32

	WatchError    bool
	WatchComplete bool
}

var (
	settings         Settings = Settings{Partition: -1}
	platformImpl     *PlatformImpl
	platformHandlers PlatformHandlers
)

func GetSettings() Settings {
	return settings
}

func prepPlatformImpl(impl *PlatformImpl) {
	if impl.Name == "" {
		log.Fatal().
			Msg("No PlatformImpl.Name specificed")
	}

	platformImpl = impl
	initPlatformName(impl.Name)
	prepLogging(impl.Name)

	platformHandlers = make(map[string]map[pb.System]map[pb.Command]Handler)

	if impl.Handlers == nil {
		log.Fatal().
			Msg("No PlatformImpl.Handlers specificed")
	}
	for concernName, concernHandlers := range impl.Handlers {
		platformHandlers[concernName] = make(map[pb.System]map[pb.Command]Handler)
		if concernHandlers.CrudHandlers == nil {
			log.Fatal().
				Str("Concern", concernName).
				Msg("No CrudHandlers for concern")
		}
		if concernHandlers.Handlers == nil || len(concernHandlers.Handlers) == 0 {
			log.Fatal().
				Str("Concern", concernName).
				Msg("No ProcessHandlers specified for concern")
		}
		platformHandlers[concernName][pb.System_PROCESS] = concernHandlers.Handlers
		// Apply CrudHandlers into storage handlers
		platformHandlers[concernName][pb.System_STORAGE] = make(map[pb.Command]Handler)
		platformHandlers[concernName][pb.System_STORAGE][pb.Command_CREATE] = concernHandlers.CrudHandlers.Create
		platformHandlers[concernName][pb.System_STORAGE][pb.Command_READ] = concernHandlers.CrudHandlers.Read
		platformHandlers[concernName][pb.System_STORAGE][pb.Command_UPDATE] = concernHandlers.CrudHandlers.Update
		platformHandlers[concernName][pb.System_STORAGE][pb.Command_DELETE] = concernHandlers.CrudHandlers.Delete
	}
}

func preRunCobra(cmd *cobra.Command, args []string) {
	/*
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
	*/
}

func runCobra(impl *PlatformImpl) {
	prepPlatformImpl(impl)

	rootCmd := &cobra.Command{
		Use:              platformName,
		Short:            "Rocketcycle Platform - " + platformName,
		PersistentPreRun: preRunCobra,
	}

	// admin sub command
	adminCmd := &cobra.Command{
		Use:   "admin",
		Short: "Admin server and client command",
	}
	rootCmd.AddCommand(adminCmd)

	adminGetCmd := &cobra.Command{
		Use:   "get",
		Short: "get a specific resource from rest api",
	}
	adminGetCmd.PersistentFlags().StringVarP(&settings.AdminAddr, "admin_addr", "", "http://localhost:11371", "Address against which to make client requests")
	adminCmd.AddCommand(adminGetCmd)

	adminGetPlatformCmd := &cobra.Command{
		Use:   "platform",
		Short: "get the platform definition",
		Run:   cobraAdminGetPlatform,
	}
	adminGetCmd.AddCommand(adminGetPlatformCmd)

	adminServeCmd := &cobra.Command{
		Use:   "serve",
		Short: "Rocketcycle Admin Server",
		Long:  "Manages topics and provides rest api for admin activities",
		Run:   cobraAdminServe,
	}
	adminServeCmd.PersistentFlags().StringVarP(&settings.BootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read platform config")
	adminServeCmd.PersistentFlags().StringVarP(&settings.HttpAddr, "http_addr", "", ":11371", "Address to host http api")
	adminServeCmd.PersistentFlags().StringVarP(&settings.GrpcAddr, "grpc_addr", "", ":11381", "Address to host grpc api")
	adminCmd.AddCommand(adminServeCmd)
	// admin sub command (END)

	procCmd := &cobra.Command{
		Use:   "process",
		Short: "APECS processing mode",
		Long:  "Runs a proc consumer against the partition specified",
		Run:   cobraProcess,
	}
	procCmd.PersistentFlags().StringVarP(&settings.BootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read platform config")
	procCmd.MarkPersistentFlagRequired("bootstrap_servers")
	procCmd.PersistentFlags().StringVarP(&settings.Topic, "topic", "t", "", "Topic to consume")
	procCmd.MarkPersistentFlagRequired("topic")
	procCmd.PersistentFlags().Int32VarP(&settings.Partition, "partition", "p", -1, "Partition to consume")
	procCmd.MarkPersistentFlagRequired("partition")
	rootCmd.AddCommand(procCmd)

	storageCmd := &cobra.Command{
		Use:   "storage",
		Short: "APECS storage mode",
		Long:  "Runs a storage consumer against the partition specified",
		Run:   cobraStorage,
	}
	storageCmd.PersistentFlags().StringVarP(&settings.BootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read platform config")
	storageCmd.MarkPersistentFlagRequired("bootstrap_servers")
	storageCmd.PersistentFlags().StringVarP(&settings.Topic, "topic", "t", "", "Topic to consume")
	storageCmd.MarkPersistentFlagRequired("topic")
	storageCmd.PersistentFlags().Int32VarP(&settings.Partition, "partition", "p", -1, "Partition to consume")
	storageCmd.MarkPersistentFlagRequired("partition")
	rootCmd.AddCommand(storageCmd)

	watchCmd := &cobra.Command{
		Use:   "watch",
		Short: "APECS watch mode",
		Long:  "Runs a watch consumer against all error/complete topics",
		Run:   cobraWatch,
	}
	watchCmd.PersistentFlags().StringVarP(&settings.BootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read platform config")
	watchCmd.PersistentFlags().BoolVarP(&settings.WatchError, "error", "e", true, "Whether to watch error topics")
	watchCmd.PersistentFlags().BoolVarP(&settings.WatchComplete, "complete", "c", true, "Whether to watch complete topics")
	rootCmd.AddCommand(watchCmd)

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run all topic consumer programs",
		Long:  "Orchestrates sub processes as specified by platform topics consumer programs",
		Run:   cobraRun,
	}
	runCmd.PersistentFlags().StringVarP(&settings.BootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read platform config and begin all other processes")
	rootCmd.AddCommand(runCmd)

	platCmd := &cobra.Command{
		Use:   "platform",
		Short: "Manage platform configuration",
	}
	platCmd.PersistentFlags().StringVarP(&settings.BootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read metadata and begin all other processes")
	platCmd.PersistentFlags().StringVarP(&settings.ConfigFilePath, "config_file_path", "c", "./platform.json", "Path to json file containing platform configuration")
	rootCmd.AddCommand(platCmd)

	platUpdateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update platform config",
		Long:  "Publishes contents of platform config file to platform topic. Creates platform topic if it doesn't already exist.",
		Run:   cobraPlatUpdate,
	}
	platUpdateCmd.PersistentFlags().StringVarP(&settings.BootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read metadata and begin all other processes")
	platUpdateCmd.PersistentFlags().StringVarP(&settings.ConfigFilePath, "config_file_path", "c", "./platform.json", "Path to json file containing platform configuration")
	platCmd.AddCommand(platUpdateCmd)

	for _, addtlCmd := range platformImpl.CobraCommands {
		rootCmd.AddCommand(addtlCmd)
	}
	rootCmd.Execute()
}
