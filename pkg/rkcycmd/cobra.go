// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcycmd

import (
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rocketcycle/pkg/admin"
	"github.com/lachlanorr/rocketcycle/pkg/apecs"
	"github.com/lachlanorr/rocketcycle/pkg/mgmt"
	"github.com/lachlanorr/rocketcycle/pkg/portal"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/runner"
	"github.com/lachlanorr/rocketcycle/pkg/watch"
)

func (rkcyCmd *RkcyCmd) addPlatformFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&rkcyCmd.settings.Environment, "environment", "e", "", "Application defined environment name, e.g. dev, test, prod, etc")
	cmd.MarkPersistentFlagRequired("environment")

	cmd.PersistentFlags().StringVar(&rkcyCmd.settings.AdminBrokers, "admin_brokers", "localhost:9092", "Kafka brokers for admin messages like platform updates")

	cmd.PersistentFlags().UintVar(&rkcyCmd.settings.AdminPingIntervalSecs, "admin_ping_interval_secs", 10, "Kafka brokers for admin messages like platform updates")
}

func (rkcyCmd *RkcyCmd) addConsumerFlags(cmd *cobra.Command) {
	rkcyCmd.addPlatformFlags(cmd)

	cmd.PersistentFlags().StringVar(&rkcyCmd.settings.ConsumerBrokers, "consumer_brokers", "", "Kafka brokers against which to consume response topic")
	cmd.MarkPersistentFlagRequired("consumer_brokers")

	cmd.PersistentFlags().StringVarP(&rkcyCmd.settings.Topic, "topic", "t", "", "Response topic for synchronous apecs transactions")
	cmd.MarkPersistentFlagRequired("topic")

	cmd.PersistentFlags().Int32VarP(&rkcyCmd.settings.Partition, "partition", "p", -1, "Partition index to consume")
	cmd.MarkPersistentFlagRequired("partition")
}

func (rkcyCmd *RkcyCmd) addEdgeFlags(cmd *cobra.Command) {
	rkcyCmd.addConsumerFlags(cmd)

	cmd.PersistentFlags().BoolVar(&rkcyCmd.settings.Edge, "edge", false, "Consume topic as response topic for ExecuteTxnSync operations")
}

func (rkcyCmd *RkcyCmd) runCobra() {
	rootCmd := &cobra.Command{
		Use:              rkcyCmd.platform,
		Short:            "Rocketcycle Platform - " + rkcyCmd.platform,
		PersistentPreRun: rkcyCmd.prerunCobra,
	}
	rootCmd.PersistentFlags().StringVar(&rkcyCmd.settings.OtelcolEndpoint, "otelcol_endpoint", "localhost:4317", "OpenTelemetry collector address")

	// admin sub command
	adminCmd := &cobra.Command{
		Use:   "admin",
		Short: "Admin service that manages topics and orchestration",
		Run: func(*cobra.Command, []string) {
			admin.Start(rkcyCmd.ctx, rkcyCmd.plat, rkcyCmd.settings.OtelcolEndpoint)
		},
	}
	rkcyCmd.addPlatformFlags(adminCmd)
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
	portalReadCmd.PersistentFlags().StringVar(&rkcyCmd.settings.HttpAddr, "http_addr", "http://localhost:11371", "Address against which to make client requests")
	portalCmd.AddCommand(portalReadCmd)

	portalReadPlatformCmd := &cobra.Command{
		Use:   "platform",
		Short: "read the platform definition",
		Run: func(*cobra.Command, []string) {
			portal.ClientReadPlatform(rkcyCmd.settings.HttpAddr)
		},
	}
	portalReadCmd.AddCommand(portalReadPlatformCmd)

	portalReadConfigCmd := &cobra.Command{
		Use:   "config",
		Short: "read the current config",
		Run: func(*cobra.Command, []string) {
			portal.ClientReadConfig(rkcyCmd.settings.HttpAddr)
		},
	}
	portalReadCmd.AddCommand(portalReadConfigCmd)

	portalReadProducersCmd := &cobra.Command{
		Use:   "producers",
		Short: "read the active tracked producers",
		Run: func(*cobra.Command, []string) {
			portal.ClientReadProducers(rkcyCmd.settings.HttpAddr)
		},
	}
	portalReadCmd.AddCommand(portalReadProducersCmd)

	portalDecodeCmd := &cobra.Command{
		Use:   "decode",
		Short: "decode base64 opaque payloads by calling portal api",
	}
	portalCmd.AddCommand(portalDecodeCmd)
	portalDecodeCmd.PersistentFlags().StringVar(&rkcyCmd.settings.HttpAddr, "http_addr", "http://localhost:11371", "Address against which to make client requests")

	portalDecodeInstanceCmd := &cobra.Command{
		Use:       "instance concern base64_payload",
		Short:     "decode and print base64 payload",
		Args:      cobra.MinimumNArgs(2),
		ValidArgs: []string{"concern", "base64_payload"},
		Run: func(cmd *cobra.Command, args []string) {
			portal.ClientDecodeInstance(rkcyCmd.settings.HttpAddr, args[0], args[1])
		},
	}
	portalDecodeCmd.AddCommand(portalDecodeInstanceCmd)

	portalCancelTxnCmd := &cobra.Command{
		Use:       "cancel",
		Short:     "Cancel APECS transaction",
		Args:      cobra.MinimumNArgs(1),
		ValidArgs: []string{"txn_id"},
		Run: func(cmd *cobra.Command, args []string) {
			portal.ClientCancelTxn(rkcyCmd.settings.GrpcAddr, args[0])
		},
	}
	portalCancelTxnCmd.PersistentFlags().StringVar(&rkcyCmd.settings.GrpcAddr, "grpc_addr", "localhost:11381", "Address against which to make client requests")
	portalCmd.AddCommand(portalCancelTxnCmd)

	portalServeCmd := &cobra.Command{
		Use:   "serve",
		Short: "Rocketcycle Portal Server",
		Long:  "Host rest api",
		Run: func(*cobra.Command, []string) {
			portal.Serve(rkcyCmd.ctx, rkcyCmd.plat, rkcyCmd.settings.HttpAddr, rkcyCmd.settings.GrpcAddr)
		},
	}
	rkcyCmd.addPlatformFlags(portalServeCmd)
	portalServeCmd.PersistentFlags().StringVar(&rkcyCmd.settings.HttpAddr, "http_addr", ":11371", "Address to host http api")
	portalServeCmd.PersistentFlags().StringVar(&rkcyCmd.settings.GrpcAddr, "grpc_addr", ":11381", "Address to host grpc api")
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
		Run: func(*cobra.Command, []string) {
			mgmt.ConfigReplace(
				rkcyCmd.ctx,
				rkcyCmd.plat.StreamProvider(),
				rkcyCmd.plat.Args.Platform,
				rkcyCmd.plat.Args.Environment,
				rkcyCmd.plat.Args.AdminBrokers,
				rkcyCmd.settings.ConfigFilePath,
			)
		},
	}
	rkcyCmd.addPlatformFlags(configReplaceCmd)
	configReplaceCmd.PersistentFlags().StringVarP(&rkcyCmd.settings.ConfigFilePath, "config_file_path", "c", "./config.json", "Path to json file containing Config values")
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
		Args:      cobra.MinimumNArgs(2),
		ValidArgs: []string{"concern", "base64_payload"},
		Run: func(cmd *cobra.Command, args []string) {
			rkcy.DecodeInstance(rkcyCmd.plat.ConcernHandlers, args[0], args[1])
		},
	}
	decodeCmd.AddCommand(decodeInstanceCmd)

	// process sub command
	procCmd := &cobra.Command{
		Use:   "process",
		Short: "APECS processing mode",
		Long:  "Runs a proc consumer against the partition specified",
		Run: func(*cobra.Command, []string) {
			apecs.StartApecsConsumer(
				rkcyCmd.ctx,
				rkcyCmd.plat,
				"",
				rkcyCmd.settings.ConsumerBrokers,
				rkcyCmd.settings.Topic,
				rkcyCmd.settings.Partition,
				rkcyCmd.bailoutCh,
			)
		},
	}
	rkcyCmd.addConsumerFlags(procCmd)
	rootCmd.AddCommand(procCmd)

	// storage sub command
	storageCmd := &cobra.Command{
		Use:   "storage",
		Short: "APECS storage mode",
		Long:  "Runs a storage consumer against the partition specified",
		Run: func(*cobra.Command, []string) {
			apecs.StartApecsConsumer(
				rkcyCmd.ctx,
				rkcyCmd.plat,
				rkcyCmd.settings.StorageTarget,
				rkcyCmd.settings.ConsumerBrokers,
				rkcyCmd.settings.Topic,
				rkcyCmd.settings.Partition,
				rkcyCmd.bailoutCh,
			)
		},
	}
	rkcyCmd.addConsumerFlags(storageCmd)
	storageCmd.PersistentFlags().StringVar(&rkcyCmd.settings.StorageTarget, "storage_target", "", "One of the named storage targets defined within platform config")
	storageCmd.MarkPersistentFlagRequired("storage_target")
	rootCmd.AddCommand(storageCmd)

	// secondary-storage sub command
	storageScndCmd := &cobra.Command{
		Use:   "storage-scnd",
		Short: "APECS secondary storage mode",
		Long:  "Runs a secondary storage consumer against the partition specified",
		Run: func(*cobra.Command, []string) {
			apecs.StartApecsConsumer(
				rkcyCmd.ctx,
				rkcyCmd.plat,
				rkcyCmd.settings.StorageTarget,
				rkcyCmd.settings.ConsumerBrokers,
				rkcyCmd.settings.Topic,
				rkcyCmd.settings.Partition,
				rkcyCmd.bailoutCh,
			)
		},
	}
	rkcyCmd.addConsumerFlags(storageScndCmd)
	storageScndCmd.PersistentFlags().StringVar(&rkcyCmd.settings.StorageTarget, "storage_target", "", "One of the named storage targets defined within platform config")
	storageScndCmd.MarkPersistentFlagRequired("storage_target")
	rootCmd.AddCommand(storageScndCmd)

	// watch sub command
	watchCmd := &cobra.Command{
		Use:   "watch",
		Short: "APECS watch mode",
		Long:  "Runs a watch consumer against all error/complete topics",
		Run: func(*cobra.Command, []string) {
			watch.Start(
				rkcyCmd.ctx,
				rkcyCmd.plat.WaitGroup(),
				rkcyCmd.plat.StreamProvider(),
				rkcyCmd.plat.Args.Platform,
				rkcyCmd.plat.Args.Environment,
				rkcyCmd.plat.Args.AdminBrokers,
				rkcyCmd.plat.ConcernHandlers,
				rkcyCmd.settings.WatchDecode,
			)
		},
	}
	rkcyCmd.addPlatformFlags(watchCmd)
	watchCmd.PersistentFlags().BoolVarP(&rkcyCmd.settings.WatchDecode, "decode", "d", false, "If set, will decode all Buffer objects when printing ApecsTxn messages")
	rootCmd.AddCommand(watchCmd)

	// run sub command
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run all topic consumer programs",
		Long:  "Orchestrates sub processes as specified by platform topics consumer programs",
		Run: func(*cobra.Command, []string) {
			runner.Start(
				rkcyCmd.ctx,
				rkcyCmd.plat.WaitGroup(),
				rkcyCmd.plat.StreamProvider(),
				rkcyCmd.plat.Args.Platform,
				rkcyCmd.plat.Args.Environment,
				rkcyCmd.plat.Args.AdminBrokers,
				rkcyCmd.settings.OtelcolEndpoint,
				rkcyCmd.settings.WatchDecode,
			)
		},
	}
	rkcyCmd.addPlatformFlags(runCmd)
	runCmd.PersistentFlags().BoolVarP(&rkcyCmd.settings.WatchDecode, "decode", "d", false, "If set, will decode all Buffer objects when printing ApecsTxn messages")
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
		Run: func(*cobra.Command, []string) {
			mgmt.PlatformReplace(
				rkcyCmd.ctx,
				rkcyCmd.plat.StreamProvider(),
				rkcyCmd.plat.Args.Platform,
				rkcyCmd.plat.Args.Environment,
				rkcyCmd.plat.Args.AdminBrokers,
				rkcyCmd.settings.PlatformFilePath,
			)
		},
	}
	rkcyCmd.addPlatformFlags(platReplaceCmd)
	platReplaceCmd.PersistentFlags().StringVarP(&rkcyCmd.settings.PlatformFilePath, "platform_file_path", "p", "./platform.json", "Path to json file containing platform configuration")
	platCmd.AddCommand(platReplaceCmd)

	for _, cmd := range rkcyCmd.customCobraCommands {
		rootCmd.AddCommand(cmd)
	}

	rootCmd.Execute()
}
