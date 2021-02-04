// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"

	"github.com/lachlanorr/rocketcycle/internal/rkcy"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	admin_pb "github.com/lachlanorr/rocketcycle/build/proto/admin"
)

func Run() {
	rkcy.Run()
}

func ServeGrpcGateway(ctx context.Context, srv interface{}) {
	rkcy.ServeGrpcGateway(ctx, srv)
}

func PrepLogging() {
	rkcy.PrepLogging()
}

func ConsumePlatformConfig(ctx context.Context, ch chan<- admin_pb.Platform, bootstrapServers string, platformName string) {
	rkcy.ConsumePlatformConfig(ctx, ch, bootstrapServers, platformName)
}

type Producer struct {
	rkProd *kafka.Producer
}

func NewProducer(conf *kafka.ConfigMap) (*Producer, error) {
	rdProd, err := kafka.NewProducer(conf)
	if err != nil {
		return nil, err
	}
	prod := Producer{
		rkProd: rdProd,
	}
	return &prod, nil
}

type Consumer struct {
	rkCons *kafka.Consumer
}

func NewConsumer(conf *kafka.ConfigMap) (*Consumer, error) {
	rdCons, err := kafka.NewConsumer(conf)
	if err != nil {
		return nil, err
	}
	cons := Consumer{
		rkCons: rdCons,
	}
	return &cons, nil
}
