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
	PlatformFilePath string
	ConfigFilePath   string

	OtelcolEndpoint string

	AdminBrokers    string
	ConsumerBrokers string

	HttpAddr   string
	GrpcAddr   string
	PortalAddr string

	System    System
	Topic     string
	Partition int32

	StorageTarget string

	WatchDecode bool
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

	// validate all command handlers exist for each concern
	if !validateConcernHandlers() {
		log.Fatal().
			Msg("validateConcernHandlers failed")
	}

	// Make sure we at least have an empty map. Concern handler
	// codegen will validate the right stuff is in there.
	if impl.StorageInits == nil {
		impl.StorageInits = make(map[string]StorageInit)
	}
}

func prerunCobra(cmd *cobra.Command, args []string) {
	var err error
	gTelem, err = NewTelemetry(context.Background())
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to NewTelemetry")
	}
}

func postrunCobra(cmd *cobra.Command, args []string) {
	if gTelem != nil {
		gTelem.Close()
	}
}

func runCobra(impl *PlatformImpl) {
	prepPlatformImpl(impl)

	rootCmd := &cobra.Command{
		Use:               gPlatformName,
		Short:             "Rocketcycle Platform - " + gPlatformName,
		PersistentPreRun:  prerunCobra,
		PersistentPostRun: postrunCobra,
	}
	rootCmd.PersistentFlags().StringVar(&gSettings.OtelcolEndpoint, "otelcol_endpoint", "localhost:4317", "OpenTelemetry collector address")
	rootCmd.PersistentFlags().StringVar(&gSettings.AdminBrokers, "admin_brokers", "localhost:9092", "Kafka brokers for admin messages like platform updates")

	// admin sub command
	adminCmd := &cobra.Command{
		Use:   "admin",
		Short: "Admin service that manages topics and orchestration",
		Run:   cobraAdminServe,
	}
	rootCmd.AddCommand(adminCmd)

	// portal sub command
	portalCmd := &cobra.Command{
		Use:   "portal",
		Short: "Portal rest apis and client commands",
	}
	rootCmd.AddCommand(portalCmd)

	portalReadCmd := &cobra.Command{
		Use:   "read",
		Short: "read a specific resource from rest api",
	}
	portalReadCmd.PersistentFlags().StringVar(&gSettings.PortalAddr, "portal_addr", "http://localhost:11371", "Address against which to make client requests")
	portalCmd.AddCommand(portalReadCmd)

	portalReadPlatformCmd := &cobra.Command{
		Use:   "platform",
		Short: "read the platform definition",
		Run:   cobraPortalReadPlatform,
	}
	portalReadCmd.AddCommand(portalReadPlatformCmd)

	portalReadConfigCmd := &cobra.Command{
		Use:   "config",
		Short: "read the current config",
		Run:   cobraPortalReadConfig,
	}
	portalReadCmd.AddCommand(portalReadConfigCmd)

	portalDecodeCmd := &cobra.Command{
		Use:   "decode",
		Short: "decode base64 opaque payloads by calling portal api",
	}
	portalCmd.AddCommand(portalDecodeCmd)
	portalDecodeCmd.PersistentFlags().StringVar(&gSettings.PortalAddr, "portal_addr", "http://localhost:11371", "Address against which to make client requests")

	portalReadProducersCmd := &cobra.Command{
		Use:   "producers",
		Short: "read the active tracked producers",
		Run:   cobraPortalReadProducers,
	}
	portalReadCmd.AddCommand(portalReadProducersCmd)

	portalCancelTxnCmd := &cobra.Command{
		Use:       "cancel",
		Short:     "Cancel APECS transaction",
		Args:      cobra.MinimumNArgs(1),
		ValidArgs: []string{"txn_id"},
		Run:       cobraPortalCancelTxn,
	}
	portalCancelTxnCmd.PersistentFlags().StringVar(&gSettings.PortalAddr, "portal_addr", "localhost:11381", "Address against which to make client requests")
	portalCmd.AddCommand(portalCancelTxnCmd)

	portalDecodeInstanceCmd := &cobra.Command{
		Use:       "instance concern base64_payload",
		Short:     "decode and print base64 payload",
		Run:       cobraPortalDecodeInstance,
		Args:      cobra.MinimumNArgs(2),
		ValidArgs: []string{"concern", "base64_payload"},
	}
	portalDecodeCmd.AddCommand(portalDecodeInstanceCmd)

	portalServeCmd := &cobra.Command{
		Use:   "serve",
		Short: "Rocketcycle Portal Server",
		Long:  "Host rest api",
		Run:   cobraPortalServe,
	}
	portalServeCmd.PersistentFlags().StringVar(&gSettings.HttpAddr, "http_addr", ":11371", "Address to host http api")
	portalServeCmd.PersistentFlags().StringVar(&gSettings.GrpcAddr, "grpc_addr", ":11381", "Address to host grpc api")
	portalCmd.AddCommand(portalServeCmd)
	// portal sub command (END)

	// config sub command
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Config manager",
	}
	rootCmd.AddCommand(configCmd)

	configReplaceCmd := &cobra.Command{
		Use:   "replace",
		Short: "Replace config",
		Long:  "WARNING: This will fully replace stored config with the file contents!!!! Publishes contents of config file to config topic and fully replaces it.",
		Run:   cobraConfigReplace,
	}
	configReplaceCmd.PersistentFlags().StringVarP(&gSettings.ConfigFilePath, "config_file_path", "c", "./config.json", "Path to json file containing Config values")
	configCmd.AddCommand(configReplaceCmd)

	// decode sub command
	decodeCmd := &cobra.Command{
		Use:   "decode",
		Short: "decode base64 opaque payloads",
	}
	rootCmd.AddCommand(decodeCmd)
	decodeInstanceCmd := &cobra.Command{
		Use:       "instance concern base64_payload",
		Short:     "decode and print base64 payload",
		Run:       cobraDecodeInstance,
		Args:      cobra.MinimumNArgs(2),
		ValidArgs: []string{"concern", "base64_payload"},
	}
	decodeCmd.AddCommand(decodeInstanceCmd)

	// process sub command
	procCmd := &cobra.Command{
		Use:   "process",
		Short: "APECS processing mode",
		Long:  "Runs a proc consumer against the partition specified",
		Run:   cobraApecsConsumer,
	}
	procCmd.PersistentFlags().StringVar(&gSettings.ConsumerBrokers, "consumer_brokers", "", "Kafka brokers against which to consume topic")
	procCmd.MarkPersistentFlagRequired("consumer_brokers")
	procCmd.PersistentFlags().StringVarP(&gSettings.Topic, "topic", "t", "", "Topic to consume")
	procCmd.MarkPersistentFlagRequired("topic")
	procCmd.PersistentFlags().Int32VarP(&gSettings.Partition, "partition", "p", -1, "Partition to consume")
	procCmd.MarkPersistentFlagRequired("partition")
	rootCmd.AddCommand(procCmd)

	// storage sub command
	storageCmd := &cobra.Command{
		Use:   "storage",
		Short: "APECS storage mode",
		Long:  "Runs a storage consumer against the partition specified",
		Run:   cobraApecsConsumer,
	}
	storageCmd.PersistentFlags().StringVar(&gSettings.ConsumerBrokers, "consumer_brokers", "", "Kafka brokers against which to consume topic")
	storageCmd.MarkPersistentFlagRequired("consumer_brokers")
	storageCmd.PersistentFlags().StringVarP(&gSettings.Topic, "topic", "t", "", "Topic to consume")
	storageCmd.MarkPersistentFlagRequired("topic")
	storageCmd.PersistentFlags().Int32VarP(&gSettings.Partition, "partition", "p", -1, "Partition to consume")
	storageCmd.MarkPersistentFlagRequired("partition")
	storageCmd.PersistentFlags().StringVar(&gSettings.StorageTarget, "storage_target", "", "One of the named storage targets defined within platform config")
	storageCmd.MarkPersistentFlagRequired("storage_target")
	rootCmd.AddCommand(storageCmd)

	// secondary-storage sub command
	storageScndCmd := &cobra.Command{
		Use:   "storage-scnd",
		Short: "APECS secondary storage mode",
		Long:  "Runs a secondary storage consumer against the partition specified",
		Run:   cobraApecsConsumer,
	}
	storageScndCmd.PersistentFlags().StringVar(&gSettings.ConsumerBrokers, "consumer_brokers", "", "Kafka brokers against which to consume topic")
	storageScndCmd.MarkPersistentFlagRequired("consumer_brokers")
	storageScndCmd.PersistentFlags().StringVarP(&gSettings.Topic, "topic", "t", "", "Topic to consume")
	storageScndCmd.MarkPersistentFlagRequired("topic")
	storageScndCmd.PersistentFlags().Int32VarP(&gSettings.Partition, "partition", "p", -1, "Partition to consume")
	storageScndCmd.MarkPersistentFlagRequired("partition")
	storageScndCmd.PersistentFlags().StringVar(&gSettings.StorageTarget, "storage_target", "", "One of the named storage targets defined within platform config")
	storageScndCmd.MarkPersistentFlagRequired("storage_target")
	rootCmd.AddCommand(storageScndCmd)

	// watch sub command
	watchCmd := &cobra.Command{
		Use:   "watch",
		Short: "APECS watch mode",
		Long:  "Runs a watch consumer against all error/complete topics",
		Run:   cobraWatch,
	}
	watchCmd.PersistentFlags().BoolVarP(&gSettings.WatchDecode, "decode", "d", false, "If set, will decode all Buffer objects when printing ApecsTxn messages")
	rootCmd.AddCommand(watchCmd)

	// run sub command
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run all topic consumer programs",
		Long:  "Orchestrates sub processes as specified by platform topics consumer programs",
		Run:   cobraRun,
	}
	runCmd.PersistentFlags().BoolVarP(&gSettings.WatchDecode, "decode", "d", false, "If set, will decode all Buffer objects when printing ApecsTxn messages")
	rootCmd.AddCommand(runCmd)

	// plaform sub command
	platCmd := &cobra.Command{
		Use:   "platform",
		Short: "Manage platform configuration",
	}
	rootCmd.AddCommand(platCmd)

	platReplaceCmd := &cobra.Command{
		Use:   "replace",
		Short: "Replace platform config",
		Long:  "WARNING: Only use this to bootstrap a new platform!!!! Publishes contents of platform config file to platform topic and fully replaces platform. Creates platform topics if they do not already exist.",
		Run:   cobraPlatReplace,
	}
	platReplaceCmd.PersistentFlags().StringVarP(&gSettings.PlatformFilePath, "platform_file_path", "p", "./platform.json", "Path to json file containing platform configuration")
	platCmd.AddCommand(platReplaceCmd)

	for _, addtlCmd := range gPlatformImpl.CobraCommands {
		rootCmd.AddCommand(addtlCmd)
	}

	rootCmd.Execute()
}
