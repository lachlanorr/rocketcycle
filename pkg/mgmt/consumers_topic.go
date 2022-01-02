// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package mgmt

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type ConsumerMessage struct {
	Directive         rkcypb.Directive
	Timestamp         time.Time
	Offset            int64
	ConsumerDirective *rkcypb.ConsumerDirective
}

func ConsumeConsumersTopic(
	ctx context.Context,
	wg *sync.WaitGroup,
	strmprov rkcy.StreamProvider,
	platform string,
	environment string,
	adminBrokers string,
	ch chan<- *ConsumerMessage,
	readyCh chan<- bool,
) {
	wg.Add(1)
	go ConsumeMgmtTopic(
		ctx,
		wg,
		strmprov,
		adminBrokers,
		rkcy.ConsumersTopic(platform, environment),
		rkcypb.Directive_CONSUMER,
		PAST_LAST_MATCH,
		func(rawMsg *RawMessage) {
			consDir := &rkcypb.ConsumerDirective{}
			err := proto.Unmarshal(rawMsg.Value, consDir)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to Unmarshal ConsumerDirective")
				return
			}

			ch <- &ConsumerMessage{
				Directive:         rawMsg.Directive,
				Timestamp:         rawMsg.Timestamp,
				Offset:            rawMsg.Offset,
				ConsumerDirective: consDir,
			}
		},
		readyCh,
	)
}
