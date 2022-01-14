// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package runner

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/lachlanorr/rocketcycle/pkg/mgmt"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/pkg/runner/program"
)

func updateRunning(
	ctx context.Context,
	wg *sync.WaitGroup,
	running map[string]program.Runnable,
	newRunnableFunc NewRunnableFunc,
	directive rkcypb.Directive,
	acd *rkcypb.ConsumerDirective,
	printCh chan<- string,
) {
	key := rkcy.ProgKey(acd.Program)
	var (
		rnbl program.Runnable
		ok   bool
		err  error
	)

	switch directive {
	case rkcypb.Directive_CONSUMER_START:
		rnbl, ok = running[key]
		if ok {
			log.Warn().
				Msg("Program already running: " + key)
			return
		}
		dets := program.NewDetails(acd.Program, key)
		rnbl, err = newRunnableFunc(ctx, dets)
		if err != nil {
			log.Error().
				Err(err).
				Str("ProgramKey", key).
				Msg("Failed to newRunnableFunc")
		}
		err = startRunnable(ctx, wg, rnbl, printCh)
		if err != nil {
			return
		}
		running[key] = rnbl
	case rkcypb.Directive_CONSUMER_STOP:
		rnbl, ok = running[key]
		if !ok {
			log.Warn().Msg("Program not running running, cannot stop: " + key)
			return
		} else {
			delete(running, key)
			rnbl.Kill()
		}
	}
}

func printer(ctx context.Context, wg *sync.WaitGroup, printCh <-chan string) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("runner printer stopped")
			return
		case line := <-printCh:
			fmt.Println(line)
		}
	}
}

func startRunnable(
	ctx context.Context,
	wg *sync.WaitGroup,
	rnbl program.Runnable,
	printCh chan<- string,
) error {
	log.Info().Msgf("Running: %s %s", rnbl.Details().Program.Name, strings.Join(rnbl.Details().Program.Args, " "))
	err := rnbl.Start(ctx, wg, printCh, true)
	if err != nil {
		log.Error().
			Err(err).
			Str("ProgramKey", rnbl.Details().Key).
			Msg("Unable to start")
		return err
	}
	rnbl.Details().RunCount++
	return nil
}

func doMaintenance(ctx context.Context, wg *sync.WaitGroup, running map[string]program.Runnable, printCh chan<- string) {
	for _, rnbl := range running {
		if !rnbl.IsRunning() {
			startRunnable(ctx, wg, rnbl, printCh)
		}
	}
}

func defaultArgs(environment string, streamType string, adminBrokers string, otelcolEndpoint string) []string {
	return []string{
		"-e", environment,
		"--stream", streamType,
		"--admin_brokers", adminBrokers,
		"--otelcol_endpoint", otelcolEndpoint,
	}
}

func startAdmin(
	ctx context.Context,
	wg *sync.WaitGroup,
	platform string,
	environment string,
	streamType string,
	adminBrokers string,
	otelcolEndpoint string,
	running map[string]program.Runnable,
	newRunnableFunc NewRunnableFunc,
	printCh chan<- string,
) {
	updateRunning(
		ctx,
		wg,
		running,
		newRunnableFunc,
		rkcypb.Directive_CONSUMER_START,
		&rkcypb.ConsumerDirective{
			Program: &rkcypb.Program{
				Name:   "./" + platform,
				Args:   append([]string{"admin"}, defaultArgs(environment, streamType, adminBrokers, otelcolEndpoint)...),
				Abbrev: "admin",
				Tags:   map[string]string{"service.name": fmt.Sprintf("rkcy.%s.%s.admin", platform, environment)},
			},
		},
		printCh,
	)
}

func startPortalServer(
	ctx context.Context,
	wg *sync.WaitGroup,
	platform string,
	environment string,
	streamType string,
	adminBrokers string,
	otelcolEndpoint string,
	running map[string]program.Runnable,
	newRunnableFunc NewRunnableFunc,
	printCh chan<- string,
) {
	updateRunning(
		ctx,
		wg,
		running,
		newRunnableFunc,
		rkcypb.Directive_CONSUMER_START,
		&rkcypb.ConsumerDirective{
			Program: &rkcypb.Program{
				Name:   "./" + platform,
				Args:   append([]string{"portal", "serve"}, defaultArgs(environment, streamType, adminBrokers, otelcolEndpoint)...),
				Abbrev: "portal",
				Tags:   map[string]string{"service.name": fmt.Sprintf("rkcy.%s.%s.portal", platform, environment)},
			},
		},
		printCh,
	)
}

func startWatch(
	ctx context.Context,
	wg *sync.WaitGroup,
	platform string,
	environment string,
	streamType string,
	adminBrokers string,
	otelcolEndpoint string,
	watchDecode bool,
	running map[string]program.Runnable,
	newRunnableFunc NewRunnableFunc,
	printCh chan<- string,
) {
	args := []string{"watch"}
	if watchDecode {
		args = append(args, "-d")
	}
	args = append(args, defaultArgs(environment, streamType, adminBrokers, otelcolEndpoint)...)

	updateRunning(
		ctx,
		wg,
		running,
		newRunnableFunc,
		rkcypb.Directive_CONSUMER_START,
		&rkcypb.ConsumerDirective{
			Program: &rkcypb.Program{
				Name:   "./" + platform,
				Args:   args,
				Abbrev: "watch",
				Tags:   map[string]string{"service.name": fmt.Sprintf("rkcy.%s.%s.watch", platform, environment)},
			},
		},
		printCh,
	)
}

func (rkcycmd *RkcyCmd) RunConsumerPrograms(
	ctx context.Context,
	wg *sync.WaitGroup,
	strmprov rkcy.StreamProvider,
	platform string,
	environment string,
	adminBrokers string,
	otelcolEndpoint string,
	watchDecode bool,
) {
	defer wg.Done()

	consCh := make(chan *mgmt.ConsumerMessage)

	mgmt.ConsumeConsumersTopic(
		ctx,
		wg,
		strmprov,
		platform,
		environment,
		adminBrokers,
		consCh,
		nil,
	)

	running := make(map[string]program.Runnable)

	printCh := make(chan string, 100)
	wg.Add(1)
	go printer(ctx, wg, printCh)
	startAdmin(ctx, wg, platform, environment, strmprov.Type(), adminBrokers, otelcolEndpoint, running, rkcycmd.newRunnableFunc, printCh)
	startPortalServer(ctx, wg, platform, environment, strmprov.Type(), adminBrokers, otelcolEndpoint, running, rkcycmd.newRunnableFunc, printCh)
	startWatch(ctx, wg, platform, environment, strmprov.Type(), adminBrokers, otelcolEndpoint, watchDecode, running, rkcycmd.newRunnableFunc, printCh)

	ticker := time.NewTicker(1000 * time.Millisecond)

	for {
		select {
		case <-ctx.Done():
			for _, rnbl := range running {
				rnbl.Kill()
			}
			return
		case <-ticker.C:
			doMaintenance(ctx, wg, running, printCh)
		case consMsg := <-consCh:
			if (consMsg.Directive & rkcypb.Directive_CONSUMER) != rkcypb.Directive_CONSUMER {
				log.Error().Msgf("Invalid directive for ConsumersTopic: %s", consMsg.Directive.String())
				continue
			}
			updateRunning(
				ctx,
				wg,
				running,
				rkcycmd.newRunnableFunc,
				consMsg.Directive,
				consMsg.ConsumerDirective,
				printCh,
			)
		}
	}
}
