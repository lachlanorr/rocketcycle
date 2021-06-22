// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package sim

import (
	"context"
	"math/rand"
	"sync"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	"github.com/lachlanorr/rocketcycle/version"

	"github.com/lachlanorr/rocketcycle/examples/rpg/edge"
)

type CommandId int

const (
	CmdCreateCharacters CommandId = iota
)

type Command struct {
	Name    string
	Handler func(context.Context, edge.RpgServiceClient, *rand.Rand, uint)
}

type SimCmd struct {
	CommandId       CommandId
	RunnerIdx       uint
	EdgeGrpcAddr    string
	SimulationCount uint
	RandomSeed      int64
}

var commands = map[CommandId]Command{
	CmdCreateCharacters: {"CreateCharacters", cmdCreateCharacters},
}

func simRunner(ctx context.Context, cmdCh <-chan *SimCmd, wg *sync.WaitGroup) {
	defer wg.Done()

	initCmd := <-cmdCh
	slog := log.With().
		Str("Command", commands[initCmd.CommandId].Name).
		Uint("RunnerIdx", initCmd.RunnerIdx).
		Uint("SimulationCount", initCmd.SimulationCount).
		Int64("RandomSeed", initCmd.RandomSeed).
		Logger()

	ctx = context.WithValue(ctx, "logger", slog)

	slog.Info().
		Msg("simRunner BEGIN")

	conn, err := grpc.Dial(initCmd.EdgeGrpcAddr, grpc.WithInsecure())
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to grpc.Dial")
	}
	defer conn.Close()
	client := edge.NewRpgServiceClient(conn)

	r := rand.New(rand.NewSource(initCmd.RandomSeed))
	commands[initCmd.CommandId].Handler(ctx, client, r, initCmd.SimulationCount)

	slog.Info().
		Msg("simRunner END")
}

func start(commandId CommandId, settings *Settings) {
	var wg sync.WaitGroup

	log.Info().
		Str("GitCommit", version.GitCommit).
		Str("Command", commands[commandId].Name).
		Str("EdgeGrpcAddr", settings.EdgeGrpcAddr).
		Uint("RunnerCount", settings.RunnerCount).
		Uint("SimulationCount", settings.SimulationCount).
		Int64("RandomSeed", settings.RandomSeed).
		Msg("simulation starting")

	r := rand.New(rand.NewSource(settings.RandomSeed))

	cmdChans := make([]chan *SimCmd, settings.RunnerCount)
	for i := uint(0); i < settings.RunnerCount; i++ {
		cmdChans[i] = make(chan *SimCmd)
		wg.Add(1)
		go simRunner(context.Background(), cmdChans[i], &wg)
		cmdChans[i] <- &SimCmd{
			CommandId:       commandId,
			RunnerIdx:       i,
			EdgeGrpcAddr:    settings.EdgeGrpcAddr,
			SimulationCount: settings.SimulationCount,
			RandomSeed:      r.Int63(),
		}
	}

	wg.Wait()
}
