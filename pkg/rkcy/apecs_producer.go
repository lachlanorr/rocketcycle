// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
			Str("BootstrapServers", bootstrapServers).
			Str("Platform", platformName).
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

func (prod *ApecsProducer) Process(key []byte, value []byte) {
	prod.process.Produce(key, value, nil)
}
