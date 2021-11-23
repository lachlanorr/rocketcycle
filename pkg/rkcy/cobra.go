// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func (plat *Platform) prerunCobra(cmd *cobra.Command, args []string) {
	var err error
	plat.telem, err = NewTelemetry(context.Background(), plat.settings.OtelcolEndpoint)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to NewTelemetry")
	}

	plat.rawProducer = NewRawProducer(plat.telem)
}

func (plat *Platform) postrunCobra(cmd *cobra.Command, args []string) {
	if plat.telem != nil {
		plat.telem.Close()
	}
}

func (plat *Platform) runCobra() {
	rootCmd := &cobra.Command{
		Use:               plat.name,
		Short:             "Rocketcycle Platform - " + plat.name,
		PersistentPreRun:  plat.prerunCobra,
		PersistentPostRun: plat.postrunCobra,
	}
	rootCmd.PersistentFlags().StringVar(&plat.settings.OtelcolEndpoint, "otelcol_endpoint", "localhost:4317", "OpenTelemetry collector address")
	rootCmd.PersistentFlags().StringVar(&plat.settings.AdminBrokers, "admin_brokers", "localhost:9092", "Kafka brokers for admin messages like platform updates")
	rootCmd.PersistentFlags().UintVar(&plat.settings.AdminPingIntervalSecs, "admin_ping_interval_secs", 1, "Interval for producers to ping the management system")

	// admin sub command
	adminCmd := &cobra.Command{
		Use:   "admin",
		Short: "Admin service that manages topics and orchestration",
		Run:   plat.cobraAdminServe,
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
	portalReadCmd.PersistentFlags().StringVar(&plat.settings.PortalAddr, "portal_addr", "http://localhost:11371", "Address against which to make client requests")
	portalCmd.AddCommand(portalReadCmd)

	portalReadPlatformCmd := &cobra.Command{
		Use:   "platform",
		Short: "read the platform definition",
		Run:   plat.cobraPortalReadPlatform,
	}
	portalReadCmd.AddCommand(portalReadPlatformCmd)

	portalReadConfigCmd := &cobra.Command{
		Use:   "config",
		Short: "read the current config",
		Run:   plat.cobraPortalReadConfig,
	}
	portalReadCmd.AddCommand(portalReadConfigCmd)

	portalDecodeCmd := &cobra.Command{
		Use:   "decode",
		Short: "decode base64 opaque payloads by calling portal api",
	}
	portalCmd.AddCommand(portalDecodeCmd)
	portalDecodeCmd.PersistentFlags().StringVar(&plat.settings.PortalAddr, "portal_addr", "http://localhost:11371", "Address against which to make client requests")

	portalReadProducersCmd := &cobra.Command{
		Use:   "producers",
		Short: "read the active tracked producers",
		Run:   plat.cobraPortalReadProducers,
	}
	portalReadCmd.AddCommand(portalReadProducersCmd)

	portalCancelTxnCmd := &cobra.Command{
		Use:       "cancel",
		Short:     "Cancel APECS transaction",
		Args:      cobra.MinimumNArgs(1),
		ValidArgs: []string{"txn_id"},
		Run:       plat.cobraPortalCancelTxn,
	}
	portalCancelTxnCmd.PersistentFlags().StringVar(&plat.settings.PortalAddr, "portal_addr", "localhost:11381", "Address against which to make client requests")
	portalCmd.AddCommand(portalCancelTxnCmd)

	portalDecodeInstanceCmd := &cobra.Command{
		Use:       "instance concern base64_payload",
		Short:     "decode and print base64 payload",
		Run:       plat.cobraPortalDecodeInstance,
		Args:      cobra.MinimumNArgs(2),
		ValidArgs: []string{"concern", "base64_payload"},
	}
	portalDecodeCmd.AddCommand(portalDecodeInstanceCmd)

	portalServeCmd := &cobra.Command{
		Use:   "serve",
		Short: "Rocketcycle Portal Server",
		Long:  "Host rest api",
		Run:   plat.cobraPortalServe,
	}
	portalServeCmd.PersistentFlags().StringVar(&plat.settings.HttpAddr, "http_addr", ":11371", "Address to host http api")
	portalServeCmd.PersistentFlags().StringVar(&plat.settings.GrpcAddr, "grpc_addr", ":11381", "Address to host grpc api")
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
		Run:   plat.cobraConfigReplace,
	}
	configReplaceCmd.PersistentFlags().StringVarP(&plat.settings.ConfigFilePath, "config_file_path", "c", "./config.json", "Path to json file containing Config values")
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
		Run:       plat.cobraDecodeInstance,
		Args:      cobra.MinimumNArgs(2),
		ValidArgs: []string{"concern", "base64_payload"},
	}
	decodeCmd.AddCommand(decodeInstanceCmd)

	// process sub command
	procCmd := &cobra.Command{
		Use:   "process",
		Short: "APECS processing mode",
		Long:  "Runs a proc consumer against the partition specified",
		Run:   plat.cobraApecsConsumer,
	}
	procCmd.PersistentFlags().StringVar(&plat.settings.ConsumerBrokers, "consumer_brokers", "", "Kafka brokers against which to consume topic")
	procCmd.MarkPersistentFlagRequired("consumer_brokers")
	procCmd.PersistentFlags().StringVarP(&plat.settings.Topic, "topic", "t", "", "Topic to consume")
	procCmd.MarkPersistentFlagRequired("topic")
	procCmd.PersistentFlags().Int32VarP(&plat.settings.Partition, "partition", "p", -1, "Partition to consume")
	procCmd.MarkPersistentFlagRequired("partition")
	rootCmd.AddCommand(procCmd)

	// storage sub command
	storageCmd := &cobra.Command{
		Use:   "storage",
		Short: "APECS storage mode",
		Long:  "Runs a storage consumer against the partition specified",
		Run:   plat.cobraApecsConsumer,
	}
	storageCmd.PersistentFlags().StringVar(&plat.settings.ConsumerBrokers, "consumer_brokers", "", "Kafka brokers against which to consume topic")
	storageCmd.MarkPersistentFlagRequired("consumer_brokers")
	storageCmd.PersistentFlags().StringVarP(&plat.settings.Topic, "topic", "t", "", "Topic to consume")
	storageCmd.MarkPersistentFlagRequired("topic")
	storageCmd.PersistentFlags().Int32VarP(&plat.settings.Partition, "partition", "p", -1, "Partition to consume")
	storageCmd.MarkPersistentFlagRequired("partition")
	storageCmd.PersistentFlags().StringVar(&plat.settings.StorageTarget, "storage_target", "", "One of the named storage targets defined within platform config")
	storageCmd.MarkPersistentFlagRequired("storage_target")
	rootCmd.AddCommand(storageCmd)

	// secondary-storage sub command
	storageScndCmd := &cobra.Command{
		Use:   "storage-scnd",
		Short: "APECS secondary storage mode",
		Long:  "Runs a secondary storage consumer against the partition specified",
		Run:   plat.cobraApecsConsumer,
	}
	storageScndCmd.PersistentFlags().StringVar(&plat.settings.ConsumerBrokers, "consumer_brokers", "", "Kafka brokers against which to consume topic")
	storageScndCmd.MarkPersistentFlagRequired("consumer_brokers")
	storageScndCmd.PersistentFlags().StringVarP(&plat.settings.Topic, "topic", "t", "", "Topic to consume")
	storageScndCmd.MarkPersistentFlagRequired("topic")
	storageScndCmd.PersistentFlags().Int32VarP(&plat.settings.Partition, "partition", "p", -1, "Partition to consume")
	storageScndCmd.MarkPersistentFlagRequired("partition")
	storageScndCmd.PersistentFlags().StringVar(&plat.settings.StorageTarget, "storage_target", "", "One of the named storage targets defined within platform config")
	storageScndCmd.MarkPersistentFlagRequired("storage_target")
	rootCmd.AddCommand(storageScndCmd)

	// watch sub command
	watchCmd := &cobra.Command{
		Use:   "watch",
		Short: "APECS watch mode",
		Long:  "Runs a watch consumer against all error/complete topics",
		Run:   plat.cobraWatch,
	}
	watchCmd.PersistentFlags().BoolVarP(&plat.settings.WatchDecode, "decode", "d", false, "If set, will decode all Buffer objects when printing ApecsTxn messages")
	rootCmd.AddCommand(watchCmd)

	// run sub command
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run all topic consumer programs",
		Long:  "Orchestrates sub processes as specified by platform topics consumer programs",
		Run:   plat.cobraRun,
	}
	runCmd.PersistentFlags().BoolVarP(&plat.settings.WatchDecode, "decode", "d", false, "If set, will decode all Buffer objects when printing ApecsTxn messages")
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
		Run:   plat.cobraPlatReplace,
	}
	platReplaceCmd.PersistentFlags().StringVarP(&plat.settings.PlatformFilePath, "platform_file_path", "p", "./platform.json", "Path to json file containing platform configuration")
	platCmd.AddCommand(platReplaceCmd)

	for _, addtlCmd := range plat.cobraCommands {
		rootCmd.AddCommand(addtlCmd)
	}

	rootCmd.Execute()
}
