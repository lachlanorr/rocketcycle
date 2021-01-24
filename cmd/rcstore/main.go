// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"time"

	"github.com/rs/zerolog/log"

	"github.com/lachlanorr/rocketcycle/internal/utils"
	"github.com/lachlanorr/rocketcycle/version"
)

func main() {
	utils.PrepLogging()
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("rcstore started")

	time.Sleep(30 * time.Second)

	log.Info().Msg("rcstore exit")
}
