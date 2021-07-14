// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// Cobra sets these values based on command parsing
type Settings struct {
	ConfigFilePath string

	AdminBrokers    string
	ConsumerBrokers string

	HttpAddr  string
	GrpcAddr  string
	AdminAddr string

	Topic     string
	Partition int32

	WatchError    bool
	WatchComplete bool
	WatchDecode   bool
}

var (
	gSettings     Settings = Settings{Partition: -1}
	gPlatformImpl *PlatformImpl
	gTelem        *Telemetry
)

func Telem() *Telemetry {
	return gTelem
}

func GetSettings() Settings {
	return gSettings
}

func prepPlatformImpl(impl *PlatformImpl) {
	if impl.Name == "" {
		log.Fatal().
			Msg("No PlatformImpl.Name specificed")
	}

	gPlatformImpl = impl
	initPlatformName(impl.Name)
	prepLogging(impl.Name)
}

func runCobra(impl *PlatformImpl) {
	prepPlatformImpl(impl)

	rootCmd := &cobra.Command{
		Use:   gPlatformName,
		Short: "Rocketcycle Platform - " + gPlatformName,
	}
	rootCmd.PersistentFlags().StringVar(&gSettings.AdminBrokers, "admin_brokers", "localhost", "Kafka brokers for admin messages like platform updates")

	// admin sub command
	adminCmd := &cobra.Command{
		Use:   "admin",
		Short: "Admin server and client command",
	}
	rootCmd.AddCommand(adminCmd)

	adminReadCmd := &cobra.Command{
		Use:   "read",
		Short: "read a specific resource from rest api",
	}
	adminReadCmd.PersistentFlags().StringVar(&gSettings.AdminAddr, "admin_addr", "http://localhost:11371", "Address against which to make client requests")
	adminCmd.AddCommand(adminReadCmd)

	adminReadPlatformCmd := &cobra.Command{
		Use:   "platform",
		Short: "read the platform definition",
		Run:   cobraAdminReadPlatform,
	}
	adminReadCmd.AddCommand(adminReadPlatformCmd)

	adminDecodeCmd := &cobra.Command{
		Use:   "decode",
		Short: "decode base64 opaque payloads",
	}
	adminCmd.AddCommand(adminDecodeCmd)
	adminDecodeCmd.PersistentFlags().StringVar(&gSettings.AdminAddr, "admin_addr", "http://localhost:11371", "Address against which to make client requests")

	adminDecodeInstanceCmd := &cobra.Command{
		Use:       "instance concern base64_payload",
		Short:     "decode and print base64 payload",
		Run:       cobraAdminDecodeInstance,
		Args:      cobra.MinimumNArgs(2),
		ValidArgs: []string{"concern", "base64_payload"},
	}
	adminDecodeCmd.AddCommand(adminDecodeInstanceCmd)

	adminServeCmd := &cobra.Command{
		Use:   "serve",
		Short: "Rocketcycle Admin Server",
		Long:  "Manages topics and provides rest api for admin activities",
		Run:   cobraAdminServe,
	}
	adminServeCmd.PersistentFlags().StringVar(&gSettings.HttpAddr, "http_addr", ":11371", "Address to host http api")
	adminServeCmd.PersistentFlags().StringVar(&gSettings.GrpcAddr, "grpc_addr", ":11381", "Address to host grpc api")
	adminCmd.AddCommand(adminServeCmd)
	// admin sub command (END)

	procCmd := &cobra.Command{
		Use:   "process",
		Short: "APECS processing mode",
		Long:  "Runs a proc consumer against the partition specified",
		Run:   cobraProcess,
	}
	procCmd.PersistentFlags().StringVar(&gSettings.ConsumerBrokers, "consumer_brokers", "", "Kafka brokers against which to consume topic")
	procCmd.MarkPersistentFlagRequired("consumer_brokers")
	procCmd.PersistentFlags().StringVarP(&gSettings.Topic, "topic", "t", "", "Topic to consume")
	procCmd.MarkPersistentFlagRequired("topic")
	procCmd.PersistentFlags().Int32VarP(&gSettings.Partition, "partition", "p", -1, "Partition to consume")
	procCmd.MarkPersistentFlagRequired("partition")
	rootCmd.AddCommand(procCmd)

	storageCmd := &cobra.Command{
		Use:   "storage",
		Short: "APECS storage mode",
		Long:  "Runs a storage consumer against the partition specified",
		Run:   cobraStorage,
	}
	storageCmd.PersistentFlags().StringVar(&gSettings.ConsumerBrokers, "consumer_brokers", "", "Kafka brokers against which to consume topic")
	storageCmd.MarkPersistentFlagRequired("consumer_brokers")
	storageCmd.PersistentFlags().StringVarP(&gSettings.Topic, "topic", "t", "", "Topic to consume")
	storageCmd.MarkPersistentFlagRequired("topic")
	storageCmd.PersistentFlags().Int32VarP(&gSettings.Partition, "partition", "p", -1, "Partition to consume")
	storageCmd.MarkPersistentFlagRequired("partition")
	rootCmd.AddCommand(storageCmd)

	watchCmd := &cobra.Command{
		Use:   "watch",
		Short: "APECS watch mode",
		Long:  "Runs a watch consumer against all error/complete topics",
		Run:   cobraWatch,
	}
	watchCmd.PersistentFlags().BoolVarP(&gSettings.WatchError, "error", "e", true, "Whether to watch error topics")
	watchCmd.PersistentFlags().BoolVarP(&gSettings.WatchComplete, "complete", "c", true, "Whether to watch complete topics")
	watchCmd.PersistentFlags().BoolVarP(&gSettings.WatchDecode, "decode", "d", false, "If set, will decode all Buffer objects when printing ApecsTxn messages")
	rootCmd.AddCommand(watchCmd)

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run all topic consumer programs",
		Long:  "Orchestrates sub processes as specified by platform topics consumer programs",
		Run:   cobraRun,
	}
	runCmd.PersistentFlags().BoolVarP(&gSettings.WatchDecode, "decode", "d", false, "If set, will decode all Buffer objects when printing ApecsTxn messages")
	rootCmd.AddCommand(runCmd)

	platCmd := &cobra.Command{
		Use:   "platform",
		Short: "Manage platform configuration",
	}
	rootCmd.AddCommand(platCmd)

	platUpdateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update platform config",
		Long:  "Publishes contents of platform config file to platform topic. Creates platform topic if it doesn't already exist.",
		Run:   cobraPlatUpdate,
	}
	platUpdateCmd.PersistentFlags().StringVarP(&gSettings.ConfigFilePath, "config_file_path", "c", "./platform.json", "Path to json file containing platform configuration")
	platCmd.AddCommand(platUpdateCmd)

	for _, addtlCmd := range gPlatformImpl.CobraCommands {
		rootCmd.AddCommand(addtlCmd)
	}

	var err error
	gTelem, err = NewTelemetry(context.Background())
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to NewTelemetry")
	}
	defer gTelem.Close()

	rootCmd.Execute()
}
