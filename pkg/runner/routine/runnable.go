// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package routine

import (
	"context"
	"errors"

	"github.com/lachlanorr/rocketcycle/pkg/runner/program"
)

type Runnable struct {
	details *program.Details
	ctx     context.Context
	cancel  context.CancelFunc
}

var gExecuteFunc func(ctx context.Context, args []string)

func SetExecuteFunc(f func(ctx context.Context, args []string)) {
	gExecuteFunc = f
}

func NewRunnable(
	ctx context.Context,
	dets *program.Details,
) (program.Runnable, error) {
	if gExecuteFunc == nil {
		return nil, errors.New("SetExecuteFunc has not been called")
	}

	rnbl := &Runnable{
		details: dets,
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
		return
	}
}

func (rnbl *Runnable) Start(
	ctx context.Context,
	printCh chan<- string,
	killIfActive bool,
) error {
	if killIfActive {
		rnbl.Kill()
	}

	rnbl.ctx, rnbl.cancel = context.WithCancel(ctx)

	go gExecuteFunc(rnbl.ctx, rnbl.details.Program.Args)

	go rnbl.Wait()
	return nil
}
