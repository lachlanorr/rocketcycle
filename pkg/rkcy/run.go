// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type rtProgram struct {
	program     *Program
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

func newRtProgram(program *Program, key string) *rtProgram {
	rtProg := &rtProgram{
		program: program,
		key:     key,
	}
	rtProg.color = gColors[gCurrColorIdx%len(gColors)]
	gCurrColorIdx++
	rtProg.abbrev = colorize(fmt.Sprintf("%-13s |  ", rtProg.program.Abbrev), rtProg.color)

	return rtProg
}

func progKey(prog *Program) string {
	// Combine name and args to a string for key lookup
	if prog == nil {
		return "NIL"
	} else {
		return prog.Name + " " + strings.Join(prog.Args, " ")
	}
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
	if err != nil {
		log.Error().
			Err(err).
			Str("Program", rtProg.program.Abbrev).
			Msg("Wait returned error")
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
	directive Directive,
	acd *ConsumerDirective,
	printCh chan<- string,
) {
	key := progKey(acd.Program)
	var (
		rtProg *rtProgram
		ok     bool
		err    error
	)

	switch directive {
	case Directive_CONSUMER_START:
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
	case Directive_CONSUMER_STOP:
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

func printer(ctx context.Context, printCh <-chan string) {
	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("printer exiting, ctx.Done()")
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

func startAdminServer(ctx context.Context, running map[string]*rtProgram, printCh chan<- string) {
	updateRunning(
		ctx,
		running,
		Directive_CONSUMER_START,
		&ConsumerDirective{
			Program: &Program{
				Name:   "./" + gPlatformName,
				Args:   []string{"admin", "serve"},
				Abbrev: "admin",
				Tags:   map[string]string{"service.name": fmt.Sprintf("rkcy.%s.admin", gPlatformImpl.Name)},
			},
		},
		printCh,
	)
}

func startWatch(ctx context.Context, running map[string]*rtProgram, printCh chan<- string) {
	args := []string{"watch"}
	if gSettings.WatchDecode {
		args = append(args, "-d")
	}

	updateRunning(
		ctx,
		running,
		Directive_CONSUMER_START,
		&ConsumerDirective{
			Program: &Program{
				Name:   "./" + gPlatformName,
				Args:   args,
				Abbrev: "watch",
				Tags:   map[string]string{"service.name": fmt.Sprintf("rkcy.%s.watch", gPlatformImpl.Name)},
			},
		},
		printCh,
	)
}

func runConsumerPrograms(ctx context.Context) {
	adminCh := make(chan *AdminMessage)

	go consumeAdminTopic(
		ctx,
		adminCh,
		gSettings.AdminBrokers,
		gPlatformName,
		Directive_CONSUMER,
		Directive_CONSUMER,
		kPastLastMatch,
	)

	running := map[string]*rtProgram{}

	printCh := make(chan string, 100)
	go printer(ctx, printCh)
	startAdminServer(ctx, running, printCh)
	startWatch(ctx, running, printCh)

	ticker := time.NewTicker(1000 * time.Millisecond)

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("runConsumerPrograms exiting, ctx.Done()")
			return
		case <-ticker.C:
			doMaintenance(ctx, running, printCh)
		case adminMsg := <-adminCh:
			updateRunning(
				ctx,
				running,
				adminMsg.Directive,
				adminMsg.ConsumerDirective,
				printCh,
			)
		}
	}
}

func cobraRun(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	interruptCh := make(chan os.Signal)
	signal.Notify(interruptCh, os.Interrupt)

	go runConsumerPrograms(ctx)
	for {
		select {
		case <-interruptCh:
			cancel()
			return
		}
	}
}
