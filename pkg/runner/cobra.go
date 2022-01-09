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

func (runner *Runner) addPlatformFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&runner.settings.Environment, "environment", "e", "", "Application defined environment name, e.g. dev, test, prod, etc")
	cmd.MarkPersistentFlagRequired("environment")

	cmd.PersistentFlags().StringVar(&runner.settings.AdminBrokers, "admin_brokers", "localhost:9092", "Kafka brokers for admin messages like platform updates")

	cmd.PersistentFlags().UintVar(&runner.settings.AdminPingIntervalSecs, "admin_ping_interval_secs", 10, "Kafka brokers for admin messages like platform updates")
}

func (runner *Runner) addConsumerFlags(cmd *cobra.Command) {
	runner.addPlatformFlags(cmd)

	cmd.PersistentFlags().StringVar(&runner.settings.ConsumerBrokers, "consumer_brokers", "", "Kafka brokers against which to consume response topic")
	cmd.MarkPersistentFlagRequired("consumer_brokers")

	cmd.PersistentFlags().StringVarP(&runner.settings.Topic, "topic", "t", "", "Response topic for synchronous apecs transactions")
	cmd.MarkPersistentFlagRequired("topic")

	cmd.PersistentFlags().Int32VarP(&runner.settings.Partition, "partition", "p", -1, "Partition index to consume")
	cmd.MarkPersistentFlagRequired("partition")
}

func (runner *Runner) addEdgeFlags(cmd *cobra.Command) {
	runner.addConsumerFlags(cmd)

	cmd.PersistentFlags().BoolVar(&runner.settings.Edge, "edge", false, "Consume topic as response topic for ExecuteTxnSync operations")
}

func (runner *Runner) BuildCobraCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:              runner.platform,
		Short:            "Rocketcycle Platform - " + runner.platform,
		PersistentPreRun: runner.prerunCobra,
	}
	rootCmd.PersistentFlags().StringVar(&runner.settings.OtelcolEndpoint, "otelcol_endpoint", "offline", "OpenTelemetry collector address")
	rootCmd.PersistentFlags().StringVar(&runner.settings.StreamType, "stream", "kafka", "Type of StreamProvider to use")

	// admin sub command
	adminCmd := &cobra.Command{
		Use:   "admin",
		Short: "Admin service that manages topics and orchestration",
		Run: func(cmd *cobra.Command, args []string) {
			admin.Start(cmd.Context(), runner.plat, runner.settings.OtelcolEndpoint)
		},
	}
	runner.addPlatformFlags(adminCmd)
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
	portalReadCmd.PersistentFlags().StringVar(&runner.settings.HttpAddr, "http_addr", "http://localhost:11371", "Address against which to make client requests")
	portalCmd.AddCommand(portalReadCmd)

	portalReadPlatformCmd := &cobra.Command{
		Use:   "platform",
		Short: "read the platform definition",
		Run: func(*cobra.Command, []string) {
			portal.ClientReadPlatform(runner.settings.HttpAddr)
		},
	}
	portalReadCmd.AddCommand(portalReadPlatformCmd)

	portalReadConfigCmd := &cobra.Command{
		Use:   "config",
		Short: "read the current config",
		Run: func(*cobra.Command, []string) {
			portal.ClientReadConfig(runner.settings.HttpAddr)
		},
	}
	portalReadCmd.AddCommand(portalReadConfigCmd)

	portalReadProducersCmd := &cobra.Command{
		Use:   "producers",
		Short: "read the active tracked producers",
		Run: func(*cobra.Command, []string) {
			portal.ClientReadProducers(runner.settings.HttpAddr)
		},
	}
	portalReadCmd.AddCommand(portalReadProducersCmd)

	portalDecodeCmd := &cobra.Command{
		Use:   "decode",
		Short: "decode base64 opaque payloads by calling portal api",
	}
	portalCmd.AddCommand(portalDecodeCmd)
	portalDecodeCmd.PersistentFlags().StringVar(&runner.settings.HttpAddr, "http_addr", "http://localhost:11371", "Address against which to make client requests")

	portalDecodeInstanceCmd := &cobra.Command{
		Use:       "instance concern base64_payload",
		Short:     "decode and print base64 payload",
		Args:      cobra.MinimumNArgs(2),
		ValidArgs: []string{"concern", "base64_payload"},
		Run: func(cmd *cobra.Command, args []string) {
			portal.ClientDecodeInstance(runner.settings.HttpAddr, args[0], args[1])
		},
	}
	portalDecodeCmd.AddCommand(portalDecodeInstanceCmd)

	portalCancelTxnCmd := &cobra.Command{
		Use:       "cancel",
		Short:     "Cancel APECS transaction",
		Args:      cobra.MinimumNArgs(1),
		ValidArgs: []string{"txn_id"},
		Run: func(cmd *cobra.Command, args []string) {
			portal.ClientCancelTxn(runner.settings.GrpcAddr, args[0])
		},
	}
	portalCancelTxnCmd.PersistentFlags().StringVar(&runner.settings.GrpcAddr, "grpc_addr", "localhost:11381", "Address against which to make client requests")
	portalCmd.AddCommand(portalCancelTxnCmd)

	portalServeCmd := &cobra.Command{
		Use:   "serve",
		Short: "Rocketcycle Portal Server",
		Long:  "Host rest api",
		Run: func(cmd *cobra.Command, args []string) {
			portal.Serve(cmd.Context(), runner.plat, runner.settings.HttpAddr, runner.settings.GrpcAddr)
		},
	}
	runner.addPlatformFlags(portalServeCmd)
	portalServeCmd.PersistentFlags().StringVar(&runner.settings.HttpAddr, "http_addr", ":11371", "Address to host http api")
	portalServeCmd.PersistentFlags().StringVar(&runner.settings.GrpcAddr, "grpc_addr", ":11381", "Address to host grpc api")
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
			mgmt.ConfigReplace(
				cmd.Context(),
				runner.plat.StreamProvider(),
				runner.plat.Args.Platform,
				runner.plat.Args.Environment,
				runner.plat.Args.AdminBrokers,
				runner.settings.ConfigFilePath,
			)
		},
	}
	runner.addPlatformFlags(configReplaceCmd)
	configReplaceCmd.PersistentFlags().StringVar(&runner.settings.ConfigFilePath, "config_file_path", "./config.json", "Path to json file containing Config values")
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
			rkcy.DecodeInstance(runner.clientCode.ConcernHandlers, args[0], args[1])
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
				runner.plat,
				"",
				runner.settings.ConsumerBrokers,
				runner.settings.Topic,
				runner.settings.Partition,
				runner.bailoutCh,
			)
		},
	}
	runner.addConsumerFlags(procCmd)
	rootCmd.AddCommand(procCmd)

	// storage sub command
	storageCmd := &cobra.Command{
		Use:   "storage",
		Short: "APECS storage mode",
		Long:  "Runs a storage consumer against the partition specified",
		Run: func(cmd *cobra.Command, args []string) {
			apecs.StartApecsConsumer(
				cmd.Context(),
				runner.plat,
				runner.settings.StorageTarget,
				runner.settings.ConsumerBrokers,
				runner.settings.Topic,
				runner.settings.Partition,
				runner.bailoutCh,
			)
		},
	}
	runner.addConsumerFlags(storageCmd)
	storageCmd.PersistentFlags().StringVar(&runner.settings.StorageTarget, "storage_target", "", "One of the named storage targets defined within platform config")
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
				runner.plat,
				runner.settings.StorageTarget,
				runner.settings.ConsumerBrokers,
				runner.settings.Topic,
				runner.settings.Partition,
				runner.bailoutCh,
			)
		},
	}
	runner.addConsumerFlags(storageScndCmd)
	storageScndCmd.PersistentFlags().StringVar(&runner.settings.StorageTarget, "storage_target", "", "One of the named storage targets defined within platform config")
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
				runner.plat.WaitGroup(),
				runner.plat.StreamProvider(),
				runner.plat.Args.Platform,
				runner.plat.Args.Environment,
				runner.plat.Args.AdminBrokers,
				runner.clientCode.ConcernHandlers,
				runner.settings.WatchDecode,
			)
		},
	}
	runner.addPlatformFlags(watchCmd)
	watchCmd.PersistentFlags().BoolVarP(&runner.settings.WatchDecode, "decode", "d", false, "If set, will decode all Buffer objects when printing ApecsTxn messages")
	rootCmd.AddCommand(watchCmd)

	// run sub command
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run all topic consumer programs",
		Long:  "Orchestrates sub processes as specified by platform topics consumer programs",
		Run: func(cmd *cobra.Command, args []string) {
			runner.plat.WaitGroup().Add(1)
			runner.RunConsumerPrograms(
				cmd.Context(),
				runner.plat.WaitGroup(),
				runner.plat.StreamProvider(),
				runner.plat.Args.Platform,
				runner.plat.Args.Environment,
				runner.settings.RunnerType,
				runner.plat.Args.AdminBrokers,
				runner.settings.OtelcolEndpoint,
				runner.settings.WatchDecode,
			)
		},
	}
	runner.addPlatformFlags(runCmd)
	runCmd.PersistentFlags().BoolVarP(&runner.settings.WatchDecode, "decode", "d", false, "If set, will decode all Buffer objects when printing ApecsTxn messages")
	runCmd.PersistentFlags().StringVar(&runner.settings.RunnerType, "runner", "process", "Type of runner to use to manage distributed processes or routines")
	runCmd.PersistentFlags().StringVar(&runner.settings.PlatformFilePath, "platform_file_path", "./platform.json", "Path to json file containing platform configuration")
	runCmd.PersistentFlags().StringVar(&runner.settings.ConfigFilePath, "config_file_path", "./config.json", "Path to json file containing Config values")
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
				runner.plat.StreamProvider(),
				runner.plat.Args.Platform,
				runner.plat.Args.Environment,
				runner.plat.Args.AdminBrokers,
				runner.settings.PlatformFilePath,
			)
		},
	}
	runner.addPlatformFlags(platReplaceCmd)
	platReplaceCmd.PersistentFlags().StringVar(&runner.settings.PlatformFilePath, "platform_file_path", "./platform.json", "Path to json file containing platform configuration")
	platCmd.AddCommand(platReplaceCmd)

	for _, cmd := range runner.clientCode.CustomCobraCommands {
		rootCmd.AddCommand(cmd)
	}

	return rootCmd
}
