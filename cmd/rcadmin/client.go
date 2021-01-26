// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func rcadminGetPlatform(cmd *cobra.Command, args []string) {
	path := "/v1/platform/get?pretty"

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(adminAddr + path)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to GET")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to ReadAll")
	}

	fmt.Println(string(body))
}
