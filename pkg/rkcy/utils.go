// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

var gExists struct{}

func NewTraceId() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

func NewSpanId() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
}

func prepLogging(platformName string) {
	logLevel := os.Getenv("RKCY_LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	badParse := false
	lvl, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		lvl = zerolog.InfoLevel
		badParse = true
	}

	zerolog.SetGlobalLevel(lvl)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02T15:04:05.999"})

	if badParse {
		log.Error().
			Msgf("Bad value for RKCY_LOG_LEVEL: %s", os.Getenv("RKCY_LOG_LEVEL"))
	}
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

func maxi32(x, y int32) int32 {
	if x < y {
		return y
	}
	return x
}

func mini32(x, y int32) int32 {
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

func GetDirective(msg *kafka.Message) rkcypb.Directive {
	val := findHeader(msg, DIRECTIVE_HEADER)
	if val != nil {
		return rkcypb.Directive(BytesToInt(val))
	} else {
		return rkcypb.Directive_UNSPECIFIED
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

func PlatformTopic(platformName string, environment string) string {
	return fmt.Sprintf("%s.%s.%s.platform", RKCY, platformName, environment)
}

func ConfigTopic(platformName string, environment string) string {
	return fmt.Sprintf("%s.%s.%s.config", RKCY, platformName, environment)
}

func ProducersTopic(platformName string, environment string) string {
	return fmt.Sprintf("%s.%s.%s.producers", RKCY, platformName, environment)
}

func ConsumersTopic(platformName string, environment string) string {
	return fmt.Sprintf("%s.%s.%s.consumers", RKCY, platformName, environment)
}

func librdkafkaToZerologLevel(kafkaLevel int) zerolog.Level {
	switch kafkaLevel {
	case 7:
		return zerolog.DebugLevel
	case 6:
		fallthrough
	case 5:
		return zerolog.InfoLevel
	case 4:
		return zerolog.WarnLevel
	default:
		return zerolog.ErrorLevel
	}
}

func printKafkaLogs(ctx context.Context, kafkaLogCh <-chan kafka.LogEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case logEvt := <-kafkaLogCh:
			log.WithLevel(librdkafkaToZerologLevel(logEvt.Level)).
				Str("Name", logEvt.Name).
				Str("Tag", logEvt.Tag).
				Int("Level", logEvt.Level).
				Str("Timestamp", logEvt.Timestamp.Format(time.RFC3339)).
				Msgf("Kafka Log: %s", logEvt.Message)
		}
	}
}

func createPlatformTopics(
	ctx context.Context,
	bootstrapServers string,
	platformName string,
	environment string,
) error {
	topicNames := []string{
		PlatformTopic(platformName, environment),
		ConfigTopic(platformName, environment),
		ProducersTopic(platformName, environment),
		ConsumersTopic(platformName, environment),
	}

	// connect to kafka and make sure we have our platform topic
	admin, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapServers,
	})
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
						ReplicationFactor: mini(3, len(md.Brokers)),
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

func standardHeaders(directive rkcypb.Directive, traceParent string) []kafka.Header {
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
	31 /*47,*/ /*63,*/, 79, 95 /*111,*/, 127, 143, 159, 175 /*191,*/, 207, 223,
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

func OffsetGT(lhs *rkcypb.CompoundOffset, rhs *rkcypb.CompoundOffset) bool {
	if lhs.Generation == rhs.Generation {
		if lhs.Partition == rhs.Partition {
			return lhs.Offset > rhs.Offset
		}
		return lhs.Partition > rhs.Partition
	} else {
		return lhs.Generation > rhs.Generation
	}
}

func OffsetGTE(lhs *rkcypb.CompoundOffset, rhs *rkcypb.CompoundOffset) bool {
	if lhs.Generation == rhs.Generation {
		if lhs.Partition == rhs.Partition {
			return lhs.Offset >= rhs.Offset
		}
		return lhs.Partition >= rhs.Partition
	} else {
		return lhs.Generation >= rhs.Generation
	}
}

func (kplat *KafkaPlatform) cobraDecodeInstance(cmd *cobra.Command, args []string) {
	concern := args[0]
	instance64 := args[1]

	ctx := context.Background()

	jsonBytes, err := kplat.concernHandlers.decodeInstance64Json(ctx, concern, instance64)
	if err != nil {
		fmt.Printf("Failed to decode: %s\n", err.Error())
		os.Stderr.WriteString(fmt.Sprintf("Failed to decode: %s\n", err.Error()))
		os.Exit(1)
	}

	var indentJson bytes.Buffer
	err = json.Indent(&indentJson, jsonBytes, "", "  ")
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Failed to json.Indent: %s\n", err.Error()))
		os.Exit(1)
	}

	fmt.Println(string(indentJson.Bytes()))
}

func LogProto(msg proto.Message) {
	reqJson := protojson.Format(msg)
	log.Info().Msg(reqJson)
}
