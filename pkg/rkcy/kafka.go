// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"fmt"
	"time"
	"unsafe"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

func UncommittedGroupName(topic string, partition int) string {
	return fmt.Sprintf("__%s_%d__non_comitted_group", topic, partition)
}

func UncommittedGroupNameAllPartitions(topic string) string {
	return fmt.Sprintf("__%s_ALL__non_comitted_group", topic)
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

func PrintKafkaLogs(ctx context.Context, kafkaLogCh <-chan kafka.LogEvent) {
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

func StandardHeaders(directive rkcypb.Directive, traceParent string) []kafka.Header {
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

func NewKafkaMessage(
	topic *string,
	partition int32,
	value proto.Message,
	directive rkcypb.Directive,
	traceParent string,
) (*kafka.Message, error) {
	valueSer, err := proto.Marshal(value)
	if err != nil {
		return nil, err
	}
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: topic, Partition: partition},
		Value:          valueSer,
		Headers:        StandardHeaders(directive, traceParent),
	}
	return msg, nil
}
