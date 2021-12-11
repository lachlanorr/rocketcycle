// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcycmd

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rocketcycle/pkg/admin"
	"github.com/lachlanorr/rocketcycle/pkg/config"
	"github.com/lachlanorr/rocketcycle/pkg/platform"
	"github.com/lachlanorr/rocketcycle/pkg/portal"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/pkg/runner"
	"github.com/lachlanorr/rocketcycle/pkg/watch"
)

type RkcyCmd struct {
	plat                rkcy.Platform
	settings            Settings
	customCobraCommands []*cobra.Command
}

type Settings struct {
	PlatformFilePath string
	ConfigFilePath   string

	OtelcolEndpoint string

	AdminBrokers    string
	ConsumerBrokers string

	HttpAddr string
	GrpcAddr string

	Topic     string
	Partition int32

	AdminPingIntervalSecs uint

	StorageTarget string

	WatchDecode bool
}

func NewRkcyCmd(plat rkcy.Platform) *RkcyCmd {
	return &RkcyCmd{
		plat:                plat,
		settings:            Settings{Partition: -1},
		customCobraCommands: make([]*cobra.Command, 0),
	}
}

func (rkcyCmd *RkcyCmd) AppendCobraCommand(cmd *cobra.Command) {
	rkcyCmd.customCobraCommands = append(rkcyCmd.customCobraCommands, cmd)
}

func (rkcyCmd *RkcyCmd) prerunCobra(cmd *cobra.Command, args []string) {
	err := rkcyCmd.plat.Init(
		rkcyCmd.settings.AdminBrokers,
		time.Duration(rkcyCmd.settings.AdminPingIntervalSecs)*time.Second,
		rkcyCmd.settings.StorageTarget,
		rkcyCmd.settings.OtelcolEndpoint,
	)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to initialize platform")
	}
}

func (rkcyCmd *RkcyCmd) postrunCobra(cmd *cobra.Command, args []string) {
	if rkcyCmd.plat.Telem() != nil {
		rkcyCmd.plat.Telem().Close()
	}
}

func (rkcyCmd *RkcyCmd) Start() {
	prepLogging()
	// validate all command handlers exist for each concern
	if !rkcyCmd.plat.ConcernHandlers().ValidateConcernHandlers() {
		log.Fatal().
			Msg("ValidateConcernHandlers failed")
	}
	rkcyCmd.runCobra()
}

func prepLogging() {
	logLevel := os.Getenv("RKCY_LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	badParse := false
	lvl, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		lvl = zerolog.InfoLevel
		badParse = true
	}

	zerolog.SetGlobalLevel(lvl)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02T15:04:05.999"})

	if badParse {
		log.Error().
			Msgf("Bad value for RKCY_LOG_LEVEL: %s", os.Getenv("RKCY_LOG_LEVEL"))
	}
}

func (rkcyCmd *RkcyCmd) runCobra() {
	rootCmd := &cobra.Command{
		Use:               rkcyCmd.plat.Name(),
		Short:             "Rocketcycle Platform - " + rkcyCmd.plat.Name(),
		PersistentPreRun:  rkcyCmd.prerunCobra,
		PersistentPostRun: rkcyCmd.postrunCobra,
	}
	rootCmd.PersistentFlags().StringVar(&rkcyCmd.settings.OtelcolEndpoint, "otelcol_endpoint", "localhost:4317", "OpenTelemetry collector address")
	rootCmd.PersistentFlags().StringVar(&rkcyCmd.settings.AdminBrokers, "admin_brokers", "localhost:9092", "Kafka brokers for admin messages like platform updates")
	rootCmd.PersistentFlags().UintVar(&rkcyCmd.settings.AdminPingIntervalSecs, "admin_ping_interval_secs", 1, "Interval for producers to ping the management system")

	// admin sub command
	adminCmd := &cobra.Command{
		Use:   "admin",
		Short: "Admin service that manages topics and orchestration",
		Run: func(*cobra.Command, []string) {
			admin.Start(rkcyCmd.plat)
		},
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
			portal.Serve(rkcyCmd.plat, rkcyCmd.settings.HttpAddr, rkcyCmd.settings.GrpcAddr)
		},
	}
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
			config.ConfigReplace(rkcyCmd.plat, rkcyCmd.settings.ConfigFilePath)
		},
	}
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
			platform.DecodeInstance(rkcyCmd.plat, args[0], args[1])
		},
	}
	decodeCmd.AddCommand(decodeInstanceCmd)

	// process sub command
	procCmd := &cobra.Command{
		Use:   "process",
		Short: "APECS processing mode",
		Long:  "Runs a proc consumer against the partition specified",
		Run: func(*cobra.Command, []string) {
			platform.StartApecsConsumer(
				rkcyCmd.plat,
				rkcyCmd.settings.ConsumerBrokers,
				rkcypb.System_PROCESS,
				rkcyCmd.settings.Topic,
				rkcyCmd.settings.Partition,
			)
		},
	}
	procCmd.PersistentFlags().StringVar(&rkcyCmd.settings.ConsumerBrokers, "consumer_brokers", "", "Kafka brokers against which to consume topic")
	procCmd.MarkPersistentFlagRequired("consumer_brokers")
	procCmd.PersistentFlags().StringVarP(&rkcyCmd.settings.Topic, "topic", "t", "", "Topic to consume")
	procCmd.MarkPersistentFlagRequired("topic")
	procCmd.PersistentFlags().Int32VarP(&rkcyCmd.settings.Partition, "partition", "p", -1, "Partition to consume")
	procCmd.MarkPersistentFlagRequired("partition")
	rootCmd.AddCommand(procCmd)

	// storage sub command
	storageCmd := &cobra.Command{
		Use:   "storage",
		Short: "APECS storage mode",
		Long:  "Runs a storage consumer against the partition specified",
		Run: func(*cobra.Command, []string) {
			platform.StartApecsConsumer(
				rkcyCmd.plat,
				rkcyCmd.settings.ConsumerBrokers,
				rkcypb.System_STORAGE,
				rkcyCmd.settings.Topic,
				rkcyCmd.settings.Partition,
			)
		},
	}
	storageCmd.PersistentFlags().StringVar(&rkcyCmd.settings.ConsumerBrokers, "consumer_brokers", "", "Kafka brokers against which to consume topic")
	storageCmd.MarkPersistentFlagRequired("consumer_brokers")
	storageCmd.PersistentFlags().StringVarP(&rkcyCmd.settings.Topic, "topic", "t", "", "Topic to consume")
	storageCmd.MarkPersistentFlagRequired("topic")
	storageCmd.PersistentFlags().Int32VarP(&rkcyCmd.settings.Partition, "partition", "p", -1, "Partition to consume")
	storageCmd.MarkPersistentFlagRequired("partition")
	storageCmd.PersistentFlags().StringVar(&rkcyCmd.settings.StorageTarget, "storage_target", "", "One of the named storage targets defined within platform config")
	storageCmd.MarkPersistentFlagRequired("storage_target")
	rootCmd.AddCommand(storageCmd)

	// secondary-storage sub command
	storageScndCmd := &cobra.Command{
		Use:   "storage-scnd",
		Short: "APECS secondary storage mode",
		Long:  "Runs a secondary storage consumer against the partition specified",
		Run: func(*cobra.Command, []string) {
			platform.StartApecsConsumer(
				rkcyCmd.plat,
				rkcyCmd.settings.ConsumerBrokers,
				rkcypb.System_STORAGE_SCND,
				rkcyCmd.settings.Topic,
				rkcyCmd.settings.Partition,
			)
		},
	}
	storageScndCmd.PersistentFlags().StringVar(&rkcyCmd.settings.ConsumerBrokers, "consumer_brokers", "", "Kafka brokers against which to consume topic")
	storageScndCmd.MarkPersistentFlagRequired("consumer_brokers")
	storageScndCmd.PersistentFlags().StringVarP(&rkcyCmd.settings.Topic, "topic", "t", "", "Topic to consume")
	storageScndCmd.MarkPersistentFlagRequired("topic")
	storageScndCmd.PersistentFlags().Int32VarP(&rkcyCmd.settings.Partition, "partition", "p", -1, "Partition to consume")
	storageScndCmd.MarkPersistentFlagRequired("partition")
	storageScndCmd.PersistentFlags().StringVar(&rkcyCmd.settings.StorageTarget, "storage_target", "", "One of the named storage targets defined within platform config")
	storageScndCmd.MarkPersistentFlagRequired("storage_target")
	rootCmd.AddCommand(storageScndCmd)

	// watch sub command
	watchCmd := &cobra.Command{
		Use:   "watch",
		Short: "APECS watch mode",
		Long:  "Runs a watch consumer against all error/complete topics",
		Run: func(*cobra.Command, []string) {
			watch.Start(rkcyCmd.plat, rkcyCmd.settings.WatchDecode)
		},
	}
	watchCmd.PersistentFlags().BoolVarP(&rkcyCmd.settings.WatchDecode, "decode", "d", false, "If set, will decode all Buffer objects when printing ApecsTxn messages")
	rootCmd.AddCommand(watchCmd)

	// run sub command
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run all topic consumer programs",
		Long:  "Orchestrates sub processes as specified by platform topics consumer programs",
		Run: func(*cobra.Command, []string) {
			runner.Start(rkcyCmd.plat, rkcyCmd.settings.WatchDecode)
		},
	}
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
			platform.PlatformReplace(rkcyCmd.plat, rkcyCmd.settings.PlatformFilePath)
		},
	}
	platReplaceCmd.PersistentFlags().StringVarP(&rkcyCmd.settings.PlatformFilePath, "platform_file_path", "p", "./platform.json", "Path to json file containing platform configuration")
	platCmd.AddCommand(platReplaceCmd)

	for _, custCmd := range rkcyCmd.customCobraCommands {
		rootCmd.AddCommand(custCmd)
	}

	rootCmd.Execute()
}
