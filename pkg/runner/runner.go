// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package runner

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rocketcycle/pkg/platform"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/pkg/stream"
	"github.com/lachlanorr/rocketcycle/pkg/telem"
)

type Runner struct {
	platform string

	ctx       context.Context
	ctxCancel context.CancelFunc
	wg        sync.WaitGroup
	bailoutCh chan bool

	settings Settings

	storageInits  []*StorageInit
	logicHandlers []*LogicHandler
	crudHandlers  []*CrudHandler

	customCobraCommands []*cobra.Command

	plat *platform.Platform
}

type StorageInit struct {
	storageType string
	storageInit rkcy.StorageInit
}

type LogicHandler struct {
	concern string
	handler interface{}
}

type CrudHandler struct {
	concern     string
	storageType string
	handler     interface{}
}

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

	Runner string
}

func NewRunner(ctx context.Context, platform string) *Runner {
	runner := &Runner{
		platform: platform,
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
	runner.storageInits = append(
		runner.storageInits,
		&StorageInit{
			storageType: storageType,
			storageInit: storageInit,
		},
	)
}

func (runner *Runner) AddLogicHandler(
	concern string,
	handler interface{},
) {
	runner.logicHandlers = append(
		runner.logicHandlers,
		&LogicHandler{
			concern: concern,
			handler: handler,
		},
	)
}

func (runner *Runner) AddCrudHandler(
	concern string,
	storageType string,
	handler interface{},
) {
	runner.crudHandlers = append(
		runner.crudHandlers,
		&CrudHandler{
			concern:     concern,
			storageType: storageType,
			handler:     handler,
		},
	)
}

func (runner *Runner) AddCobraCommand(cmd *cobra.Command) {
	runner.customCobraCommands = append(runner.customCobraCommands, cmd)
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

	err := telem.Initialize(runner.ctx, runner.settings.OtelcolEndpoint)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to telem.Initialize")
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

	strmprov := stream.NewKafkaStreamProvider()
	adminPingInterval := time.Duration(runner.settings.AdminPingIntervalSecs) * time.Second

	runner.plat, err = platform.NewPlatform(
		runner.ctx,
		&runner.wg,
		runner.platform,
		runner.settings.Environment,
		runner.settings.AdminBrokers,
		adminPingInterval,
		strmprov,
		respTarget,
	)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to NewKafkaPlatform")
	}

	for _, stgInit := range runner.storageInits {
		runner.plat.AddStorageInit(stgInit.storageType, stgInit.storageInit)
	}

	for _, logicHdlr := range runner.logicHandlers {
		runner.plat.AddLogicHandler(logicHdlr.concern, logicHdlr.handler)
	}

	for _, crudHdlr := range runner.crudHandlers {
		runner.plat.AddCrudHandler(crudHdlr.concern, crudHdlr.storageType, crudHdlr.handler)
	}

	// validate all command handlers exist for each concern
	if !runner.plat.ConcernHandlers.ValidateConcernHandlers() {
		log.Fatal().
			Msg("ValidateConcernHandlers failed")
	}
}

func (runner *Runner) Execute() {
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
	cobraCmd.Execute()

	cleanup()
}
