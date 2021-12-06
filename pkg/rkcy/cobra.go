// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func (kplat *KafkaPlatform) prerunCobra(cmd *cobra.Command, args []string) {
	var err error
	kplat.telem, err = NewTelemetry(context.Background(), kplat.settings.OtelcolEndpoint)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to NewTelemetry")
	}

	kplat.rawProducer = NewRawProducer(kplat.telem)
}

func (kplat *KafkaPlatform) postrunCobra(cmd *cobra.Command, args []string) {
	if kplat.telem != nil {
		kplat.telem.Close()
	}
}

func (kplat *KafkaPlatform) runCobra() {
	rootCmd := &cobra.Command{
		Use:               kplat.name,
		Short:             "Rocketcycle Platform - " + kplat.name,
		PersistentPreRun:  kplat.prerunCobra,
		PersistentPostRun: kplat.postrunCobra,
	}
	rootCmd.PersistentFlags().StringVar(&kplat.settings.OtelcolEndpoint, "otelcol_endpoint", "localhost:4317", "OpenTelemetry collector address")
	rootCmd.PersistentFlags().StringVar(&kplat.settings.AdminBrokers, "admin_brokers", "localhost:9092", "Kafka brokers for admin messages like platform updates")
	rootCmd.PersistentFlags().UintVar(&kplat.settings.AdminPingIntervalSecs, "admin_ping_interval_secs", 1, "Interval for producers to ping the management system")

	// admin sub command
	adminCmd := &cobra.Command{
		Use:   "admin",
		Short: "Admin service that manages topics and orchestration",
		Run:   kplat.cobraAdminServe,
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
	portalReadCmd.PersistentFlags().StringVar(&kplat.settings.PortalAddr, "portal_addr", "http://localhost:11371", "Address against which to make client requests")
	portalCmd.AddCommand(portalReadCmd)

	portalReadPlatformCmd := &cobra.Command{
		Use:   "platform",
		Short: "read the platform definition",
		Run:   kplat.cobraPortalReadPlatform,
	}
	portalReadCmd.AddCommand(portalReadPlatformCmd)

	portalReadConfigCmd := &cobra.Command{
		Use:   "config",
		Short: "read the current config",
		Run:   kplat.cobraPortalReadConfig,
	}
	portalReadCmd.AddCommand(portalReadConfigCmd)

	portalDecodeCmd := &cobra.Command{
		Use:   "decode",
		Short: "decode base64 opaque payloads by calling portal api",
	}
	portalCmd.AddCommand(portalDecodeCmd)
	portalDecodeCmd.PersistentFlags().StringVar(&kplat.settings.PortalAddr, "portal_addr", "http://localhost:11371", "Address against which to make client requests")

	portalReadProducersCmd := &cobra.Command{
		Use:   "producers",
		Short: "read the active tracked producers",
		Run:   kplat.cobraPortalReadProducers,
	}
	portalReadCmd.AddCommand(portalReadProducersCmd)

	portalCancelTxnCmd := &cobra.Command{
		Use:       "cancel",
		Short:     "Cancel APECS transaction",
		Args:      cobra.MinimumNArgs(1),
		ValidArgs: []string{"txn_id"},
		Run:       kplat.cobraPortalCancelTxn,
	}
	portalCancelTxnCmd.PersistentFlags().StringVar(&kplat.settings.PortalAddr, "portal_addr", "localhost:11381", "Address against which to make client requests")
	portalCmd.AddCommand(portalCancelTxnCmd)

	portalDecodeInstanceCmd := &cobra.Command{
		Use:       "instance concern base64_payload",
		Short:     "decode and print base64 payload",
		Run:       kplat.cobraPortalDecodeInstance,
		Args:      cobra.MinimumNArgs(2),
		ValidArgs: []string{"concern", "base64_payload"},
	}
	portalDecodeCmd.AddCommand(portalDecodeInstanceCmd)

	portalServeCmd := &cobra.Command{
		Use:   "serve",
		Short: "Rocketcycle Portal Server",
		Long:  "Host rest api",
		Run:   kplat.cobraPortalServe,
	}
	portalServeCmd.PersistentFlags().StringVar(&kplat.settings.HttpAddr, "http_addr", ":11371", "Address to host http api")
	portalServeCmd.PersistentFlags().StringVar(&kplat.settings.GrpcAddr, "grpc_addr", ":11381", "Address to host grpc api")
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
		Run:   kplat.cobraConfigReplace,
	}
	configReplaceCmd.PersistentFlags().StringVarP(&kplat.settings.ConfigFilePath, "config_file_path", "c", "./config.json", "Path to json file containing Config values")
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
		Run:       kplat.cobraDecodeInstance,
		Args:      cobra.MinimumNArgs(2),
		ValidArgs: []string{"concern", "base64_payload"},
	}
	decodeCmd.AddCommand(decodeInstanceCmd)

	// process sub command
	procCmd := &cobra.Command{
		Use:   "process",
		Short: "APECS processing mode",
		Long:  "Runs a proc consumer against the partition specified",
		Run:   kplat.cobraApecsConsumer,
	}
	procCmd.PersistentFlags().StringVar(&kplat.settings.ConsumerBrokers, "consumer_brokers", "", "Kafka brokers against which to consume topic")
	procCmd.MarkPersistentFlagRequired("consumer_brokers")
	procCmd.PersistentFlags().StringVarP(&kplat.settings.Topic, "topic", "t", "", "Topic to consume")
	procCmd.MarkPersistentFlagRequired("topic")
	procCmd.PersistentFlags().Int32VarP(&kplat.settings.Partition, "partition", "p", -1, "Partition to consume")
	procCmd.MarkPersistentFlagRequired("partition")
	rootCmd.AddCommand(procCmd)

	// storage sub command
	storageCmd := &cobra.Command{
		Use:   "storage",
		Short: "APECS storage mode",
		Long:  "Runs a storage consumer against the partition specified",
		Run:   kplat.cobraApecsConsumer,
	}
	storageCmd.PersistentFlags().StringVar(&kplat.settings.ConsumerBrokers, "consumer_brokers", "", "Kafka brokers against which to consume topic")
	storageCmd.MarkPersistentFlagRequired("consumer_brokers")
	storageCmd.PersistentFlags().StringVarP(&kplat.settings.Topic, "topic", "t", "", "Topic to consume")
	storageCmd.MarkPersistentFlagRequired("topic")
	storageCmd.PersistentFlags().Int32VarP(&kplat.settings.Partition, "partition", "p", -1, "Partition to consume")
	storageCmd.MarkPersistentFlagRequired("partition")
	storageCmd.PersistentFlags().StringVar(&kplat.settings.StorageTarget, "storage_target", "", "One of the named storage targets defined within platform config")
	storageCmd.MarkPersistentFlagRequired("storage_target")
	rootCmd.AddCommand(storageCmd)

	// secondary-storage sub command
	storageScndCmd := &cobra.Command{
		Use:   "storage-scnd",
		Short: "APECS secondary storage mode",
		Long:  "Runs a secondary storage consumer against the partition specified",
		Run:   kplat.cobraApecsConsumer,
	}
	storageScndCmd.PersistentFlags().StringVar(&kplat.settings.ConsumerBrokers, "consumer_brokers", "", "Kafka brokers against which to consume topic")
	storageScndCmd.MarkPersistentFlagRequired("consumer_brokers")
	storageScndCmd.PersistentFlags().StringVarP(&kplat.settings.Topic, "topic", "t", "", "Topic to consume")
	storageScndCmd.MarkPersistentFlagRequired("topic")
	storageScndCmd.PersistentFlags().Int32VarP(&kplat.settings.Partition, "partition", "p", -1, "Partition to consume")
	storageScndCmd.MarkPersistentFlagRequired("partition")
	storageScndCmd.PersistentFlags().StringVar(&kplat.settings.StorageTarget, "storage_target", "", "One of the named storage targets defined within platform config")
	storageScndCmd.MarkPersistentFlagRequired("storage_target")
	rootCmd.AddCommand(storageScndCmd)

	// watch sub command
	watchCmd := &cobra.Command{
		Use:   "watch",
		Short: "APECS watch mode",
		Long:  "Runs a watch consumer against all error/complete topics",
		Run:   kplat.cobraWatch,
	}
	watchCmd.PersistentFlags().BoolVarP(&kplat.settings.WatchDecode, "decode", "d", false, "If set, will decode all Buffer objects when printing ApecsTxn messages")
	rootCmd.AddCommand(watchCmd)

	// run sub command
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run all topic consumer programs",
		Long:  "Orchestrates sub processes as specified by platform topics consumer programs",
		Run:   kplat.cobraRun,
	}
	runCmd.PersistentFlags().BoolVarP(&kplat.settings.WatchDecode, "decode", "d", false, "If set, will decode all Buffer objects when printing ApecsTxn messages")
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
		Run:   kplat.cobraPlatReplace,
	}
	platReplaceCmd.PersistentFlags().StringVarP(&kplat.settings.PlatformFilePath, "platform_file_path", "p", "./platform.json", "Path to json file containing platform configuration")
	platCmd.AddCommand(platReplaceCmd)

	for _, addtlCmd := range kplat.cobraCommands {
		rootCmd.AddCommand(addtlCmd)
	}

	rootCmd.Execute()
}
