// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package sim

import (
	"context"
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/version"

	store_pg "github.com/lachlanorr/rocketcycle/examples/rpg/crud_handlers/postgresql"
	"github.com/lachlanorr/rocketcycle/examples/rpg/edge"
)

type CommandId int

type RunnerArgs struct {
	RunnerIdx          uint
	EdgeGrpcAddr       string
	SimulationCount    uint
	RandomSeed         int64
	Ratios             []float64
	InitCharacterCount uint
	PreSleepSecs       uint
}

const (
	CmdCreateCharacter CommandId = iota
	CmdFund
	CmdTrade
	CmdReadPlayer
	CmdReadCharacter

	Cmd_COUNT
)

type Handler func(context.Context, edge.RpgServiceClient, *rand.Rand, *StateDb) (string, error)

type Command struct {
	Handler Handler
	Ratio   float64
}

var commands = map[CommandId]Command{
	CmdCreateCharacter: {Handler: cmdCreateCharacter, Ratio: 3},
	CmdFund:            {Handler: cmdFund, Ratio: 3},
	CmdTrade:           {Handler: cmdTrade, Ratio: 4},
	CmdReadPlayer:      {Handler: cmdReadPlayer, Ratio: 20},
	CmdReadCharacter:   {Handler: cmdReadCharacter, Ratio: 70},
}

type DifferenceType string

const (
	Error   DifferenceType = "Error"
	Process DifferenceType = "Process"
	Storage DifferenceType = "Storage"
)

type Difference struct {
	Message string
	Type    DifferenceType
	StateDb proto.Message
	Rkcy    proto.Message
}

func diff2json(diff *Difference) string {
	stateDbJson, err := protojson.Marshal(diff.StateDb)
	if err != nil {
		panic("diff2json: " + err.Error())
	}
	rkcyJson, err := protojson.Marshal(diff.Rkcy)
	if err != nil {
		panic("diff2json: " + err.Error())
	}

	stateDbMap := make(map[string]interface{})
	err = json.Unmarshal(stateDbJson, &stateDbMap)
	if err != nil {
		panic("diff2json: " + err.Error())
	}
	rkcyMap := make(map[string]interface{})
	err = json.Unmarshal(rkcyJson, &rkcyMap)
	if err != nil {
		panic("diff2json: " + err.Error())
	}

	out := make(map[string]interface{})
	out["message"] = diff.Message
	out["type"] = diff.Type
	out["stateDb"] = stateDbMap
	out["rkcy"] = rkcyMap

	outJson, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		panic("diff2json: " + err.Error())
	}
	return string(outJson)
}

func computeRatios(commands map[CommandId]Command) []float64 {
	ratios := make([]float64, Cmd_COUNT)

	ratioSum := 0.0
	for i := CommandId(0); i < Cmd_COUNT; i++ {
		ratioSum += commands[i].Ratio
		ratios[i] = ratioSum
	}
	if ratioSum != 100.0 {
		log.Fatal().
			Msgf("Invalid ratios, not summing to 100.0")
	}
	return ratios
}

func simRunner(ctx context.Context, args *RunnerArgs, wg *sync.WaitGroup) {
	defer wg.Done()

	log.Info().
		Msgf("%d RUNNER BEGIN", args.RunnerIdx)

	stateDb := NewStateDb()

	conn, err := grpc.Dial(args.EdgeGrpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatal().
			Err(err).
			Str("EdgeGrpcAddr", args.EdgeGrpcAddr).
			Msg("Failed to grpc.Dial")
	}
	defer conn.Close()
	client := edge.NewRpgServiceClient(conn)

	r := rand.New(rand.NewSource(args.RandomSeed))

	for i := uint(1); i <= args.InitCharacterCount; i++ {
		msg, err := cmdCreateCharacter(ctx, client, r, stateDb)
		if err != nil {
			log.Fatal().
				Err(err).
				Msg("Failed to create initial characters")
		} else {
			log.Info().
				Msgf("%d:%d/%d INIT %s", args.RunnerIdx, i, args.InitCharacterCount, msg)
		}
	}

	if args.PreSleepSecs > 0 {
		time.Sleep(time.Duration(args.PreSleepSecs) * time.Second)
	}

	for simIdx := uint(1); simIdx <= args.SimulationCount; simIdx++ {
		pct := r.Float64() * 100.0
		for cmdId := CommandId(0); cmdId < Cmd_COUNT; cmdId++ {
			if pct <= args.Ratios[cmdId] {
				msg, err := commands[cmdId].Handler(ctx, client, r, stateDb)
				if err != nil {
					log.Error().
						Err(err).
						Msgf("%d:%d/%d Error", args.RunnerIdx, simIdx, args.SimulationCount)
				} else {
					log.Info().
						Msgf("%d:%d/%d %s", args.RunnerIdx, simIdx, args.SimulationCount, msg)
				}
				break
			}
		}

	}

	diffsProcess := compareProcess(ctx, stateDb, client)
	for _, diff := range diffsProcess {
		log.Error().Msgf("%d PROCESS DIFF: \n%s", args.RunnerIdx, diff2json(diff))
	}

	var diffsStorage []*Difference
	diffWait := time.Duration(settings.DiffWaitSecs) * time.Second
	start := time.Now()
	for {
		diffsStorage = compareStorage(ctx, stateDb)

		t := time.Now()
		if t.Sub(start) > diffWait {
			break
		}

		if len(diffsStorage) == 0 {
			break
		} else {
			log.Warn().Msgf("%d STORAGE DIFFS %d, WAITING 10s", args.RunnerIdx, len(diffsStorage))
			time.Sleep(10 * time.Second)
		}
	}

	for _, diff := range diffsStorage {
		log.Error().Msgf("%d STORAGE DIFF: \n%s", args.RunnerIdx, diff2json(diff))
	}

	log.Info().
		Msgf("%d RUNNER END DIFFS p/%d s/%d", args.RunnerIdx, len(diffsProcess), len(diffsStorage))
}

func compareProcess(ctx context.Context, stateDb *StateDb, client edge.RpgServiceClient) []*Difference {
	diffs := make([]*Difference, 0, 10)

	for _, stateDbPlayer := range stateDb.Players {
		rkcyPlayer, err := client.ReadPlayer(ctx, &edge.RpgRequest{Id: stateDbPlayer.Id})
		if err != nil {
			diffs = append(diffs, &Difference{Message: err.Error(), Type: Error, StateDb: stateDbPlayer})
		}

		stateDbJson := protojson.Format(stateDbPlayer)
		rkcyJson := protojson.Format(rkcyPlayer.Player)

		if stateDbJson != rkcyJson {
			diffs = append(diffs, &Difference{Type: Process, StateDb: stateDbPlayer, Rkcy: rkcyPlayer})
		}
	}

	for _, stateDbCharacter := range stateDb.Characters {
		rkcyCharacter, err := client.ReadCharacter(ctx, &edge.RpgRequest{Id: stateDbCharacter.Id})
		if err != nil {
			diffs = append(diffs, &Difference{Message: err.Error(), Type: Error, StateDb: stateDbCharacter})
		}

		stateDbJson := protojson.Format(stateDbCharacter)
		rkcyJson := protojson.Format(rkcyCharacter.Character)

		if stateDbJson != rkcyJson {
			diffs = append(diffs, &Difference{Type: Process, StateDb: stateDbCharacter, Rkcy: rkcyCharacter})
		}
	}

	return diffs
}

func compareStorage(ctx context.Context, stateDb *StateDb) []*Difference {
	diffs := make([]*Difference, 0, 10)

	playerPg := store_pg.Player{}
	characterPg := store_pg.Character{}

	for _, stateDbPlayer := range stateDb.Players {
		rkcyPlayer, _, _, err := playerPg.Read(ctx, stateDbPlayer.Id)
		if err != nil {
			diffs = append(diffs, &Difference{Message: err.Error(), Type: Error, StateDb: stateDbPlayer})
		}

		stateDbJson := protojson.Format(stateDbPlayer)
		rkcyJson := protojson.Format(rkcyPlayer)

		if stateDbJson != rkcyJson {
			diffs = append(diffs, &Difference{Type: Storage, StateDb: stateDbPlayer, Rkcy: rkcyPlayer})
		}
	}

	for _, stateDbCharacter := range stateDb.Characters {
		rkcyCharacter, _, _, err := characterPg.Read(ctx, stateDbCharacter.Id)
		if err != nil {
			diffs = append(diffs, &Difference{Message: err.Error(), Type: Error, StateDb: stateDbCharacter})
		}

		stateDbJson := protojson.Format(stateDbCharacter)
		rkcyJson := protojson.Format(rkcyCharacter)

		if stateDbJson != rkcyJson {
			diffs = append(diffs, &Difference{Type: Storage, StateDb: stateDbCharacter, Rkcy: rkcyCharacter})
		}
	}

	return diffs
}

func start(settings *Settings) {
	ctx := context.Background()

	log.Info().
		Str("GitCommit", version.GitCommit).
		Str("EdgeGrpcAddr", settings.EdgeGrpcAddr).
		Uint("RunnerCount", settings.RunnerCount).
		Uint("SimulationCount", settings.SimulationCount).
		Int64("RandomSeed", settings.RandomSeed).
		Msg("simulation starting")

	// consider ratios
	ratios := computeRatios(commands)

	r := rand.New(rand.NewSource(settings.RandomSeed))

	var wg sync.WaitGroup
	for i := uint(0); i < settings.RunnerCount; i++ {
		wg.Add(1)
		go simRunner(
			ctx,
			&RunnerArgs{
				RunnerIdx:          i,
				EdgeGrpcAddr:       settings.EdgeGrpcAddr,
				SimulationCount:    settings.SimulationCount,
				RandomSeed:         r.Int63(),
				Ratios:             ratios,
				InitCharacterCount: settings.InitCharacterCount,
				PreSleepSecs:       settings.PreSleepSecs,
			},
			&wg,
		)
	}

	wg.Wait()
}
