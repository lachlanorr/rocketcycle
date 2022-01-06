// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package runner

import (
	"fmt"

	"github.com/lachlanorr/rocketcycle/pkg/runner/process"
	"github.com/lachlanorr/rocketcycle/pkg/runner/program"
)

var gRunnableRegistry = make(map[string]program.NewRunnableFunc)

func init() {
	RegisterNewRunnableFunc("process", process.NewRunnable)
}

func RegisterNewRunnableFunc(name string, newRunnableFunc program.NewRunnableFunc) {
	_, ok := gRunnableRegistry[name]
	if ok {
		panic(fmt.Sprintf("RunnerStartFunc already registered: %s", name))
	}

	gRunnableRegistry[name] = newRunnableFunc
}

func GetRunnerStartFunc(name string) program.NewRunnableFunc {
	newRunnableFunc, ok := gRunnableRegistry[name]
	if !ok {
		panic(fmt.Sprintf("RunnerStartFunc not found: %s", name))
	}

	return newRunnableFunc
}
