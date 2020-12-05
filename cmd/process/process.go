// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v4"

	"github.com/lachlanorr/gaeneco/internal/dispatch"
)

func main() {

	dispatch.Process()

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	var username string
	var active bool
	err = conn.QueryRow(context.Background(), "select username, active from oltp.player where id=$1", 1).Scan(&username, &active)
	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(username, active)
}
