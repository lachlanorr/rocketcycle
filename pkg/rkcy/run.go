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
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rkcy/pkg/rkcy/pb"
)

type rtProgram struct {
	program     *pb.Program
	key         string
	color       int
	abbrev      string
	abbrevColor string

	cmd      *exec.Cmd
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	runCount int
}

var currColorIdx int = 0

func newRtProgram(program *pb.Program, key string) *rtProgram {
	rtProg := &rtProgram{
		program: program,
		key:     key,
	}
	rtProg.color = colors[currColorIdx%len(colors)]
	currColorIdx++
	rtProg.abbrev = colorize(fmt.Sprintf("%-13s |  ", rtProg.program.Abbrev), rtProg.color)

	return rtProg
}

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

func progKey(prog *pb.Program) string {
	// Combine name and args to a string for key lookup
	if prog == nil {
		return "NIL"
	} else {
		return prog.Name + " " + strings.Join(prog.Args, " ")
	}
}

func (rtProg *rtProgram) kill() bool {
	if rtProg.cmd != nil && rtProg.cmd.Process != nil && rtProg.cmd.Process.Pid != 0 {
		proc, err := os.FindProcess(rtProg.cmd.Process.Pid)
		if err != nil {
			proc.Kill()
			return true
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

	// only reset color and abbrev if this is the first time through
	if rtProg.abbrev == "" {
		rtProg.color = colors[currColorIdx%len(colors)]
		currColorIdx++
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
	directive pb.Directive,
	acd *pb.AdminConsumerDirective,
	printCh chan<- string,
) {
	key := progKey(acd.Program)

	var (
		rtProg *rtProgram
		ok     bool
		err    error
	)

	switch directive {
	case pb.Directive_ADMIN_CONSUMER_START:
		rtProg, ok = running[key]
		if ok {
			log.Warn().
				Msg("Program already running, doing nothing: " + key)
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
	case pb.Directive_ADMIN_CONSUMER_STOP:
		rtProg, ok = running[key]
		if !ok {
			log.Warn().Msg("Program not running running, doing nothing: " + key)
			return
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

	if err := scanner.Err(); err != nil {
		log.Error().
			Err(err).
			Msg("io error reading: " + key)
	}

	log.Info().Msg("readOutput exiting")
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
		pb.Directive_ADMIN_CONSUMER_START,
		&pb.AdminConsumerDirective{
			Program: &pb.Program{
				Name:   "./" + platformName,
				Args:   []string{"admin", "serve"},
				Abbrev: "admin",
			},
		},
		printCh,
	)
}

func runConsumerPrograms(ctx context.Context, platCh <-chan *pb.Platform) {
	rkcyCh := make(chan *rkcyMessage, 1)
	go consumePlatformAdminTopic(ctx, rkcyCh, settings.BootstrapServers, platformName, pb.Directive_ADMIN_CONSUMER, pb.Directive_ALL)

	running := map[string]*rtProgram{}

	printCh := make(chan string, 100)
	go printer(ctx, printCh)
	startAdminServer(ctx, running, printCh)

	ticker := time.NewTicker(1000 * time.Millisecond)

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("runConsumerPrograms exiting, ctx.Done()")
			return
		case <-ticker.C:
			doMaintenance(ctx, running, printCh)
		case rkcyMsg := <-rkcyCh:
			acd := pb.AdminConsumerDirective{}
			err := proto.Unmarshal(rkcyMsg.Value, &acd)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to Unmarshal AdminProducerDirective")
			} else {
				updateRunning(
					ctx,
					running,
					pb.Directive_ADMIN_CONSUMER_START,
					&acd,
					printCh,
				)
			}
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
	for {
		select {
		case <-interruptCh:
			cancel()
			return
		}
	}
}
