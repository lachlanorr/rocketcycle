// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rkcy/pkg/rkcy/pb"
)

func runStorage(ctx context.Context, consumeTopic *pb.Platform_Concern_Topics) {
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

func substStr(s string, topicName string, partition int32) string {
	s = strings.ReplaceAll(s, "@platform", platformName)
	s = strings.ReplaceAll(s, "@topic", topicName)
	s = strings.ReplaceAll(s, "@partition", strconv.Itoa(int(partition)))
	return s
}

func progKey(prog *pb.Platform_Concern_Program) string {
	// Combine name and args to a string for key lookup
	if prog == nil {
		return "NIL"
	} else {
		return prog.Name + " " + strings.Join(prog.Args, " ")
	}
}

func updateRunning(running map[string]*exec.Cmd, progs []*pb.Platform_Concern_Program) {
	for _, prog := range progs {
		log.Info().
			Msgf("progKey: %s", progKey(prog))
	}
}

func expandProgs(concern *pb.Platform_Concern, topics *pb.Platform_Concern_Topics) []*pb.Platform_Concern_Program {
	progs := make([]*pb.Platform_Concern_Program, topics.Current.PartitionCount)
	for i := int32(0); i < topics.Current.PartitionCount; i++ {
		topicName := BuildFullTopicName(platformName, concern.Name, concern.Type, topics.Name, topics.Current.Generation)
		progs[i] = &pb.Platform_Concern_Program{
			Name: substStr(topics.ConsumerProgram.Name, topicName, i),
			Args: make([]string, len(topics.ConsumerProgram.Args)),
		}
		for j := 0; j < len(topics.ConsumerProgram.Args); j++ {
			progs[i].Args[j] = substStr(topics.ConsumerProgram.Args[j], topicName, i)
		}
	}
	return progs
}

func runConsumerPrograms(ctx context.Context, platCh <-chan *pb.Platform) {
	defaultProgs := []*pb.Platform_Concern_Program{
		{Name: "rpg", Args: []string{"admin", "serve"}},
	}

	running := make(map[string]*exec.Cmd)

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("manageTopics exiting, ctx.Done()")
			return
		case plat := <-platCh:
			rtPlat, err := newRtPlatform(plat)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to newRtPlatform")
				continue
			}

			progs := make([]*pb.Platform_Concern_Program, len(defaultProgs))
			copy(progs, defaultProgs)
			for _, concern := range rtPlat.Platform.Concerns {
				for _, topics := range concern.Topics {
					if topics.ConsumerProgram != nil {
						progs = append(progs, expandProgs(concern, topics)...)
					}
				}
			}
			updateRunning(running, progs)
		}
	}
}

func cobraRun(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	interruptCh := make(chan os.Signal)
	signal.Notify(interruptCh, os.Interrupt)

	platCh := make(chan *pb.Platform, 10)

	go runConsumerPrograms(ctx, platCh)
	go consumePlatformConfig(ctx, platCh, settings.BootstrapServers, platformName)
	for {
		select {
		case <-interruptCh:
			cancel()
			return
		}
	}
}
