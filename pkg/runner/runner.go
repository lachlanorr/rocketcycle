// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package runner

import (
	"context"
	"io/ioutil"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rocketcycle/pkg/mgmt"
	"github.com/lachlanorr/rocketcycle/pkg/platform"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/pkg/runner/routine"
	"github.com/lachlanorr/rocketcycle/pkg/stream"
	"github.com/lachlanorr/rocketcycle/pkg/stream/offline"
	"github.com/lachlanorr/rocketcycle/pkg/telem"
)

type Runner struct {
	platform   string
	clientCode *rkcy.ClientCode

	ctx       context.Context
	ctxCancel context.CancelFunc
	wg        sync.WaitGroup
	bailoutCh chan bool

	settings Settings

	plat *platform.Platform
}

var gOfflineMgr *offline.Manager

type Settings struct {
	PlatformFilePath string
	ConfigFilePath   string

	OtelcolEndpoint string

	Environment     string
	AdminBrokers    string
	ConsumerBrokers string

	HttpAddr string
	GrpcAddr string

	Topic     string
	Partition int32

	Edge bool

	AdminPingIntervalSecs uint

	StorageTarget string

	WatchDecode bool

	StreamType string
	RunnerType string
}

func NewRunner(ctx context.Context, platform string) *Runner {
	return NewRunnerWithClientCode(ctx, platform, rkcy.NewClientCode())
}

func NewRunnerWithClientCode(ctx context.Context, platform string, clientCode *rkcy.ClientCode) *Runner {
	runner := &Runner{
		platform:   platform,
		clientCode: clientCode,
	}

	runner.ctx, runner.ctxCancel = context.WithCancel(ctx)
	runner.bailoutCh = make(chan bool)

	return runner
}

func (runner *Runner) PlatformFunc() func() *platform.Platform {
	return func() *platform.Platform {
		if runner.plat == nil {
			panic("No platform initialized")
		}
		return runner.plat
	}
}

func (runner *Runner) AddStorageInit(storageType string, storageInit rkcy.StorageInit) {
	runner.clientCode.AddStorageInit(storageType, storageInit)
}

func (runner *Runner) AddLogicHandler(
	concern string,
	handler interface{},
) {
	runner.clientCode.AddLogicHandler(concern, handler)
}

func (runner *Runner) AddCrudHandler(
	concern string,
	storageType string,
	handler interface{},
) {
	runner.clientCode.AddCrudHandler(concern, storageType, handler)
}

func (runner *Runner) AddCobraCommand(cmd *cobra.Command) {
	runner.clientCode.CustomCobraCommands = append(runner.clientCode.CustomCobraCommands, cmd)
}

func (runner *Runner) AddCobraConsumerCommand(cmd *cobra.Command) {
	runner.addConsumerFlags(cmd)
	runner.AddCobraCommand(cmd)
}

func (runner *Runner) AddCobraEdgeCommand(cmd *cobra.Command) {
	runner.addEdgeFlags(cmd)
	runner.AddCobraCommand(cmd)
}

func (runner *Runner) prerunCobra(cmd *cobra.Command, args []string) {
	rkcy.PrepLogging()

	err := telem.Initialize(cmd.Context(), runner.settings.OtelcolEndpoint)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to telem.Initialize")
	}

	// validate all command handlers exist for each concern
	if !runner.clientCode.ConcernHandlers.ValidateConcernHandlers() {
		log.Fatal().
			Msg("ValidateConcernHandlers failed")
	}

	var respTarget *rkcypb.TopicTarget
	if runner.settings.Edge {
		if runner.settings.ConsumerBrokers == "" || runner.settings.Topic == "" || runner.settings.Partition == -1 {
			log.Fatal().
				Msgf("edge flag set with missing consumer_brokers, topic, or partition")
		}
		respTarget = &rkcypb.TopicTarget{
			Brokers:   runner.settings.ConsumerBrokers,
			Topic:     runner.settings.Topic,
			Partition: runner.settings.Partition,
		}
	}

	var strmprov rkcy.StreamProvider

	if runner.settings.StreamType == "kafka" {
		strmprov = stream.NewKafkaStreamProvider()
	} else if runner.settings.StreamType == "offline" {
		if gOfflineMgr == nil {
			routine.SetExecuteFunc(func(ctx context.Context, args []string) {
				ExecuteFromArgs(ctx, runner.platform, runner.clientCode, args)
			})

			platformDefJson, err := ioutil.ReadFile(runner.settings.PlatformFilePath)
			if err != nil {
				log.Fatal().
					Str("PlatformFilePath", runner.settings.PlatformFilePath).
					Err(err).
					Msg("Failed to ReadFile")
			}
			rtPlatDef, err := rkcy.NewRtPlatformDefFromJson(platformDefJson)
			if err != nil {
				log.Fatal().
					Str("PlatformFilePath", runner.settings.PlatformFilePath).
					Err(err).
					Msg("Failed to NewRtPlatformDefFromJson")
			}
			gOfflineMgr, err = offline.NewManager(rtPlatDef)
			if err != nil {
				log.Fatal().
					Str("PlatformFilePath", runner.settings.PlatformFilePath).
					Err(err).
					Msg("Failed to offline.NewManager")
			}

			strmprov = offline.NewOfflineStreamProviderFromManager(gOfflineMgr)
			err = rkcy.CreatePlatformTopics(
				cmd.Context(),
				strmprov,
				runner.platform,
				runner.settings.Environment,
				runner.settings.AdminBrokers,
			)
			if err != nil {
				log.Fatal().
					Str("PlatformFilePath", runner.settings.PlatformFilePath).
					Err(err).
					Msg("Failed to CreatePlatformTopics")
			}

			err = rkcy.UpdateTopics(
				cmd.Context(),
				strmprov,
				rtPlatDef.PlatformDef,
			)
			if err != nil {
				log.Fatal().
					Str("PlatformFilePath", runner.settings.PlatformFilePath).
					Err(err).
					Msg("Failed to UpdateTopics")
			}

			// publish our platform so everyone gets it
			mgmt.PlatformReplaceFromJson(
				cmd.Context(),
				strmprov,
				runner.platform,
				runner.settings.Environment,
				runner.settings.AdminBrokers,
				platformDefJson,
			)
			mgmt.ConfigReplace(
				cmd.Context(),
				strmprov,
				runner.platform,
				runner.settings.Environment,
				runner.settings.AdminBrokers,
				runner.settings.ConfigFilePath,
			)
		} else {
			strmprov = offline.NewOfflineStreamProviderFromManager(gOfflineMgr)
		}
	} else {
		log.Fatal().Msgf("Invalid --stream: %s", runner.settings.StreamType)
	}

	adminPingInterval := time.Duration(runner.settings.AdminPingIntervalSecs) * time.Second

	runner.plat, err = platform.NewPlatform(
		cmd.Context(),
		&runner.wg,
		runner.platform,
		runner.settings.Environment,
		runner.settings.AdminBrokers,
		adminPingInterval,
		strmprov,
		runner.clientCode,
		respTarget,
	)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to NewKafkaPlatform")
	}
}

func (runner *Runner) Execute() error {
	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)

	cleanupHasRun := false
	var cleanupMtx sync.Mutex
	cleanup := func() {
		cleanupMtx.Lock()
		defer cleanupMtx.Unlock()
		if !cleanupHasRun {
			cleanupHasRun = true
			signal.Stop(interruptCh)
			telem.Shutdown(runner.ctx)
			runner.ctxCancel()
			runner.wg.Wait()
			log.Info().Msgf("Runner.Start completed")
		}
	}

	go func() {
		defer cleanup()

		select {
		case <-runner.ctx.Done():
			return
		case <-interruptCh:
			return
		case <-runner.bailoutCh:
			log.Warn().
				Msg("APECS Consumer Stopped, BAILOUT")
			return
		}
	}()

	cobraCmd := runner.BuildCobraCommand()
	err := cobraCmd.ExecuteContext(runner.ctx)
	cleanup()
	return err
}

func ExecuteFromArgs(ctx context.Context, platform string, clientCode *rkcy.ClientCode, args []string) {
	runner := NewRunnerWithClientCode(ctx, platform, clientCode)
	cobraCmd := runner.BuildCobraCommand()
	cobraCmd.SetArgs(args)
	cobraCmd.ExecuteContext(ctx)
}
