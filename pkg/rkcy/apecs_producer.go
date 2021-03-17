// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy/pb"
)

type ApecsProducer struct {
	platformName string
	concernName  string
	slog         zerolog.Logger

	process *Producer
	storage *Producer
}

func NewApecsProducer(
	ctx context.Context,
	bootstrapServers string,
	platformName string,
	concernName string,
) *ApecsProducer {

	prod := ApecsProducer{
		platformName: platformName,
		concernName:  concernName,
		slog: log.With().
			Str("Concern", concernName).
			Logger(),
	}

	prod.process = NewProducer(ctx, bootstrapServers, platformName, concernName, "process")
	if prod.process == nil {
		prod.slog.Error().
			Msg("Failed to create 'process' Producer")
		return nil
	}

	prod.storage = NewProducer(ctx, bootstrapServers, platformName, concernName, "storage")
	if prod.storage == nil {
		prod.slog.Error().
			Msg("Failed to create 'storage' Producer")
		return nil
	}

	return &prod
}

func (prod *ApecsProducer) Close() {
	prod.process.Close()
}

func (prod *ApecsProducer) Process(txn *pb.ApecsTxn) error {
	step := nextStep(txn)
	if step == nil {
		return errors.New("No 'next' step in ApecsTxn to process")
	}

	if step.ConcernName != prod.concernName {
		return errors.New(fmt.Sprintf("ConcernName mismatch: step=%s producer=%s", step.ConcernName, prod.concernName))
	}

	txnSer, err := proto.Marshal(txn)
	if err != nil {
		return err
	}

	prod.process.Produce(pb.Directive_APECS_TXN, []byte(step.Key), txnSer, nil)
	return nil
}

func (prod *ApecsProducer) Storage(req *pb.ApecsStorageRequest) error {
	if req.ConcernName != prod.concernName {
		return errors.New(fmt.Sprintf("ConcernName mismatch: req=%s producer=%s", req.ConcernName, prod.concernName))
	}

	reqSer, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	prod.storage.Produce(pb.Directive_APECS_STORAGE_REQUEST, []byte(req.Key), reqSer, nil)
	return nil
}
