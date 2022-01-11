// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package runner

import (
	"context"
	"fmt"

	"github.com/lachlanorr/rocketcycle/pkg/runner/process"
	"github.com/lachlanorr/rocketcycle/pkg/runner/program"
)

type NewRunnableFunc func(
	ctx context.Context,
	dets *program.Details,
) (program.Runnable, error)

var gRunnableRegistry = make(map[string]NewRunnableFunc)

func init() {
	RegisterNewRunnableFunc("process", process.NewRunnable)
}

func RegisterNewRunnableFunc(name string, newRunnableFunc NewRunnableFunc) {
	_, ok := gRunnableRegistry[name]
	if ok {
		panic(fmt.Sprintf("RunnerStartFunc already registered: %s", name))
	}

	gRunnableRegistry[name] = newRunnableFunc
}

func GetNewRunnableFunc(name string) NewRunnableFunc {
	newRunnableFunc, ok := gRunnableRegistry[name]
	if !ok {
		panic(fmt.Sprintf("RunnerStartFunc not found: %s", name))
	}

	return newRunnableFunc
}
