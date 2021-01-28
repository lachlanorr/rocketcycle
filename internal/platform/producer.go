// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package platform

import (
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type Producer struct {
	prod *kafka.Producer
}

//func NewProducer(RtPlatform * rtPlat
