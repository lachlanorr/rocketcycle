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

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	admin_pb "github.com/lachlanorr/rocketcycle/build/proto/admin"
)

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

func runCommand(cmd *cobra.Command, args []string) {
	flags.platformName = args[0]

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)

	platCh := make(chan admin_pb.Platform, 10)
	go ConsumePlatformConfig(ctx, platCh, flags.bootstrapServers, flags.platformName)

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
