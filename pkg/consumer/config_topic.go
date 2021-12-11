// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package consumer

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type ConfigPublishMessage struct {
	Directive rkcypb.Directive
	Timestamp time.Time
	Offset    int64
	Config    *rkcypb.Config
}

func ConsumeConfigTopic(
	ctx context.Context,
	plat rkcy.Platform,
	chPublish chan<- *ConfigPublishMessage,
	readyCh chan<- bool,
	wg *sync.WaitGroup,
) {
	wg.Add(1)
	go ConsumeMgmtTopic(
		ctx,
		plat,
		rkcy.ConfigTopic(plat.Name(), plat.Environment()),
		rkcypb.Directive_CONFIG,
		AT_LAST_MATCH,
		func(rawMsg *RawMessage) {
			if chPublish != nil && (rawMsg.Directive&rkcypb.Directive_CONFIG_PUBLISH) == rkcypb.Directive_CONFIG_PUBLISH {
				conf := &rkcypb.Config{}
				err := proto.Unmarshal(rawMsg.Value, conf)
				if err != nil {
					log.Error().
						Err(err).
						Msg("Failed to Unmarshal Config")
					return
				}

				chPublish <- &ConfigPublishMessage{
					Directive: rawMsg.Directive,
					Timestamp: rawMsg.Timestamp,
					Offset:    rawMsg.Offset,
					Config:    conf,
				}
			}
		},
		readyCh,
		wg,
	)
}
