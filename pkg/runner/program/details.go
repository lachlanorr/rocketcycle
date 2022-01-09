// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package program

import (
	"context"
	"fmt"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type NewRunnableFunc func(
	ctx context.Context,
	dets *Details,
) (Runnable, error)

type Runnable interface {
	Details() *Details

	Kill() bool
	Stop() bool
	IsRunning() bool
	Wait()
	Start(
		ctx context.Context,
		printCh chan<- string,
		killIfActive bool,
	) error
}

type Details struct {
	Program *rkcypb.Program
	Key     string

	Color       int
	Abbrev      string
	AbbrevColor string
	RunCount    int
}

var gCurrColorIdx int = 0

func NewDetails(
	program *rkcypb.Program,
	key string,
) *Details {
	dets := &Details{
		Program: program,
		Key:     key,
	}
	dets.Color = gColors[gCurrColorIdx%len(gColors)]
	gCurrColorIdx++
	dets.Abbrev = colorize(fmt.Sprintf("%-20s |  ", dets.Program.Abbrev), dets.Color)

	return dets
}
