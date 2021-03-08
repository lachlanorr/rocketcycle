// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	//	"context"
	"fmt"
	//	"time"

	//	"github.com/jackc/pgx/v4"
	// "github.com/rs/zerolog/log"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rkcy/version"
)

func cobraStorage(cmd *cobra.Command, args []string) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("storage started")

	// read from kafka message queue
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost",
		"group.id":          "myGroup",
		"auto.offset.reset": "earliest",
	})

	if err != nil {
		panic(err)
	}

	c.SubscribeTopics([]string{"myTopic", "^aRegex.*[Tt]opic"}, nil)

	for {
		msg, err := c.ReadMessage(-1)
		if err == nil {
			fmt.Printf("Message on %s: %s\n", msg.TopicPartition, string(msg.Value))
		} else {
			// The client will automatically try to recover from all errors.
			fmt.Printf("Consumer error: %v (%v)\n", err, msg)
		}
	}

	c.Close()

	/*


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
	*/
}
