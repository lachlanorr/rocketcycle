// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/rs/zerolog/log"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

func CreatePlatformTopics(
	ctx context.Context,
	plat Platform,
) error {
	topicNames := []string{
		PlatformTopic(plat.Name(), plat.Environment()),
		ConfigTopic(plat.Name(), plat.Environment()),
		ProducersTopic(plat.Name(), plat.Environment()),
		ConsumersTopic(plat.Name(), plat.Environment()),
	}

	// connect to kafka and make sure we have our platform topic
	admin, err := plat.NewAdminClient(plat.AdminBrokers())
	if err != nil {
		return errors.New("Failed to NewAdminClient")
	}

	md, err := admin.GetMetadata(nil, true, 1000)
	if err != nil {
		return errors.New("Failed to GetMetadata")
	}

	for _, topicName := range topicNames {
		_, ok := md.Topics[topicName]
		if !ok { // platform topic doesn't exist
			result, err := admin.CreateTopics(
				context.Background(),
				[]kafka.TopicSpecification{
					{
						Topic:             topicName,
						NumPartitions:     1,
						ReplicationFactor: Mini(3, len(md.Brokers)),
						Config: map[string]string{
							"retention.ms":    "-1",
							"retention.bytes": strconv.Itoa(int(PLATFORM_TOPIC_RETENTION_BYTES)),
						},
					},
				},
				nil,
			)
			if err != nil {
				return fmt.Errorf("Failed to create platform topic: %s", topicName)
			}
			for _, res := range result {
				if res.Error.Code() != kafka.ErrNoError {
					return fmt.Errorf("Failed to create platform topic: %s", topicName)
				}
			}
			log.Info().
				Str("Topic", topicName).
				Msg("Platform topic created")
		}
	}
	return nil
}
