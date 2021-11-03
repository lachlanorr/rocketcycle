// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy/jsonutils"
	"github.com/lachlanorr/rocketcycle/version"
)

type watchTopic struct {
	clusterName    string
	brokers        string
	topicName      string
	partitionCount int32
	logLevel       zerolog.Level
}

func (wt *watchTopic) String() string {
	return fmt.Sprintf("%s__%s", wt.clusterName, wt.topicName)
}

func (wt *watchTopic) consume(ctx context.Context) {
	groupName := uncommittedGroupNameAllPartitions(wt.topicName)
	log.Info().Msgf("watching: %s", wt.topicName)

	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        wt.brokers,
		"group.id":                 groupName,
		"enable.auto.commit":       false,
		"enable.auto.offset.store": false,
	})
	if err != nil {
		log.Error().
			Err(err).
			Str("BoostrapServers", wt.brokers).
			Str("GroupId", groupName).
			Msg("Unable to kafka.NewConsumer")
		return
	}
	defer cons.Close()

	topicParts := make([]kafka.TopicPartition, wt.partitionCount)
	for i := int32(0); i < wt.partitionCount; i++ {
		topicParts[i] = kafka.TopicPartition{
			Topic:     &wt.topicName,
			Partition: i,
			Offset:    kafka.OffsetEnd,
		}
	}
	cons.Assign(topicParts)

	pjOpts := protojson.MarshalOptions{Multiline: true, Indent: "  ", EmitUnpopulated: true}

	for {
		select {
		case <-ctx.Done():
			log.Warn().
				Msg("watchTopic.consume exiting, ctx.Done()")
			return
		default:
			msg, err := cons.ReadMessage(100 * time.Millisecond)
			timedOut := err != nil && err.(kafka.Error).Code() == kafka.ErrTimedOut
			if err != nil && !timedOut {
				log.Error().
					Err(err).
					Msg("Error during ReadMessage")
			} else if !timedOut && msg != nil {
				log.WithLevel(wt.logLevel).
					Str("Directive", fmt.Sprintf("0x%08X", int(GetDirective(msg)))).
					Str("TxnId", GetTraceId(msg)).
					Int("Offset", int(msg.TopicPartition.Offset)).
					Msg(wt.topicName)
				txn := &ApecsTxn{}
				err := proto.Unmarshal(msg.Value, txn)
				if err == nil {
					if gSettings.WatchDecode {
						txnJsonDec, err := DecodeTxnOpaques(ctx, txn, &pjOpts)
						if err == nil {
							log.WithLevel(wt.logLevel).
								Msg(string(txnJsonDec))
						} else {
							log.Error().
								Err(err).
								Msgf("Failed to decodeOpaques: %s", err.Error())
						}
					} else {
						txnJson := pjOpts.Format(txn)
						log.WithLevel(wt.logLevel).
							Msg(string(txnJson))
					}
				}
			}
		}
	}
}

func getAllWatchTopics(rtPlat *rtPlatform) []*watchTopic {
	var wts []*watchTopic
	for _, concern := range rtPlat.Concerns {
		for _, topic := range concern.Topics {
			tp, err := ParseFullTopicName(topic.CurrentTopic)
			if err == nil {
				if tp.Topic == ERROR || tp.Topic == COMPLETE {
					wt := watchTopic{
						clusterName:    topic.CurrentCluster.Name,
						brokers:        topic.CurrentCluster.Brokers,
						topicName:      topic.CurrentTopic,
						partitionCount: topic.CurrentTopicPartitionCount,
					}
					if tp.Topic == ERROR {
						wt.logLevel = zerolog.ErrorLevel
					} else {
						wt.logLevel = zerolog.DebugLevel
					}
					wts = append(wts, &wt)
				}
			}
		}
	}
	return wts
}

func DecodeTxnOpaques(ctx context.Context, txn *ApecsTxn, pjOpts *protojson.MarshalOptions) ([]byte, error) {
	txnJson, err := pjOpts.Marshal(txn)
	if err != nil {
		return nil, err
	}

	var txnTopLvl *jsonutils.OrderedMap
	err = jsonutils.UnmarshalOrdered(txnJson, &txnTopLvl)
	if err != nil {
		return nil, err
	}

	var decSteps = func(iSteps interface{}) {
		steps, ok := iSteps.([]interface{})
		if ok {
			for _, iStep := range steps {
				step, ok := iStep.(*jsonutils.OrderedMap)
				if ok {
					iConcern, ok := step.Get("concern")
					if !ok {
						continue
					}
					concern, ok := iConcern.(string)
					if !ok {
						continue
					}

					iSystem, ok := step.Get("system")
					if !ok {
						continue
					}
					systemStr, ok := iSystem.(string)
					if !ok {
						continue
					}
					systemInt32, ok := System_value[systemStr]
					if !ok {
						continue
					}
					system := System(systemInt32)

					iCommand, ok := step.Get("command")
					if !ok {
						continue
					}
					command, ok := iCommand.(string)
					if !ok {
						continue
					}

					iB64, ok := step.Get("payload")
					if ok {
						b64, ok := iB64.(string)
						if ok && b64 != "" {
							argJson, err := decodeArgPayload64Json(ctx, concern, system, command, b64)
							var argOmap *jsonutils.OrderedMap
							err = jsonutils.UnmarshalOrdered(argJson, &argOmap)
							if err == nil {
								step.SetAfter("payloadDec", argOmap, "payload")
							} else {
								step.SetAfter("payloadDecErr", err.Error(), "payload")
							}
						}
					}

					iRslt, ok := step.Get("result")
					if ok {
						rslt, ok := iRslt.(*jsonutils.OrderedMap)
						if ok {
							iB64, ok := rslt.Get("payload")
							if ok {
								b64, ok := iB64.(string)
								if ok && b64 != "" {
									resJson, err := decodeResultPayload64Json(ctx, concern, system, command, b64)
									var resOmap *jsonutils.OrderedMap
									err = jsonutils.UnmarshalOrdered(resJson, &resOmap)
									if err == nil {
										rslt.SetAfter("payloadDec", resOmap, "payload")
									} else {
										rslt.SetAfter("payloadDecErr", err.Error(), "payload")
									}
								}
							}

							iB64, ok = rslt.Get("instance")
							if ok {
								b64, ok := iB64.(string)
								if ok && b64 != "" {
									instJson, err := decodeInstance64Json(ctx, concern, b64)
									var instOmap *jsonutils.OrderedMap
									err = jsonutils.UnmarshalOrdered(instJson, &instOmap)
									if err == nil {
										rslt.SetAfter("instanceDec", instOmap, "instance")
									} else {
										rslt.SetAfter("instanceDecErr", err.Error(), "instance")
									}
								}
							}
						}
					}
				}
			}
		}
	}

	iFwSteps, ok := txnTopLvl.Get("forwardSteps")
	if ok {
		decSteps(iFwSteps)
	}
	iRevSteps, ok := txnTopLvl.Get("reverseSteps")
	if ok {
		decSteps(iRevSteps)
	}

	jsonSer, err := jsonutils.MarshalOrderedIndent(txnTopLvl, "", "  ")
	if err != nil {
		return nil, err
	}

	return jsonSer, nil
}

func watchResultTopics(ctx context.Context, adminBrokers string, wg *sync.WaitGroup) {
	defer wg.Done()

	wtMap := make(map[string]bool)

	platformCh := make(chan *PlatformMessage)
	consumePlatformTopic(
		ctx,
		platformCh,
		adminBrokers,
		PlatformName(),
		Environment(),
		wg,
	)

	for {
		select {
		case <-ctx.Done():
			log.Warn().
				Msg("watchResultTopics exiting, ctx.Done()")
			return
		case platMsg := <-platformCh:
			if (platMsg.Directive & Directive_PLATFORM) != Directive_PLATFORM {
				log.Error().Msgf("Invalid directive for PlatformTopic: %s", platMsg.Directive.String())
				continue
			}

			wts := getAllWatchTopics(platMsg.NewRtPlat)

			for _, wt := range wts {
				_, ok := wtMap[wt.String()]
				if !ok {
					wtMap[wt.String()] = true
					go wt.consume(ctx)
				}
			}
		}
	}
}

func cobraWatch(cmd *cobra.Command, args []string) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("APECS WATCH started")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	defer func() {
		signal.Stop(interruptCh)
		cancel()
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go watchResultTopics(ctx, gSettings.AdminBrokers, &wg)

	select {
	case <-interruptCh:
		log.Info().
			Msg("APECS WATCH stopped")
		cancel()
		wg.Wait()
		return
	}
}
