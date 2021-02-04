// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

var exists struct{}

func PrepLogging() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02T15:04:05.999"})
}

func contains(slice []string, item string) bool {
	for _, val := range slice {
		if val == item {
			return true
		}
	}
	return false
}

func maxi(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func mini(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func maxi64(x, y int64) int64 {
	if x < y {
		return y
	}
	return x
}

func mini64(x, y int64) int64 {
	if x > y {
		return y
	}
	return x
}

func findHeader(msg *kafka.Message, key string) []byte {
	for _, hdr := range msg.Headers {
		if key == hdr.Key {
			return hdr.Value
		}
	}
	return nil
}

func adminTopic(internalName string) string {
	return fmt.Sprintf("rc.%s.admin", internalName)
}

func createAdminTopic(ctx context.Context, bootstrapServers string, internalName string) (string, error) {
	topicName := adminTopic(internalName)

	// connect to kafka and make sure we have our platform topic
	admin, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapServers,
	})
	if err != nil {
		return "", errors.New("Failed to NewAdminClient")
	}

	md, err := admin.GetMetadata(nil, true, 1000)
	if err != nil {
		return "", errors.New("Failed to GetMetadata")
	}

	_, ok := md.Topics[topicName]
	if !ok { // platform topic doesn't exist
		result, err := admin.CreateTopics(
			context.Background(),
			[]kafka.TopicSpecification{
				{
					Topic:             topicName,
					NumPartitions:     1,
					ReplicationFactor: len(md.Brokers),
					Config: map[string]string{
						"retention.ms":    "-1",
						"retention.bytes": strconv.Itoa(10 * 1024 * 1024),
					},
				},
			},
			nil,
		)
		if err != nil {
			return "", errors.New("Failed to create metadata topic")
		}
		for _, res := range result {
			if res.Error.Code() != kafka.ErrNoError {
				return "", errors.New("Failed to create metadata topic")
			}
		}
	}
	return topicName, nil
}

func msgTypeName(msg proto.Message) string {
	msgR := msg.ProtoReflect()
	desc := msgR.Descriptor()
	return string(desc.FullName())
}

func msgTypeHeaders(msg proto.Message) []kafka.Header {
	return []kafka.Header{{Key: "type", Value: []byte(msgTypeName(msg))}}
}

const (
	colorBlack = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
	colorCyan
	colorWhite

	colorBold     = 1
	colorDarkGray = 90
)

func colorize(s interface{}, c int) string {
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}
