// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func cobraStorage(cmd *cobra.Command, args []string) {
	conn, err := pgx.Connect(context.Background(), "postgresql://postgres@127.0.0.1:5432/rpg")
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Unable to connect to database")
	}
	defer conn.Close(context.Background())

	var username string
	var active bool
	err = conn.QueryRow(context.Background(), "select username, active from rpg.player where id=$1", 1).Scan(&username, &active)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("QueryRow failed")
	}

	log.Info().Msg(fmt.Sprintf("username = %s, active = %t", username, active))

	time.Sleep(30 * time.Second)

	log.Info().Msg("rcstore exit")
}
