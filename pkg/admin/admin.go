// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package admin

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/mgmt"
	"github.com/lachlanorr/rocketcycle/pkg/platform"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/pkg/telem"
	"github.com/lachlanorr/rocketcycle/version"
)

type AdminCmd struct {
	plat            *platform.Platform
	producerTracker *ProducerTracker
	otelcolEndpoint string
}

func Start(
	ctx context.Context,
	plat *platform.Platform,
	otelcolEndpoint string,
) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("admin started")

	adminCmd := &AdminCmd{
		plat:            plat,
		producerTracker: NewProducerTracker(plat.Args.Platform, plat.Args.Environment),
		otelcolEndpoint: otelcolEndpoint,
	}

	go managePlatform(ctx, plat.WaitGroup(), adminCmd)

	select {
	case <-ctx.Done():
		return
	}
}

func managePlatform(
	ctx context.Context,
	wg *sync.WaitGroup,
	adminCmd *AdminCmd,
) {
	consumersTopic := rkcy.ConsumersTopic(adminCmd.plat.Args.Platform, adminCmd.plat.Args.Environment)
	adminProdCh := adminCmd.plat.GetProducerCh(ctx, wg, adminCmd.plat.Args.AdminBrokers)

	platCh := make(chan *mgmt.PlatformMessage)
	mgmt.ConsumePlatformTopic(
		ctx,
		wg,
		adminCmd.plat.StreamProvider(),
		adminCmd.plat.Args.Platform,
		adminCmd.plat.Args.Environment,
		adminCmd.plat.Args.AdminBrokers,
		platCh,
		nil,
	)

	prodCh := make(chan *mgmt.ProducerMessage)
	mgmt.ConsumeProducersTopic(
		ctx,
		wg,
		adminCmd.plat.StreamProvider(),
		adminCmd.plat.Args.Platform,
		adminCmd.plat.Args.Environment,
		adminCmd.plat.Args.AdminBrokers,
		prodCh,
		nil,
	)

	cullInterval := adminCmd.plat.Args.AdminPingInterval * 10
	cullTicker := time.NewTicker(cullInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-cullTicker.C:
			// cull stale producers
			adminCmd.producerTracker.cull(cullInterval)
		case platMsg := <-platCh:
			if (platMsg.Directive & rkcypb.Directive_PLATFORM) != rkcypb.Directive_PLATFORM {
				log.Error().Msgf("Invalid directive for PlatformTopic: %s", platMsg.Directive.String())
				continue
			}

			adminCmd.plat.RtPlatformDef = platMsg.NewRtPlatDef

			jsonBytes, _ := protojson.Marshal(proto.Message(adminCmd.plat.RtPlatformDef.PlatformDef))
			log.Info().
				Str("PlatformJson", string(jsonBytes)).
				Msg("Platform Replaced")

			platDiff := adminCmd.plat.RtPlatformDef.Diff(platMsg.OldRtPlatDef, adminCmd.plat.StreamProvider().Type(), adminCmd.plat.Args.AdminBrokers, adminCmd.otelcolEndpoint)
			rkcy.UpdateTopics(ctx, adminCmd.plat.StreamProvider(), adminCmd.plat.RtPlatformDef.PlatformDef)
			updateRunner(ctx, adminCmd.plat, adminProdCh, consumersTopic, platDiff)
		case prodMsg := <-prodCh:
			if (prodMsg.Directive & rkcypb.Directive_PRODUCER_STATUS) != rkcypb.Directive_PRODUCER_STATUS {
				log.Error().Msgf("Invalid directive for ProducerTopic: %s", prodMsg.Directive.String())
				continue
			}

			adminCmd.producerTracker.update(prodMsg.ProducerDirective, prodMsg.Timestamp, adminCmd.plat.Args.AdminPingInterval*2)
		}
	}
}

func updateRunner(
	ctx context.Context,
	plat *platform.Platform,
	adminProdCh rkcy.ProducerCh,
	consumersTopic string,
	platDiff *rkcy.PlatformDiff,
) {
	ctx, span := telem.StartFunc(ctx)
	defer span.End()
	traceParent := telem.ExtractTraceParent(ctx)

	for _, p := range platDiff.ProgsToStop {
		msg, err := rkcy.NewKafkaMessage(
			&consumersTopic,
			0,
			&rkcypb.ConsumerDirective{Program: p},
			rkcypb.Directive_CONSUMER_STOP,
			traceParent,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failure in kafkaMessage during updateRunner")
			return
		}

		adminProdCh <- msg
	}

	for _, p := range platDiff.ProgsToStart {
		msg, err := rkcy.NewKafkaMessage(
			&consumersTopic,
			0,
			&rkcypb.ConsumerDirective{Program: p},
			rkcypb.Directive_CONSUMER_START,
			traceParent,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failure in kafkaMessage during updateRunner")
			return
		}

		adminProdCh <- msg
	}
}
