// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package admin

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type ProducerTracker struct {
	platformName  string
	environment   string
	topicProds    map[string]map[string]time.Time
	topicProdsMtx *sync.Mutex
}

func NewProducerTracker(platformName string, environment string) *ProducerTracker {
	pt := &ProducerTracker{
		platformName:  platformName,
		environment:   environment,
		topicProds:    make(map[string]map[string]time.Time),
		topicProdsMtx: &sync.Mutex{},
	}
	return pt
}

func (pt *ProducerTracker) update(pd *rkcypb.ProducerDirective, timestamp time.Time, pingInterval time.Duration) {
	pt.topicProdsMtx.Lock()
	defer pt.topicProdsMtx.Unlock()

	now := time.Now()
	if now.Sub(timestamp) < pingInterval {
		fullTopicName := rkcy.BuildFullTopicName(
			pt.platformName,
			pt.environment,
			pd.ConcernName,
			pd.ConcernType,
			pd.Topic,
			pd.Generation,
		)
		prodMap, prodMapFound := pt.topicProds[fullTopicName]
		if !prodMapFound {
			prodMap = make(map[string]time.Time)
			pt.topicProds[fullTopicName] = prodMap
		}

		_, prodFound := prodMap[pd.Id]
		if !prodFound {
			log.Info().Msgf("New producer %s:%s", fullTopicName, pd.Id)
		}
		pt.topicProds[fullTopicName][pd.Id] = timestamp
	}
}

func (pt *ProducerTracker) cull(ageLimit time.Duration) {
	pt.topicProdsMtx.Lock()
	defer pt.topicProdsMtx.Unlock()

	now := time.Now()
	for topic, prodMap := range pt.topicProds {
		for id, timestamp := range prodMap {
			age := now.Sub(timestamp)
			if age >= ageLimit {
				log.Info().Msgf("Culling producer %s:%s, age %s", topic, id, age)
				delete(prodMap, id)
			}
		}
		if len(pt.topicProds[topic]) == 0 {
			delete(pt.topicProds, topic)
		}
	}
}

func (pt *ProducerTracker) toTrackedProducers() *rkcypb.TrackedProducers {
	pt.topicProdsMtx.Lock()
	defer pt.topicProdsMtx.Unlock()

	tp := &rkcypb.TrackedProducers{}
	now := time.Now()

	for topic, prodMap := range pt.topicProds {
		for id, timestamp := range prodMap {
			age := now.Sub(timestamp)
			tp.TopicProducers = append(tp.TopicProducers, &rkcypb.TrackedProducers_ProducerInfo{
				Topic:           topic,
				Id:              id,
				TimeSinceUpdate: age.String(),
			})
		}
	}

	return tp
}
