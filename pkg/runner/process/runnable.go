// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package process

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/lachlanorr/rocketcycle/pkg/runner/program"
)

type Runnable struct {
	details *program.Details
	cmd     *exec.Cmd
	stdout  io.ReadCloser
	stderr  io.ReadCloser
}

func readOutput(printCh chan<- string, rdr io.ReadCloser, key string, abbrev string) {
	scanner := bufio.NewScanner(rdr)

	for scanner.Scan() {
		printCh <- abbrev + scanner.Text()
	}
}

func NewRunnable(
	ctx context.Context,
	dets *program.Details,
) (program.Runnable, error) {
	rnbl := &Runnable{
		details: dets,
	}

	rnbl.cmd = exec.CommandContext(ctx, rnbl.details.Program.Name, rnbl.details.Program.Args...)

	if rnbl.details.Program.Tags != nil {
		otelResourceAttrs := "OTEL_RESOURCE_ATTRIBUTES="
		for k, v := range rnbl.details.Program.Tags {
			otelResourceAttrs += fmt.Sprintf("%s=%s,", k, v)
		}
		otelResourceAttrs = otelResourceAttrs[:len(otelResourceAttrs)-1]
		rnbl.cmd.Env = os.Environ()
		rnbl.cmd.Env = append(rnbl.cmd.Env, otelResourceAttrs)
	}

	var err error
	rnbl.stdout, err = rnbl.cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	rnbl.stderr, err = rnbl.cmd.StderrPipe()
	if err != nil {
		rnbl.stdout.Close()
		return nil, err
	}

	return rnbl, nil
}

func (rnbl *Runnable) Details() *program.Details {
	return rnbl.details
}

func (rnbl *Runnable) Kill() bool {
	// try to 'kill' gracefully
	stopped := rnbl.Stop()

	if !stopped {
		if rnbl.cmd != nil && rnbl.cmd.Process != nil && rnbl.cmd.Process.Pid != 0 {
			proc, err := os.FindProcess(rnbl.cmd.Process.Pid)
			if err != nil {
				proc.Kill()
				return true
			}
		}
	}
	return false
}

func (rnbl *Runnable) Stop() bool {
	if rnbl.cmd != nil && rnbl.cmd.Process != nil && rnbl.cmd.Process.Pid != 0 {
		proc, err := os.FindProcess(rnbl.cmd.Process.Pid)
		if err == nil && proc != nil {
			err = proc.Signal(syscall.SIGINT)
			if err == nil {
				return true
			}
		}
	}
	return false
}

func (rnbl *Runnable) IsRunning() bool {
	if rnbl.cmd != nil && rnbl.cmd.Process != nil && rnbl.cmd.Process.Pid != 0 {
		proc, err := os.FindProcess(rnbl.cmd.Process.Pid)
		if err == nil && proc != nil {
			err = proc.Signal(syscall.SIGCONT)
			if err == nil {
				return true
			}
		}
	}
	return false
}

func (rnbl *Runnable) Wait() {
	err := rnbl.cmd.Wait()
	if err != nil && err.Error() != "signal: killed" {
		log.Error().
			Err(err).
			Str("Program", rnbl.details.Abbrev).
			Msgf("Wait returned error")
	}
}

func (rnbl *Runnable) Start(
	ctx context.Context,
	wg *sync.WaitGroup,
	printCh chan<- string,
	killIfActive bool,
) error {
	if killIfActive {
		rnbl.Kill()
	}

	var err error

	err = rnbl.cmd.Start()
	if err != nil {
		rnbl.stdout.Close()
		rnbl.stderr.Close()
		return err
	}
	rnbl.details.RunCount++

	go readOutput(printCh, rnbl.stdout, rnbl.details.Key, rnbl.details.Abbrev)
	go readOutput(printCh, rnbl.stderr, rnbl.details.Key, rnbl.details.Abbrev)
	go rnbl.Wait()
	return nil
}
