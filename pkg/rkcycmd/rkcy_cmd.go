// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcycmd

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

type RkcyCmd struct {
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
}

func NewRkcyCmd(ctx context.Context, platform string) *RkcyCmd {
	rkcyCmd := &RkcyCmd{
		platform: platform,
	}

	rkcyCmd.ctx, rkcyCmd.ctxCancel = context.WithCancel(ctx)
	rkcyCmd.bailoutCh = make(chan bool)

	return rkcyCmd
}

func (rkcyCmd *RkcyCmd) PlatformFunc() func() *platform.Platform {
	return func() *platform.Platform {
		if rkcyCmd.plat == nil {
			panic("No platform initialized")
		}
		return rkcyCmd.plat
	}
}

func (rkcyCmd *RkcyCmd) AddStorageInit(storageType string, storageInit rkcy.StorageInit) {
	rkcyCmd.storageInits = append(
		rkcyCmd.storageInits,
		&StorageInit{
			storageType: storageType,
			storageInit: storageInit,
		},
	)
}

func (rkcyCmd *RkcyCmd) AddLogicHandler(
	concern string,
	handler interface{},
) {
	rkcyCmd.logicHandlers = append(
		rkcyCmd.logicHandlers,
		&LogicHandler{
			concern: concern,
			handler: handler,
		},
	)
}

func (rkcyCmd *RkcyCmd) AddCrudHandler(
	concern string,
	storageType string,
	handler interface{},
) {
	rkcyCmd.crudHandlers = append(
		rkcyCmd.crudHandlers,
		&CrudHandler{
			concern:     concern,
			storageType: storageType,
			handler:     handler,
		},
	)
}

func (rkcyCmd *RkcyCmd) AddCobraCommand(cmd *cobra.Command) {
	rkcyCmd.customCobraCommands = append(rkcyCmd.customCobraCommands, cmd)
}

func (rkcyCmd *RkcyCmd) AddCobraConsumerCommand(cmd *cobra.Command) {
	rkcyCmd.addConsumerFlags(cmd)
	rkcyCmd.AddCobraCommand(cmd)
}

func (rkcyCmd *RkcyCmd) AddCobraEdgeCommand(cmd *cobra.Command) {
	rkcyCmd.addEdgeFlags(cmd)
	rkcyCmd.AddCobraCommand(cmd)
}

func (rkcyCmd *RkcyCmd) prerunCobra(cmd *cobra.Command, args []string) {
	rkcy.PrepLogging()

	err := telem.Initialize(rkcyCmd.ctx, rkcyCmd.settings.OtelcolEndpoint)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to telem.Initialize")
	}

	var respTarget *rkcypb.TopicTarget
	if rkcyCmd.settings.Edge {
		if rkcyCmd.settings.ConsumerBrokers == "" || rkcyCmd.settings.Topic == "" || rkcyCmd.settings.Partition == -1 {
			log.Fatal().
				Msgf("edge flag set with missing consumer_brokers, topic, or partition")
		}
		respTarget = &rkcypb.TopicTarget{
			Brokers:   rkcyCmd.settings.ConsumerBrokers,
			Topic:     rkcyCmd.settings.Topic,
			Partition: rkcyCmd.settings.Partition,
		}
	}

	strmprov := stream.NewKafkaStreamProvider()
	adminPingInterval := time.Duration(rkcyCmd.settings.AdminPingIntervalSecs) * time.Second

	rkcyCmd.plat, err = platform.NewPlatform(
		rkcyCmd.ctx,
		&rkcyCmd.wg,
		rkcyCmd.platform,
		rkcyCmd.settings.Environment,
		rkcyCmd.settings.AdminBrokers,
		adminPingInterval,
		strmprov,
		respTarget,
	)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to NewKafkaPlatform")
	}

	for _, stgInit := range rkcyCmd.storageInits {
		rkcyCmd.plat.AddStorageInit(stgInit.storageType, stgInit.storageInit)
	}

	for _, logicHdlr := range rkcyCmd.logicHandlers {
		rkcyCmd.plat.AddLogicHandler(logicHdlr.concern, logicHdlr.handler)
	}

	for _, crudHdlr := range rkcyCmd.crudHandlers {
		rkcyCmd.plat.AddCrudHandler(crudHdlr.concern, crudHdlr.storageType, crudHdlr.handler)
	}

	// validate all command handlers exist for each concern
	if !rkcyCmd.plat.ConcernHandlers.ValidateConcernHandlers() {
		log.Fatal().
			Msg("ValidateConcernHandlers failed")
	}
}

func (rkcyCmd *RkcyCmd) Start() {
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
			telem.Shutdown(rkcyCmd.ctx)
			rkcyCmd.ctxCancel()
			rkcyCmd.wg.Wait()
			log.Info().Msgf("RkcyCmd.Start completed")
		}
	}

	go func() {
		defer cleanup()

		select {
		case <-rkcyCmd.ctx.Done():
			return
		case <-interruptCh:
			return
		case <-rkcyCmd.bailoutCh:
			log.Warn().
				Msg("APECS Consumer Stopped, BAILOUT")
			return
		}
	}()

	rkcyCmd.runCobra()
	cleanup()
}
