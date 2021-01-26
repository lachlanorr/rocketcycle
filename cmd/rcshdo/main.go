// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	admin_pb "github.com/lachlanorr/rocketcycle/build/proto/admin"
	"github.com/lachlanorr/rocketcycle/internal/rckafka"
	"github.com/lachlanorr/rocketcycle/internal/utils"
)

// Cobra sets these values based on command parsing
var (
	platform         string
	bootstrapServers string
	configFilePath   string
)

func runCobra() {
	rootCmd := &cobra.Command{
		Use:   "rcshdo",
		Short: "Rocketcycle Runner",
		Long:  "Runs sub processes as required by platform definition",
	}

	runCmd := &cobra.Command{
		Use:       "run platform",
		Short:     "Run rocketcycle platform",
		Long:      "Orchestrates sub rcproc and rcstore processes as specified by platform definition",
		Run:       rcshdoRun,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"platform"},
	}
	runCmd.PersistentFlags().StringVarP(&bootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read platform config and begin all other processes")
	rootCmd.AddCommand(runCmd)

	confCmd := &cobra.Command{
		Use:       "conf",
		Short:     "Update platform config",
		Long:      "Publishes contents of platform config file to platform topic. Creates platform topic if it doesn't already exist.",
		Run:       rcshdoConf,
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"config_path"},
	}
	confCmd.PersistentFlags().StringVarP(&bootstrapServers, "bootstrap_servers", "b", "localhost", "Kafka bootstrap servers from which to read metadata and begin all other processes")
	confCmd.PersistentFlags().StringVarP(&configFilePath, "config_file_path", "c", "./platform.json", "Path to json file containing platform configuration")
	rootCmd.AddCommand(confCmd)

	rootCmd.Execute()
}

const (
	colorBlack = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
	colorCyan
	colorWhite

	colorBold     = 1
	colorDarkGray = 90
)

func colorize(s interface{}, c int) string {
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}

type runningProgram struct {
}

func runStorage(ctx context.Context, consumeTopic *admin_pb.Platform_App_Topics) {
	cmd := exec.CommandContext(ctx, "./rcstore")

	stderr, _ := cmd.StderrPipe()
	cmd.Start()

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		m := scanner.Text()
		fmt.Fprintf(os.Stderr, colorize("%s %s\n", colorBlue), "rcstore", m)
	}
	cmd.Wait()
	log.Info().Msg("runStorage exit")
}

func rcshdoConf(cmd *cobra.Command, args []string) {
	slog := log.With().
		Str("BootstrapServers", bootstrapServers).
		Str("ConfigPath", configFilePath).
		Logger()

	// read platform conf file and deserialize
	conf, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to ReadFile")
	}
	plat := admin_pb.Platform{}
	err = protojson.Unmarshal(conf, proto.Message(&plat))
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to unmarshal platform")
	}
	platformTopic := rckafka.PlatformTopic(plat.Name)
	slog = slog.With().
		Str("Topic", platformTopic).
		Logger()
	platMar, err := proto.Marshal(&plat)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to Marshal platform")
	}

	// connect to kafka and make sure we have our platform topic
	admin, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapServers,
	})
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to NewAdminClient")
	}

	md, err := admin.GetMetadata(nil, true, 1000)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to GetMetadata")
	}

	_, ok := md.Topics[platformTopic]
	if !ok { // platform topic doesn't exist
		result, err := admin.CreateTopics(
			context.Background(),
			[]kafka.TopicSpecification{
				{
					Topic:             platformTopic,
					NumPartitions:     1,
					ReplicationFactor: len(md.Brokers),
				},
			},
			nil,
		)
		if err != nil {
			slog.Fatal().
				Err(err).
				Msg("Failed to create metadata topic")
		}
		for _, res := range result {
			if res.Error.Code() != kafka.ErrNoError {
				slog.Fatal().
					Err(res.Error).
					Msg("Failed to create metadata topic")
			}
		}
		slog.Info().
			Msg("Created platform topic")
	}

	// At this point we are guaranteed to have a platform config topic

	prod, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": bootstrapServers})
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to NewProducer")
	}
	defer prod.Close()

	err = prod.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &platformTopic, Partition: 0},
		Value:          platMar,
	}, nil)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to Produce")
	}

	// check channel for delivery event
	timer := time.NewTimer(5 * time.Second)
	select {
	case <-timer.C:
		slog.Fatal().
			Msg("Timeout waiting for producer callback")
	case ev := <-prod.Events():
		msgEv, ok := ev.(*kafka.Message)
		if !ok {
			slog.Fatal().
				Msg("Non *kafka.Message event received from producer")
		} else {
			if msgEv.TopicPartition.Error != nil {
				slog.Fatal().
					Err(msgEv.TopicPartition.Error).
					Msg("Error reported while producing platform message")
			} else {
				slog.Info().
					Msg("Platform config successfully produced")
			}
		}
	}
}

func rcshdoRun(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	platformName := args[0]

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)

	platCh := make(chan admin_pb.Platform, 10)
	go rckafka.ConsumePlatformConfig(ctx, platCh, bootstrapServers, platformName)

	for {
		select {
		case plat := <-platCh:
			log.Info().
				Msgf("platform message read %s", plat.Name)
		case <-interruptCh:
			return
		}
	}
}

func main() {
	utils.PrepLogging()
	runCobra()
}
