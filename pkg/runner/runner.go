// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/lachlanorr/rocketcycle/pkg/consumer"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type rtProgram struct {
	program     *rkcypb.Program
	key         string
	color       int
	abbrev      string
	abbrevColor string

	cmd      *exec.Cmd
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	runCount int
}

var gCurrColorIdx int = 0

func newRtProgram(program *rkcypb.Program, key string) *rtProgram {
	rtProg := &rtProgram{
		program: program,
		key:     key,
	}
	rtProg.color = gColors[gCurrColorIdx%len(gColors)]
	gCurrColorIdx++
	rtProg.abbrev = colorize(fmt.Sprintf("%-20s |  ", rtProg.program.Abbrev), rtProg.color)

	return rtProg
}

func (rtProg *rtProgram) kill() bool {
	// try to 'kill' gracefully
	stopped := rtProg.stop()

	if !stopped {
		if rtProg.cmd != nil && rtProg.cmd.Process != nil && rtProg.cmd.Process.Pid != 0 {
			proc, err := os.FindProcess(rtProg.cmd.Process.Pid)
			if err != nil {
				proc.Kill()
				return true
			}
		}
	}
	return false
}

func (rtProg *rtProgram) stop() bool {
	if rtProg.cmd != nil && rtProg.cmd.Process != nil && rtProg.cmd.Process.Pid != 0 {
		proc, err := os.FindProcess(rtProg.cmd.Process.Pid)
		if err == nil && proc != nil {
			err = proc.Signal(syscall.SIGINT)
			if err == nil {
				return true
			}
		}
	}
	return false
}

func (rtProg *rtProgram) isRunning() bool {
	if rtProg.cmd != nil && rtProg.cmd.Process != nil && rtProg.cmd.Process.Pid != 0 {
		proc, err := os.FindProcess(rtProg.cmd.Process.Pid)
		if err == nil && proc != nil {
			err = proc.Signal(syscall.SIGCONT)
			if err == nil {
				return true
			}
		}
	}
	return false
}

func (rtProg *rtProgram) wait() {
	err := rtProg.cmd.Wait()
	if err != nil && err.Error() != "signal: killed" {
		log.Error().
			Err(err).
			Str("Program", rtProg.program.Abbrev).
			Msgf("Wait returned error")
	}
}

func (rtProg *rtProgram) start(
	ctx context.Context,
	key string,
	printCh chan<- string,
	killIfActive bool,
) error {
	if killIfActive {
		rtProg.kill()
	}

	rtProg.cmd = exec.CommandContext(ctx, rtProg.program.Name, rtProg.program.Args...)

	if rtProg.program.Tags != nil {
		otelResourceAttrs := "OTEL_RESOURCE_ATTRIBUTES="
		for k, v := range rtProg.program.Tags {
			otelResourceAttrs += fmt.Sprintf("%s=%s,", k, v)
		}
		otelResourceAttrs = otelResourceAttrs[:len(otelResourceAttrs)-1]
		rtProg.cmd.Env = os.Environ()
		rtProg.cmd.Env = append(rtProg.cmd.Env, otelResourceAttrs)
	}

	// only reset color and abbrev if this is the first time through
	if rtProg.abbrev == "" {
		rtProg.color = gColors[gCurrColorIdx%len(gColors)]
		gCurrColorIdx++
		rtProg.abbrev = colorize(fmt.Sprintf("%-13s |  ", rtProg.program.Abbrev), rtProg.color)
	}

	var err error

	rtProg.stdout, err = rtProg.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	rtProg.stderr, err = rtProg.cmd.StderrPipe()
	if err != nil {
		rtProg.stdout.Close()
		return err
	}
	err = rtProg.cmd.Start()
	if err != nil {
		rtProg.stdout.Close()
		rtProg.stderr.Close()
		return err
	}
	rtProg.runCount++

	go readOutput(printCh, rtProg.stdout, key, rtProg.abbrev)
	go readOutput(printCh, rtProg.stderr, key, rtProg.abbrev)
	go rtProg.wait()
	return nil
}

func updateRunning(
	ctx context.Context,
	running map[string]*rtProgram,
	directive rkcypb.Directive,
	acd *rkcypb.ConsumerDirective,
	printCh chan<- string,
) {
	key := rkcy.ProgKey(acd.Program)
	var (
		rtProg *rtProgram
		ok     bool
		err    error
	)

	switch directive {
	case rkcypb.Directive_CONSUMER_START:
		rtProg, ok = running[key]
		if ok {
			log.Warn().
				Msg("Program already running: " + key)
			return
		}
		rtProg = newRtProgram(acd.Program, key)
		err = rtProg.start(ctx, key, printCh, true)
		if err != nil {
			log.Error().
				Err(err).
				Str("Program", rtProg.program.Abbrev).
				Msg("Unable to start")
			return
		}
		running[key] = rtProg
	case rkcypb.Directive_CONSUMER_STOP:
		rtProg, ok = running[key]
		if !ok {
			log.Warn().Msg("Program not running running, cannot stop: " + key)
			return
		} else {
			delete(running, key)
			rtProg.kill()
		}
	}
}

func printer(ctx context.Context, printCh <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case line := <-printCh:
			fmt.Println(line)
		}
	}
}

func readOutput(printCh chan<- string, rdr io.ReadCloser, key string, abbrev string) {
	scanner := bufio.NewScanner(rdr)

	for scanner.Scan() {
		printCh <- abbrev + scanner.Text()
	}
}

func doMaintenance(ctx context.Context, running map[string]*rtProgram, printCh chan<- string) {
	for key, rtProg := range running {
		if !rtProg.isRunning() {
			rtProg.start(ctx, key, printCh, false)
		}
	}
}

func defaultArgs(adminBrokers string, otelcolEndpoint string) []string {
	return []string{
		"--admin_brokers=" + adminBrokers,
		"--otelcol_endpoint=" + otelcolEndpoint,
	}
}

func startAdmin(
	ctx context.Context,
	adminBrokers string,
	otelcolEndpoint string,
	platformName string,
	environment string,
	running map[string]*rtProgram,
	printCh chan<- string,
) {
	updateRunning(
		ctx,
		running,
		rkcypb.Directive_CONSUMER_START,
		&rkcypb.ConsumerDirective{
			Program: &rkcypb.Program{
				Name:   "./" + platformName,
				Args:   append([]string{"admin"}, defaultArgs(adminBrokers, otelcolEndpoint)...),
				Abbrev: "admin",
				Tags:   map[string]string{"service.name": fmt.Sprintf("rkcy.%s.%s.admin", platformName, environment)},
			},
		},
		printCh,
	)
}

func startPortalServer(
	ctx context.Context,
	adminBrokers string,
	otelcolEndpoint string,
	platformName string,
	environment string,
	running map[string]*rtProgram,
	printCh chan<- string,
) {
	updateRunning(
		ctx,
		running,
		rkcypb.Directive_CONSUMER_START,
		&rkcypb.ConsumerDirective{
			Program: &rkcypb.Program{
				Name:   "./" + platformName,
				Args:   append([]string{"portal", "serve"}, defaultArgs(adminBrokers, otelcolEndpoint)...),
				Abbrev: "portal",
				Tags:   map[string]string{"service.name": fmt.Sprintf("rkcy.%s.%s.portal", platformName, environment)},
			},
		},
		printCh,
	)
}

func startWatch(
	ctx context.Context,
	watchDecode bool,
	adminBrokers string,
	otelcolEndpoint string,
	platformName string,
	environment string,
	running map[string]*rtProgram,
	printCh chan<- string,
) {
	args := []string{"watch"}
	if watchDecode {
		args = append(args, "-d")
	}
	args = append(args, defaultArgs(adminBrokers, otelcolEndpoint)...)

	updateRunning(
		ctx,
		running,
		rkcypb.Directive_CONSUMER_START,
		&rkcypb.ConsumerDirective{
			Program: &rkcypb.Program{
				Name:   "./" + platformName,
				Args:   args,
				Abbrev: "watch",
				Tags:   map[string]string{"service.name": fmt.Sprintf("rkcy.%s.%s.watch", platformName, environment)},
			},
		},
		printCh,
	)
}

func RunConsumerPrograms(
	ctx context.Context,
	watchDecode bool,
	adminBrokers string,
	otelcolEndpoint string,
	platformName string,
	environment string,
	wg *sync.WaitGroup,
) {
	consCh := make(chan *consumer.ConsumerMessage)

	consumer.ConsumeConsumersTopic(
		ctx,
		consCh,
		adminBrokers,
		platformName,
		environment,
		nil,
		wg,
	)

	running := map[string]*rtProgram{}

	printCh := make(chan string, 100)
	wg.Add(1)
	go printer(ctx, printCh, wg)
	startAdmin(ctx, adminBrokers, otelcolEndpoint, platformName, environment, running, printCh)
	startPortalServer(ctx, adminBrokers, otelcolEndpoint, platformName, environment, running, printCh)
	startWatch(ctx, watchDecode, adminBrokers, otelcolEndpoint, platformName, environment, running, printCh)

	ticker := time.NewTicker(1000 * time.Millisecond)

	for {
		select {
		case <-ctx.Done():
			for _, prog := range running {
				prog.kill()
			}
			return
		case <-ticker.C:
			doMaintenance(ctx, running, printCh)
		case consMsg := <-consCh:
			if (consMsg.Directive & rkcypb.Directive_CONSUMER) != rkcypb.Directive_CONSUMER {
				log.Error().Msgf("Invalid directive for ConsumersTopic: %s", consMsg.Directive.String())
				continue
			}
			updateRunning(
				ctx,
				running,
				consMsg.Directive,
				consMsg.ConsumerDirective,
				printCh,
			)
		}
	}
}

func Start(plat rkcy.Platform, watchDecode bool) {
	ctx, cancel := context.WithCancel(context.Background())
	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	defer func() {
		signal.Stop(interruptCh)
		cancel()
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go RunConsumerPrograms(
		ctx,
		watchDecode,
		plat.AdminBrokers(),
		plat.Telem().OtelcolEndpoint,
		plat.Name(),
		plat.Environment(),
		&wg,
	)
	for {
		select {
		case <-interruptCh:
			cancel()
			// We don't do wg.Wait() to exit quickly and we are readonly
			return
		}
	}
}
