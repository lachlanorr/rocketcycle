// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"os"
	"os/signal"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rocketcycle/version"
)

func cobraProcess(cmd *cobra.Command, args []string) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("APECS PROCESS consumer started")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	defer func() {
		signal.Stop(interruptCh)
		cancel()
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go startApecsRunner(
		ctx,
		gSettings.AdminBrokers,
		gSettings.ConsumerBrokers,
		gPlatformName,
		gSettings.Topic,
		gSettings.Partition,
		&wg,
	)

	select {
	case <-interruptCh:
		log.Info().
			Msg("APECS PROCESS consumer stopped")
		cancel()
		wg.Wait()
		return
	}
}
