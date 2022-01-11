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

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rocketcycle/pkg/mgmt"
	"github.com/lachlanorr/rocketcycle/pkg/platform"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/pkg/runner/program"
	"github.com/lachlanorr/rocketcycle/pkg/runner/routine"
	"github.com/lachlanorr/rocketcycle/pkg/stream"
	"github.com/lachlanorr/rocketcycle/pkg/stream/offline"
	"github.com/lachlanorr/rocketcycle/pkg/telem"
)

type RkcyCmd struct {
	platform   string
	clientCode *rkcy.ClientCode

	ctx       context.Context
	ctxCancel context.CancelFunc
	wg        sync.WaitGroup
	bailoutCh chan bool

	settings *Settings

	plat *platform.Platform

	newRunnableFunc NewRunnableFunc

	customCommandFuncs []CustomCommandFunc

	offlineMgr *offline.Manager
}

type CustomCommandFunc func(rkcycmd *RkcyCmd) *cobra.Command

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

func NewRkcyCmd(ctx context.Context, platform string) *RkcyCmd {
	return newRkcyCmd(ctx, platform, rkcy.NewClientCode(), nil, nil)
}

func newRkcyCmd(
	ctx context.Context,
	platform string,
	clientCode *rkcy.ClientCode,
	customCommandFuncs []CustomCommandFunc,
	offlineMgr *offline.Manager,
) *RkcyCmd {
	rkcycmd := &RkcyCmd{
		platform:           platform,
		clientCode:         clientCode,
		settings:           &Settings{},
		customCommandFuncs: customCommandFuncs,
		offlineMgr:         offlineMgr,
	}

	if rkcycmd.customCommandFuncs == nil {
		rkcycmd.customCommandFuncs = make([]CustomCommandFunc, 0)
	}

	rkcycmd.ctx, rkcycmd.ctxCancel = context.WithCancel(ctx)
	rkcycmd.bailoutCh = make(chan bool)

	return rkcycmd
}

func (rkcycmd *RkcyCmd) Platform() *platform.Platform {
	return rkcycmd.plat
}

func (rkcycmd *RkcyCmd) AddStorageInit(storageType string, storageInit rkcy.StorageInit) {
	rkcycmd.clientCode.AddStorageInit(storageType, storageInit)
}

func (rkcycmd *RkcyCmd) AddLogicHandler(
	concern string,
	handler interface{},
) {
	rkcycmd.clientCode.AddLogicHandler(concern, handler)
}

func (rkcycmd *RkcyCmd) AddCrudHandler(
	concern string,
	storageType string,
	handler interface{},
) {
	rkcycmd.clientCode.AddCrudHandler(concern, storageType, handler)
}

func (rkcycmd *RkcyCmd) AddCobraCommandFunc(custCmdFunc CustomCommandFunc) {
	rkcycmd.customCommandFuncs = append(rkcycmd.customCommandFuncs, custCmdFunc)
}

func (rkcycmd *RkcyCmd) AddCobraConsumerCommandFunc(custCmdFunc CustomCommandFunc) {
	rkcycmd.customCommandFuncs = append(rkcycmd.customCommandFuncs, func(rkcycmd *RkcyCmd) *cobra.Command {
		cmd := custCmdFunc(rkcycmd)
		rkcycmd.addConsumerFlags(cmd)
		return cmd
	})
}

func (rkcycmd *RkcyCmd) AddCobraEdgeCommandFunc(custCmdFunc CustomCommandFunc) {
	rkcycmd.customCommandFuncs = append(rkcycmd.customCommandFuncs, func(rkcycmd *RkcyCmd) *cobra.Command {
		cmd := custCmdFunc(rkcycmd)
		rkcycmd.addEdgeFlags(cmd)
		return cmd
	})
}

func (rkcycmd *RkcyCmd) prerunCobra(cmd *cobra.Command, args []string) {
	rkcy.PrepLogging()

	platId := uuid.NewString()

	err := telem.Initialize(cmd.Context(), rkcycmd.settings.OtelcolEndpoint)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to telem.Initialize")
	}

	// validate all command handlers exist for each concern
	if !rkcycmd.clientCode.ConcernHandlers.ValidateConcernHandlers() {
		log.Fatal().
			Msg("ValidateConcernHandlers failed")
	}

	if rkcycmd.settings.StreamType == "auto" {
		if rkcycmd.settings.RunnerType == "routine" {
			rkcycmd.settings.StreamType = "offline"
		} else {
			rkcycmd.settings.StreamType = "kafka"
		}
	}

	var respTarget *rkcypb.TopicTarget
	if rkcycmd.settings.Edge {
		if rkcycmd.settings.ConsumerBrokers == "" || rkcycmd.settings.Topic == "" || rkcycmd.settings.Partition == -1 {
			log.Fatal().
				Msgf("edge flag set with missing consumer_brokers, topic, or partition")
		}
		respTarget = &rkcypb.TopicTarget{
			Brokers:   rkcycmd.settings.ConsumerBrokers,
			Topic:     rkcycmd.settings.Topic,
			Partition: rkcycmd.settings.Partition,
		}
	}

	var strmprov rkcy.StreamProvider

	if rkcycmd.settings.StreamType == "kafka" {
		strmprov = stream.NewKafkaStreamProvider()
	} else if rkcycmd.settings.StreamType == "offline" {
		if rkcycmd.offlineMgr == nil {
			platformDefJson, err := ioutil.ReadFile(rkcycmd.settings.PlatformFilePath)
			if err != nil {
				log.Fatal().
					Str("PlatformFilePath", rkcycmd.settings.PlatformFilePath).
					Err(err).
					Msg("Failed to ReadFile")
			}
			rtPlatDef, err := rkcy.NewRtPlatformDefFromJson(platformDefJson)
			if err != nil {
				log.Fatal().
					Str("PlatformFilePath", rkcycmd.settings.PlatformFilePath).
					Err(err).
					Msg("Failed to NewRtPlatformDefFromJson")
			}
			rkcycmd.offlineMgr, err = offline.NewManager(rtPlatDef)
			if err != nil {
				log.Fatal().
					Str("PlatformFilePath", rkcycmd.settings.PlatformFilePath).
					Err(err).
					Msg("Failed to offline.NewManager")
			}

			strmprov = offline.NewOfflineStreamProviderFromManager(rkcycmd.offlineMgr)
			err = rkcy.CreatePlatformTopics(
				cmd.Context(),
				strmprov,
				rkcycmd.platform,
				rkcycmd.settings.Environment,
				rkcycmd.settings.AdminBrokers,
			)
			if err != nil {
				log.Fatal().
					Str("PlatformFilePath", rkcycmd.settings.PlatformFilePath).
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
					Str("PlatformFilePath", rkcycmd.settings.PlatformFilePath).
					Err(err).
					Msg("Failed to UpdateTopics")
			}

			// publish our platform so everyone gets it
			mgmt.PlatformReplaceFromJson(
				cmd.Context(),
				strmprov,
				rkcycmd.platform,
				rkcycmd.settings.Environment,
				rkcycmd.settings.AdminBrokers,
				platformDefJson,
			)
			mgmt.ConfigReplace(
				cmd.Context(),
				strmprov,
				rkcycmd.platform,
				rkcycmd.settings.Environment,
				rkcycmd.settings.AdminBrokers,
				rkcycmd.settings.ConfigFilePath,
			)
		} else {
			strmprov = offline.NewOfflineStreamProviderFromManager(rkcycmd.offlineMgr)
		}

	} else {
		log.Fatal().Msgf("Invalid --stream: %s", rkcycmd.settings.StreamType)
	}

	if rkcycmd.settings.StreamType == "offline" {
		rkcycmd.newRunnableFunc = func(ctx context.Context, dets *program.Details) (program.Runnable, error) {
			return routine.NewRunnable(
				ctx,
				dets,
				func(ctx context.Context, args []string) {
					ExecuteFromArgs(
						ctx,
						rkcycmd.platform,
						rkcycmd.clientCode,
						rkcycmd.customCommandFuncs,
						rkcycmd.offlineMgr,
						args,
					)
				},
			)
		}
	} else {
		rkcycmd.newRunnableFunc = GetNewRunnableFunc(rkcycmd.settings.RunnerType)
	}

	adminPingInterval := time.Duration(rkcycmd.settings.AdminPingIntervalSecs) * time.Second

	rkcycmd.plat, err = platform.NewPlatform(
		cmd.Context(),
		&rkcycmd.wg,
		platId,
		rkcycmd.platform,
		rkcycmd.settings.Environment,
		rkcycmd.settings.AdminBrokers,
		adminPingInterval,
		strmprov,
		rkcycmd.clientCode,
		respTarget,
	)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to NewKafkaPlatform")
	}
}

func (rkcycmd *RkcyCmd) Execute() error {
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
			telem.Shutdown(rkcycmd.ctx)
			rkcycmd.ctxCancel()
			rkcycmd.wg.Wait()
			log.Info().Msgf("RkcyCmd.Execute completed")
		}
	}

	go func() {
		defer cleanup()

		select {
		case <-rkcycmd.ctx.Done():
			return
		case <-interruptCh:
			return
		case <-rkcycmd.bailoutCh:
			log.Warn().
				Msg("APECS Consumer Stopped, BAILOUT")
			return
		}
	}()

	cobraCmd := rkcycmd.BuildCobraCommand()
	err := cobraCmd.ExecuteContext(rkcycmd.ctx)
	cleanup()
	return err
}

func ExecuteFromArgs(
	ctx context.Context,
	platform string,
	clientCode *rkcy.ClientCode,
	customCommandFuncs []CustomCommandFunc,
	offlineMgr *offline.Manager,
	args []string,
) {
	rkcycmd := newRkcyCmd(ctx, platform, clientCode, customCommandFuncs, offlineMgr)
	cobraCmd := rkcycmd.BuildCobraCommand()
	cobraCmd.SetArgs(args)
	cobraCmd.ExecuteContext(ctx)
}
