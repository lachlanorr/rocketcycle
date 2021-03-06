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
	"strings"
	"unsafe"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

var gExists struct{}

func NewTraceId() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

func NewSpanId() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
}

func prepLogging(platformName string) {
	if os.Getenv("RKCY_DEBUG") == "1" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
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

func GetDirective(msg *kafka.Message) Directive {
	val := findHeader(msg, DIRECTIVE_HEADER)
	if val != nil {
		return Directive(BytesToInt(val))
	} else {
		return Directive_UNSPECIFIED
	}
}

func GetTraceParent(msg *kafka.Message) string {
	val := findHeader(msg, TRACE_PARENT_HEADER)
	if val != nil {
		return string(val)
	}
	return ""
}

func GetTraceId(msg *kafka.Message) string {
	return TraceIdFromTraceParent(GetTraceParent(msg))
}

func AdminTopic(platformName string) string {
	return fmt.Sprintf("%s.%s.admin", RKCY, platformName)
}

func createAdminTopic(ctx context.Context, bootstrapServers string, internalName string) (string, error) {
	topicName := AdminTopic(internalName)

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
						"retention.bytes": strconv.Itoa(int(PLATFORM_ADMIN_RETENTION_BYTES)),
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

func standardHeaders(directive Directive, traceParent string) []kafka.Header {
	if TraceParentIsValid(traceParent) {
		return []kafka.Header{
			{
				Key:   DIRECTIVE_HEADER,
				Value: IntToBytes(int(directive)),
			},
			{
				Key:   TRACE_PARENT_HEADER,
				Value: []byte(traceParent),
			},
		}
	} else {
		return []kafka.Header{
			{
				Key:   DIRECTIVE_HEADER,
				Value: IntToBytes(int(directive)),
			},
		}
	}
}

const (
	kColorBlack = iota + 30
	kColorRed
	kColorGreen
	kColorYellow
	kColorBlue
	kColorMagenta
	kColorCyan
	kColorWhite

	kColorBold     = 1
	kColorDarkGray = 90
)

// reasonable list of colors that change greatly each time
var gColors []int = []int{
	11, 12, /*13, 14, 10,*/ /*9,*/
	31 /*47,*/ /*63,*/, 79, 95 /*111,*/, 127, 143, 159, 175, 191, 207, 223,
	25, 41, 57, 73, 89, 105, 121, 137, 153, 169, 185, 201, 217,
	26, 42, 58, 74, 90, 106, 122, 138, 154, 170, 186, 202, 218,
	27, 43, 59, 75, 91, 107, 123, 139, 155, 171, 187, 203, 219,
	28, 44, 60, 76, 92, 108, 124, 140, 156, 172, 188, 204, 220,
	29, 45, 61, 77, 93, 109, 125, 141, 157, 173, 189, 205, 221,
}

func colorize(s interface{}, c int) string {
	return fmt.Sprintf("\x1b[38;5;%dm%v\x1b[0m", c, s)
}

func IntToBytes(num int) []byte {
	arr := make([]byte, 4)
	arr[0] = *(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&num)) + uintptr(0)))
	arr[1] = *(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&num)) + uintptr(1)))
	arr[2] = *(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&num)) + uintptr(2)))
	arr[3] = *(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&num)) + uintptr(3)))
	return arr
}

func BytesToInt(arr []byte) int {
	var val int
	*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&val)) + uintptr(0))) = arr[0]
	*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&val)) + uintptr(1))) = arr[1]
	*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&val)) + uintptr(2))) = arr[2]
	*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&val)) + uintptr(3))) = arr[3]
	return val
}

func OffsetGreaterThan(lhs *Offset, rhs *Offset) bool {
	if lhs.Generation == rhs.Generation {
		if lhs.Partition == rhs.Partition {
			return lhs.Offset > rhs.Offset
		}
		return lhs.Partition > rhs.Partition
	} else {
		return lhs.Generation > rhs.Generation
	}
}
