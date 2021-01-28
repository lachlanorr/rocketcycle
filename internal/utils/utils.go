// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package utils

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func PrepLogging() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02T15:04:05.999"})
}

func Contains(slice []string, item string) bool {
	for _, val := range slice {
		if val == item {
			return true
		}
	}
	return false
}

func Maxi(x, y int64) int64 {
	if x < y {
		return y
	}
	return x
}

func Mini(x, y int64) int64 {
	if x > y {
		return y
	}
	return x
}
