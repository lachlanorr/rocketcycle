// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	otel_codes "go.opentelemetry.io/otel/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type ConfigMgr struct {
	config            *Config
	lastChanged       time.Time
	lastChangedOffset int64

	mtx sync.Mutex
}

func NewConfigMgr(
	ctx context.Context,
	adminBrokers string,
	platformName string,
	wg *sync.WaitGroup,
) *ConfigMgr {
	confMgr := &ConfigMgr{
		config: newEmptyConfig(),
	}

	wg.Add(1)
	go confMgr.manageConfigTopic(
		ctx,
		adminBrokers,
		platformName,
		wg,
	)

	return confMgr
}

func newEmptyConfig() *Config {
	return &Config{
		StringVals:  make(map[string]string),
		BoolVals:    make(map[string]bool),
		Float64Vals: make(map[string]float64),
		DecimalVals: make(map[string]*Decimal),
		ComplexVals: make(map[string]*Config_Complex),
	}
}

func (confMgr *ConfigMgr) GetString(key string) (string, bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config != nil && confMgr.config.StringVals != nil {
		val, ok := confMgr.config.StringVals[key]
		return val, ok
	}
	return "", false
}

func (confMgr *ConfigMgr) SetString(key string, val string) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config == nil {
		confMgr.config = newEmptyConfig()
	}
	confMgr.config.StringVals[key] = val
}

func (confMgr *ConfigMgr) GetBool(key string) (bool, bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config != nil && confMgr.config.BoolVals != nil {
		val, ok := confMgr.config.BoolVals[key]
		return val, ok
	}
	return false, false
}

func (confMgr *ConfigMgr) SetBool(key string, val bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config == nil {
		confMgr.config = newEmptyConfig()
	}
	confMgr.config.BoolVals[key] = val
}

func (confMgr *ConfigMgr) GetFloat64(key string) (float64, bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config != nil && confMgr.config.Float64Vals != nil {
		val, ok := confMgr.config.Float64Vals[key]
		return val, ok
	}
	return 0.0, false
}

func (confMgr *ConfigMgr) SetFloat64(key string, val float64) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config == nil {
		confMgr.config = newEmptyConfig()
	}
	confMgr.config.Float64Vals[key] = val
}

func (confMgr *ConfigMgr) GetDecimal(key string) (Decimal, bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config != nil && confMgr.config.DecimalVals != nil {
		val, ok := confMgr.config.DecimalVals[key]
		return *val, ok
	}
	return Decimal{}, false
}

func (confMgr *ConfigMgr) SetDecimal(key string, val Decimal) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config == nil {
		confMgr.config = newEmptyConfig()
	}
	confMgr.config.DecimalVals[key] = &val
}

func (confMgr *ConfigMgr) GetComplex(msgType string, key string) ([]byte, bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config != nil && confMgr.config.ComplexVals != nil {
		complex, ok := confMgr.config.ComplexVals[msgType]
		if !ok || complex == nil || complex.MessageVals == nil {
			return nil, false
		}
		msgBytes, ok := complex.MessageVals[key]
		return msgBytes, ok
	}
	return nil, false
}

func (confMgr *ConfigMgr) SetComplex(msgType string, key string, val []byte) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config == nil {
		confMgr.config = newEmptyConfig()
	}
	complex, ok := confMgr.config.ComplexVals[msgType]
	if !ok {
		complex = &Config_Complex{
			MessageVals: make(map[string][]byte),
		}
		confMgr.config.ComplexVals[msgType] = complex
	}
	complex.MessageVals[key] = val
}

func (confMgr *ConfigMgr) manageConfigTopic(
	ctx context.Context,
	adminBrokers string,
	platformName string,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	configTopic := ConfigTopic(platformName)
	groupName := uncommittedGroupName(configTopic, 0)

	slog := log.With().
		Str("Topic", configTopic).
		Logger()

	found, lastPlatformOff, err := FindMostRecentMatching(
		adminBrokers,
		configTopic,
		0,
		Directive_CONFIG_PUBLISH,
		kAtLastMatch,
	)
	if err != nil {
		slog.Warn().
			Msg("Failed to FindMostRecentOffset, defaulting to empty config")
	}
	if !found {
		slog.Warn().
			Msg("No matching found with FindMostRecentOffset, defaulting to empty config")
	}

	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        adminBrokers,
		"group.id":                 groupName,
		"enable.auto.commit":       false, // we never commit this topic, always want every consumer go have latest
		"enable.auto.offset.store": false, // we never commit this topic
	})
	if err != nil {
		slog.Error().
			Err(err).
			Msg("Failed to NewConsumer")
		return
	}
	defer func() {
		log.Warn().
			Str("Topic", configTopic).
			Msgf("Closing kafka consumer")
		cons.Close()
		log.Warn().
			Str("Topic", configTopic).
			Msgf("Closed kafka consumer")
	}()

	err = cons.Assign([]kafka.TopicPartition{
		{
			Topic:     &configTopic,
			Partition: 0,
			Offset:    kafka.Offset(lastPlatformOff),
		},
	})

	if err != nil {
		slog.Error().
			Err(err).
			Msg("Failed to Assign")
		return
	}

	for {
		select {
		case <-ctx.Done():
			log.Warn().
				Msg("consumeConfigTopic exiting, ctx.Done()")
			return
		default:
			msg, err := cons.ReadMessage(100 * time.Millisecond)
			timedOut := err != nil && err.(kafka.Error).Code() == kafka.ErrTimedOut
			if err != nil && !timedOut {
				slog.Error().
					Err(err).
					Msg("Error during ReadMessage")
			} else if !timedOut && msg != nil {
				directive := GetDirective(msg)
				if (directive & Directive_CONFIG_PUBLISH) == Directive_CONFIG_PUBLISH {
					newConf := &Config{}
					err := proto.Unmarshal(msg.Value, newConf)
					if err != nil {
						log.Error().
							Err(err).
							Msg("Failed to Unmarshal Config")
						continue
					}
					confMgr.mtx.Lock()
					confMgr.config = newConf
					confMgr.lastChanged = msg.Timestamp
					confMgr.lastChangedOffset = int64(msg.TopicPartition.Offset)
					confMgr.mtx.Unlock()
				}
			}
		}
	}
}

func (confMgr *ConfigMgr) BuildConfigResponse() *ConfigReadResponse {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()

	confRsp := &ConfigReadResponse{
		Config:            confMgr.config,
		LastChanged:       timestamppb.New(confMgr.lastChanged),
		LastChangedOffset: confMgr.lastChangedOffset,
	}

	var err error
	confRsp.ConfigJson, err = config2json(confMgr.config)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to convert config to json")
		confRsp.ConfigJson = "{\"error\": \"error converting config to json\"}"
	}

	return confRsp
}

func json2config(data []byte) (*Config, error) {
	var confj map[string]interface{}
	err := json.Unmarshal(data, &confj)
	if err != nil {
		return nil, err
	}

	conf := newEmptyConfig()

	for k, i := range confj {
		switch v := i.(type) {
		case string:
			// try to parse decimal abbreviated string
			d, err := DecimalFromString(v)
			if err == nil {
				conf.DecimalVals[k] = &d
			} else {
				conf.StringVals[k] = v
			}
		case bool:
			conf.BoolVals[k] = v
		case float64:
			conf.Float64Vals[k] = v
		case map[string]interface{}:
			lPart, lOk := v["l"]
			rPart, rOk := v["r"]
			if len(v) == 2 && lOk && rOk {
				conf.DecimalVals[k] = &Decimal{L: int64(lPart.(float64)), R: int64(rPart.(float64))}
			} else {
				return nil, fmt.Errorf("Complex parsing not implemented: %+v", i)
			}
		default:
			return nil, fmt.Errorf("Failed to parse config value: %+v", i)
		}
	}
	return conf, nil
}

func config2json(conf *Config) (string, error) {
	jmap := make(map[string]interface{})
	for k, v := range conf.StringVals {
		jmap[k] = v
	}
	for k, v := range conf.BoolVals {
		jmap[k] = v
	}
	for k, v := range conf.Float64Vals {
		jmap[k] = v
	}
	for k, v := range conf.DecimalVals {
		// always output verbose decimals, not the abbreviated ones
		decMap := make(map[string]int64)
		decMap["l"] = v.L
		decMap["r"] = v.R
		jmap[k] = decMap
	}

	jbytes, err := json.Marshal(jmap)
	if err != nil {
		return "", err
	}
	return string(jbytes), nil
}

func cobraConfigReplace(cmd *cobra.Command, args []string) {
	ctx, span := Telem().StartFunc(context.Background())
	defer span.End()

	slog := log.With().
		Str("Brokers", gSettings.AdminBrokers).
		Str("ConfigPath", gSettings.ConfigFilePath).
		Logger()

	// read config file and deserialize
	data, err := ioutil.ReadFile(gSettings.ConfigFilePath)
	if err != nil {
		span.SetStatus(otel_codes.Error, err.Error())
		slog.Fatal().
			Err(err).
			Msg("Failed to ReadFile")
	}

	conf, err := json2config(data)
	if err != nil {
		span.SetStatus(otel_codes.Error, err.Error())
		slog.Fatal().
			Err(err).
			Msg("Failed to parseConfigFile")
	}

	// connect to kafka and make sure we have our platform topics
	err = createPlatformTopics(context.Background(), gSettings.AdminBrokers, PlatformName())
	if err != nil {
		span.SetStatus(otel_codes.Error, err.Error())
		slog.Fatal().
			Err(err).
			Str("Platform", PlatformName()).
			Msg("Failed to create platform topics")
	}

	configTopic := ConfigTopic(PlatformName())
	slog = slog.With().
		Str("Topic", configTopic).
		Logger()

	// At this point we are guaranteed to have a platform admin topic
	prod, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":  gSettings.AdminBrokers,
		"acks":               -1,     // acks required from all in-sync replicas
		"message.timeout.ms": 600000, // 10 minutes
	})
	if err != nil {
		span.SetStatus(otel_codes.Error, err.Error())
		slog.Fatal().
			Err(err).
			Msg("Failed to NewProducer")
	}
	defer func() {
		log.Warn().
			Str("Brokers", gSettings.AdminBrokers).
			Msg("Closing kafka producer")
		prod.Close()
		log.Warn().
			Str("Brokers", gSettings.AdminBrokers).
			Msg("Closed kafka producer")
	}()

	msg, err := kafkaMessage(&configTopic, 0, conf, Directive_CONFIG_PUBLISH, ExtractTraceParent(ctx))
	if err != nil {
		span.SetStatus(otel_codes.Error, err.Error())
		slog.Fatal().
			Err(err).
			Msg("Failed to kafkaMessage")
	}

	produce := func() {
		err := prod.Produce(msg, nil)
		if err != nil {
			span.SetStatus(otel_codes.Error, err.Error())
			slog.Fatal().
				Err(err).
				Msg("Failed to Produce")
		}
	}

	produce()

	// check channel for delivery event
	timer := time.NewTimer(10 * time.Second)
Loop:
	for {
		select {
		case <-timer.C:
			slog.Fatal().
				Msg("Timeout producing config message")
		case ev := <-prod.Events():
			msgEv, ok := ev.(*kafka.Message)
			if !ok {
				slog.Warn().
					Msg("Non *kafka.Message event received from producer")
			} else {
				if msgEv.TopicPartition.Error != nil {
					slog.Warn().
						Err(msgEv.TopicPartition.Error).
						Msg("Error reported while producing config message, trying again after a delay")
					time.Sleep(1 * time.Second)
					produce()
				} else {
					slog.Info().
						Msg("Config successfully produced")
					break Loop
				}
			}
		}
	}
}
