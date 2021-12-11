// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package admin

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/consumer"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/version"
)

type AdminCmd struct {
	plat            rkcy.Platform
	producerTracker *ProducerTracker
}

func Start(plat rkcy.Platform) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("admin started")

	adminCmd := &AdminCmd{
		plat:            plat,
		producerTracker: NewProducerTracker(plat.Name(), plat.Environment()),
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	defer func() {
		signal.Stop(interruptCh)
		cancel()
	}()

	var wg sync.WaitGroup
	go managePlatform(ctx, plat, adminCmd, &wg)

	select {
	case <-interruptCh:
		log.Warn().
			Msg("admin stopped")
		cancel()
		wg.Wait()
		return
	}
}

func managePlatform(
	ctx context.Context,
	plat rkcy.Platform,
	adminCmd *AdminCmd,
	wg *sync.WaitGroup,
) {
	consumersTopic := rkcy.ConsumersTopic(adminCmd.plat.Name(), adminCmd.plat.Environment())
	adminProdCh := adminCmd.plat.GetProducerCh(ctx, adminCmd.plat.AdminBrokers(), wg)

	platCh := make(chan *rkcy.PlatformMessage)
	consumer.ConsumePlatformTopic(
		ctx,
		plat,
		platCh,
		nil,
		wg,
	)

	prodCh := make(chan *consumer.ProducerMessage)
	consumer.ConsumeProducersTopic(
		ctx,
		plat,
		prodCh,
		nil,
		wg,
	)

	cullInterval := adminCmd.plat.AdminPingInterval() * 10
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

			adminCmd.plat.SetPlatformDef(platMsg.NewRtPlatDef)

			jsonBytes, _ := protojson.Marshal(proto.Message(adminCmd.plat.PlatformDef().PlatformDef))
			log.Info().
				Str("PlatformJson", string(jsonBytes)).
				Msg("Platform Replaced")

			platDiff := adminCmd.plat.PlatformDef().Diff(platMsg.OldRtPlatDef, adminCmd.plat.AdminBrokers(), adminCmd.plat.Telem().OtelcolEndpoint)
			updateTopics(adminCmd.plat.PlatformDef())
			updateRunner(ctx, adminCmd.plat, adminProdCh, consumersTopic, platDiff)
		case prodMsg := <-prodCh:
			if (prodMsg.Directive & rkcypb.Directive_PRODUCER_STATUS) != rkcypb.Directive_PRODUCER_STATUS {
				log.Error().Msgf("Invalid directive for ProducerTopic: %s", prodMsg.Directive.String())
				continue
			}

			adminCmd.producerTracker.update(prodMsg.ProducerDirective, prodMsg.Timestamp, adminCmd.plat.AdminPingInterval()*2)
		}
	}
}

func updateTopics(rtPlatDef *rkcy.RtPlatformDef) {
	// start admin connections to all clusters
	clusterInfos := make(map[string]*clusterInfo)
	for _, cluster := range rtPlatDef.PlatformDef.Clusters {
		ci, err := newClusterInfo(cluster)

		if err != nil {
			log.Printf("Unable to connect to cluster '%s', brokers '%s': %s", cluster.Name, cluster.Brokers, err.Error())
			return
		}

		clusterInfos[cluster.Name] = ci
		defer ci.Close()
		log.Info().
			Str("Cluster", cluster.Name).
			Str("Brokers", cluster.Brokers).
			Msg("Connected to cluster")
	}

	var concernTypesAutoCreate = []string{"GENERAL", "APECS"}

	for _, concern := range rtPlatDef.PlatformDef.Concerns {
		if rkcy.Contains(concernTypesAutoCreate, rkcypb.Concern_Type_name[int32(concern.Type)]) {
			for _, topics := range concern.Topics {
				createMissingTopics(
					rkcy.BuildTopicNamePrefix(rtPlatDef.PlatformDef.Name, rtPlatDef.PlatformDef.Environment, concern.Name, concern.Type),
					topics,
					clusterInfos)
			}
		}
	}
}

func updateRunner(
	ctx context.Context,
	plat rkcy.Platform,
	adminProdCh rkcy.ProducerCh,
	consumersTopic string,
	platDiff *rkcy.PlatformDiff,
) {
	ctx, span := plat.Telem().StartFunc(ctx)
	defer span.End()
	traceParent := rkcy.ExtractTraceParent(ctx)

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
