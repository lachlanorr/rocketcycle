// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package runner

import (
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rocketcycle/pkg/admin"
	"github.com/lachlanorr/rocketcycle/pkg/apecs"
	"github.com/lachlanorr/rocketcycle/pkg/mgmt"
	"github.com/lachlanorr/rocketcycle/pkg/portal"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/watch"
)

func (rkcycmd *RkcyCmd) addPlatformFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&rkcycmd.settings.Environment, "environment", "e", "", "Application defined environment name, e.g. dev, test, prod, etc")
	cmd.MarkPersistentFlagRequired("environment")

	cmd.PersistentFlags().StringVar(&rkcycmd.settings.AdminBrokers, "admin_brokers", "localhost:9092", "Kafka brokers for admin messages like platform updates")

	cmd.PersistentFlags().UintVar(&rkcycmd.settings.AdminPingIntervalSecs, "admin_ping_interval_secs", 10, "Kafka brokers for admin messages like platform updates")
}

func (rkcycmd *RkcyCmd) addConsumerFlags(cmd *cobra.Command) {
	rkcycmd.addPlatformFlags(cmd)

	cmd.PersistentFlags().StringVar(&rkcycmd.settings.ConsumerBrokers, "consumer_brokers", "", "Kafka brokers against which to consume response topic")
	cmd.MarkPersistentFlagRequired("consumer_brokers")

	cmd.PersistentFlags().StringVarP(&rkcycmd.settings.Topic, "topic", "t", "", "Response topic for synchronous apecs transactions")
	cmd.MarkPersistentFlagRequired("topic")

	cmd.PersistentFlags().Int32VarP(&rkcycmd.settings.Partition, "partition", "p", -1, "Partition index to consume")
	cmd.MarkPersistentFlagRequired("partition")
}

func (rkcycmd *RkcyCmd) addEdgeFlags(cmd *cobra.Command) {
	rkcycmd.addConsumerFlags(cmd)

	cmd.PersistentFlags().BoolVar(&rkcycmd.settings.Edge, "edge", false, "Consume topic as response topic for ExecuteTxnSync operations")
}

func (rkcycmd *RkcyCmd) BuildCobraCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:              rkcycmd.platform,
		Short:            "Rocketcycle Platform - " + rkcycmd.platform,
		PersistentPreRun: rkcycmd.prerunCobra,
	}
	rootCmd.PersistentFlags().StringVar(&rkcycmd.settings.OtelcolEndpoint, "otelcol_endpoint", "offline", "OpenTelemetry collector address, e.g. localhost:4317")
	rootCmd.PersistentFlags().StringVar(&rkcycmd.settings.StreamType, "stream", "auto", "Type of StreamProvider to use")

	// admin sub command
	adminCmd := &cobra.Command{
		Use:   "admin",
		Short: "Admin service that manages topics and orchestration",
		Run: func(cmd *cobra.Command, args []string) {
			admin.Start(cmd.Context(), rkcycmd.plat, rkcycmd.settings.OtelcolEndpoint)
		},
	}
	rkcycmd.addPlatformFlags(adminCmd)
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
	portalReadCmd.PersistentFlags().StringVar(&rkcycmd.settings.HttpAddr, "http_addr", "http://localhost:11371", "Address against which to make client requests")
	portalCmd.AddCommand(portalReadCmd)

	portalReadPlatformCmd := &cobra.Command{
		Use:   "platform",
		Short: "read the platform definition",
		Run: func(*cobra.Command, []string) {
			portal.ClientReadPlatform(rkcycmd.settings.HttpAddr)
		},
	}
	portalReadCmd.AddCommand(portalReadPlatformCmd)

	portalReadConfigCmd := &cobra.Command{
		Use:   "config",
		Short: "read the current config",
		Run: func(*cobra.Command, []string) {
			portal.ClientReadConfig(rkcycmd.settings.HttpAddr)
		},
	}
	portalReadCmd.AddCommand(portalReadConfigCmd)

	portalReadProducersCmd := &cobra.Command{
		Use:   "producers",
		Short: "read the active tracked producers",
		Run: func(*cobra.Command, []string) {
			portal.ClientReadProducers(rkcycmd.settings.HttpAddr)
		},
	}
	portalReadCmd.AddCommand(portalReadProducersCmd)

	portalDecodeCmd := &cobra.Command{
		Use:   "decode",
		Short: "decode base64 opaque payloads by calling portal api",
	}
	portalCmd.AddCommand(portalDecodeCmd)
	portalDecodeCmd.PersistentFlags().StringVar(&rkcycmd.settings.HttpAddr, "http_addr", "http://localhost:11371", "Address against which to make client requests")

	portalDecodeInstanceCmd := &cobra.Command{
		Use:       "instance concern base64_payload",
		Short:     "decode and print base64 payload",
		Args:      cobra.MinimumNArgs(2),
		ValidArgs: []string{"concern", "base64_payload"},
		Run: func(cmd *cobra.Command, args []string) {
			portal.ClientDecodeInstance(rkcycmd.settings.HttpAddr, args[0], args[1])
		},
	}
	portalDecodeCmd.AddCommand(portalDecodeInstanceCmd)

	portalCancelTxnCmd := &cobra.Command{
		Use:       "cancel",
		Short:     "Cancel APECS transaction",
		Args:      cobra.MinimumNArgs(1),
		ValidArgs: []string{"txn_id"},
		Run: func(cmd *cobra.Command, args []string) {
			portal.ClientCancelTxn(rkcycmd.settings.GrpcAddr, args[0])
		},
	}
	portalCancelTxnCmd.PersistentFlags().StringVar(&rkcycmd.settings.GrpcAddr, "grpc_addr", "localhost:11381", "Address against which to make client requests")
	portalCmd.AddCommand(portalCancelTxnCmd)

	portalServeCmd := &cobra.Command{
		Use:   "serve",
		Short: "Rocketcycle Portal Server",
		Long:  "Host rest api",
		Run: func(cmd *cobra.Command, args []string) {
			portal.Serve(cmd.Context(), rkcycmd.plat, rkcycmd.settings.HttpAddr, rkcycmd.settings.GrpcAddr)
		},
	}
	rkcycmd.addPlatformFlags(portalServeCmd)
	portalServeCmd.PersistentFlags().StringVar(&rkcycmd.settings.HttpAddr, "http_addr", ":11371", "Address to host http api")
	portalServeCmd.PersistentFlags().StringVar(&rkcycmd.settings.GrpcAddr, "grpc_addr", ":11381", "Address to host grpc api")
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
		Run: func(cmd *cobra.Command, args []string) {
			mgmt.ConfigReplaceFromFile(
				cmd.Context(),
				rkcycmd.plat.StreamProvider(),
				rkcycmd.plat.Args.Platform,
				rkcycmd.plat.Args.Environment,
				rkcycmd.plat.Args.AdminBrokers,
				rkcycmd.settings.ConfigFilePath,
			)
		},
	}
	rkcycmd.addPlatformFlags(configReplaceCmd)
	configReplaceCmd.PersistentFlags().StringVar(&rkcycmd.settings.ConfigFilePath, "config_file_path", "./config.json", "Path to json file containing Config values")
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
			rkcy.DecodeInstance(rkcycmd.clientCode.ConcernHandlers, args[0], args[1])
		},
	}
	decodeCmd.AddCommand(decodeInstanceCmd)

	// process sub command
	procCmd := &cobra.Command{
		Use:   "process",
		Short: "APECS processing mode",
		Long:  "Runs a proc consumer against the partition specified",
		Run: func(cmd *cobra.Command, args []string) {
			apecs.StartApecsConsumer(
				cmd.Context(),
				rkcycmd.plat,
				"",
				rkcycmd.settings.ConsumerBrokers,
				rkcycmd.settings.Topic,
				rkcycmd.settings.Partition,
				rkcycmd.bailoutCh,
			)
		},
	}
	rkcycmd.addConsumerFlags(procCmd)
	rootCmd.AddCommand(procCmd)

	// storage sub command
	storageCmd := &cobra.Command{
		Use:   "storage",
		Short: "APECS storage mode",
		Long:  "Runs a storage consumer against the partition specified",
		Run: func(cmd *cobra.Command, args []string) {
			apecs.StartApecsConsumer(
				cmd.Context(),
				rkcycmd.plat,
				rkcycmd.settings.StorageTarget,
				rkcycmd.settings.ConsumerBrokers,
				rkcycmd.settings.Topic,
				rkcycmd.settings.Partition,
				rkcycmd.bailoutCh,
			)
		},
	}
	rkcycmd.addConsumerFlags(storageCmd)
	storageCmd.PersistentFlags().StringVar(&rkcycmd.settings.StorageTarget, "storage_target", "", "One of the named storage targets defined within platform config")
	storageCmd.MarkPersistentFlagRequired("storage_target")
	rootCmd.AddCommand(storageCmd)

	// secondary-storage sub command
	storageScndCmd := &cobra.Command{
		Use:   "storage-scnd",
		Short: "APECS secondary storage mode",
		Long:  "Runs a secondary storage consumer against the partition specified",
		Run: func(cmd *cobra.Command, args []string) {
			apecs.StartApecsConsumer(
				cmd.Context(),
				rkcycmd.plat,
				rkcycmd.settings.StorageTarget,
				rkcycmd.settings.ConsumerBrokers,
				rkcycmd.settings.Topic,
				rkcycmd.settings.Partition,
				rkcycmd.bailoutCh,
			)
		},
	}
	rkcycmd.addConsumerFlags(storageScndCmd)
	storageScndCmd.PersistentFlags().StringVar(&rkcycmd.settings.StorageTarget, "storage_target", "", "One of the named storage targets defined within platform config")
	storageScndCmd.MarkPersistentFlagRequired("storage_target")
	rootCmd.AddCommand(storageScndCmd)

	// watch sub command
	watchCmd := &cobra.Command{
		Use:   "watch",
		Short: "APECS watch mode",
		Long:  "Runs a watch consumer against all error/complete topics",
		Run: func(cmd *cobra.Command, args []string) {
			watch.Start(
				cmd.Context(),
				rkcycmd.plat.WaitGroup(),
				rkcycmd.plat.StreamProvider(),
				rkcycmd.plat.Args.Platform,
				rkcycmd.plat.Args.Environment,
				rkcycmd.plat.Args.AdminBrokers,
				rkcycmd.clientCode.ConcernHandlers,
				rkcycmd.settings.WatchDecode,
			)
		},
	}
	rkcycmd.addPlatformFlags(watchCmd)
	watchCmd.PersistentFlags().BoolVarP(&rkcycmd.settings.WatchDecode, "decode", "d", false, "If set, will decode all Buffer objects when printing ApecsTxn messages")
	rootCmd.AddCommand(watchCmd)

	// run sub command
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run all topic consumer programs",
		Long:  "Orchestrates sub processes as specified by platform topics consumer programs",
		Run: func(cmd *cobra.Command, args []string) {
			rkcycmd.plat.WaitGroup().Add(1)
			rkcycmd.RunConsumerPrograms(
				cmd.Context(),
				rkcycmd.plat.WaitGroup(),
				rkcycmd.plat.StreamProvider(),
				rkcycmd.plat.Args.Platform,
				rkcycmd.plat.Args.Environment,
				rkcycmd.plat.Args.AdminBrokers,
				rkcycmd.settings.OtelcolEndpoint,
				rkcycmd.settings.WatchDecode,
			)
		},
	}
	rkcycmd.addPlatformFlags(runCmd)
	runCmd.PersistentFlags().BoolVarP(&rkcycmd.settings.WatchDecode, "decode", "d", false, "If set, will decode all Buffer objects when printing ApecsTxn messages")
	runCmd.PersistentFlags().StringVar(&rkcycmd.settings.RunnerType, "runner", "process", "Type of runner to use to manage distributed processes or routines")
	runCmd.PersistentFlags().StringVar(&rkcycmd.settings.PlatformFilePath, "platform_file_path", "./platform.json", "Path to json file containing platform configuration")
	runCmd.PersistentFlags().StringVar(&rkcycmd.settings.ConfigFilePath, "config_file_path", "./config.json", "Path to json file containing Config values")
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
		Run: func(cmd *cobra.Command, args []string) {
			mgmt.PlatformReplaceFromFile(
				cmd.Context(),
				rkcycmd.plat.StreamProvider(),
				rkcycmd.plat.Args.Platform,
				rkcycmd.plat.Args.Environment,
				rkcycmd.plat.Args.AdminBrokers,
				rkcycmd.settings.PlatformFilePath,
			)
		},
	}
	rkcycmd.addPlatformFlags(platReplaceCmd)
	platReplaceCmd.PersistentFlags().StringVar(&rkcycmd.settings.PlatformFilePath, "platform_file_path", "./platform.json", "Path to json file containing platform configuration")
	platCmd.AddCommand(platReplaceCmd)

	for _, custCmdFunc := range rkcycmd.customCommandFuncs {
		rootCmd.AddCommand(custCmdFunc(rkcycmd))
	}

	return rootCmd
}
