// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"errors"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rkcy/pkg/rkcy/pb"
)

type ApecsProducer struct {
	platformName string
	concernName  string
	slog         zerolog.Logger

	process *Producer
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

	txnSer, err := proto.Marshal(txn)
	if err != nil {
		return err
	}

	prod.process.Produce(pb.Directive_APECS_TXN, []byte(step.Key), txnSer, nil)
	return nil
}
