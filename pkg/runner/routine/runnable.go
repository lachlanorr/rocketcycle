// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package routine

import (
	"context"
	"sync"

	"github.com/lachlanorr/rocketcycle/pkg/runner/program"
)

type Runnable struct {
	details     *program.Details
	executeFunc func(ctx context.Context, args []string)
	ctx         context.Context
	cancel      context.CancelFunc
	wg          *sync.WaitGroup
}

func NewRunnable(
	ctx context.Context,
	dets *program.Details,
	executeFunc func(ctx context.Context, args []string),
) (program.Runnable, error) {
	rnbl := &Runnable{
		details:     dets,
		executeFunc: executeFunc,
	}

	return rnbl, nil
}

func (rnbl *Runnable) Details() *program.Details {
	return rnbl.details
}

func (rnbl *Runnable) Kill() bool {
	if rnbl.cancel != nil {
		rnbl.cancel()
		rnbl.ctx = nil
		rnbl.cancel = nil
	}
	return true
}

func (rnbl *Runnable) Stop() bool {
	return rnbl.Kill()
}

func (rnbl *Runnable) IsRunning() bool {
	return rnbl.ctx != nil
}

func (rnbl *Runnable) Wait() {
	select {
	case <-rnbl.ctx.Done():
		rnbl.wg.Done()
		return
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

	rnbl.ctx, rnbl.cancel = context.WithCancel(ctx)
	rnbl.wg = wg

	go rnbl.executeFunc(rnbl.ctx, rnbl.details.Program.Args)

	wg.Add(1)
	go rnbl.Wait()
	return nil
}
