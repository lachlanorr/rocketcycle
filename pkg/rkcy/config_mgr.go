// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	otel_codes "go.opentelemetry.io/otel/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type ComplexConfigHandler interface {
	GetKey(proto.Message) string
	Unmarshal(b []byte) (proto.Message, error)
	UnmarshalJson(b []byte) (proto.Message, error)
}

var gComplexConfigHandlers map[string]ComplexConfigHandler = make(map[string]ComplexConfigHandler)
var gComplexConfigHandlersMtx sync.Mutex

func RegisterComplexConfigHandler(msgType string, handler ComplexConfigHandler) {
	gComplexConfigHandlersMtx.Lock()
	defer gComplexConfigHandlersMtx.Unlock()

	if gComplexConfigHandlers[msgType] != nil {
		log.Fatal().
			Str("Type", msgType).
			Msg("Multiple ComplexConfigHandlers registered for type")
	}
	gComplexConfigHandlers[msgType] = handler
}

func getComplexConfigHandler(msgType string) (ComplexConfigHandler, bool) {
	gComplexConfigHandlersMtx.Lock()
	defer gComplexConfigHandlersMtx.Unlock()

	handler, ok := gComplexConfigHandlers[msgType]
	return handler, ok
}

func ComplexConfigUnmarshal(msgType string, b []byte) (proto.Message, error) {
	handler, ok := getComplexConfigHandler(msgType)
	if !ok {
		return nil, fmt.Errorf("No CommandHandler for type: %s", msgType)
	}
	return handler.Unmarshal(b)
}

func ComplexConfigUnmarshalJson(msgType string, b []byte) (proto.Message, error) {
	handler, ok := getComplexConfigHandler(msgType)
	if !ok {
		return nil, fmt.Errorf("No CommandHandler for type: %s", msgType)
	}
	return handler.UnmarshalJson(b)
}

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
	environment string,
	wg *sync.WaitGroup,
) *ConfigMgr {
	confMgr := &ConfigMgr{
		config: newEmptyConfig(),
	}

	go confMgr.manageConfigTopic(
		ctx,
		adminBrokers,
		platformName,
		environment,
		wg,
	)

	return confMgr
}

func newEmptyConfig() *Config {
	return &Config{
		StringVals:  make(map[string]string),
		BoolVals:    make(map[string]bool),
		Float64Vals: make(map[string]float64),
		ComplexVals: make(map[string]*Config_Complex),
	}
}

func (conf *Config) getString(key string) (string, bool) {
	if conf.StringVals != nil {
		val, ok := conf.StringVals[key]
		return val, ok
	}
	return "", false
}

func (conf *Config) setString(key string, val string) {
	if conf.StringVals == nil {
		conf.StringVals = make(map[string]string)
	}
	conf.StringVals[key] = val
}

func (conf *Config) getBool(key string) (bool, bool) {
	if conf.BoolVals != nil {
		val, ok := conf.BoolVals[key]
		return val, ok
	}
	return false, false
}

func (conf *Config) setBool(key string, val bool) {
	if conf.BoolVals == nil {
		conf.BoolVals = make(map[string]bool)
	}
	conf.BoolVals[key] = val
}

func (conf *Config) getFloat64(key string) (float64, bool) {
	if conf.Float64Vals != nil {
		val, ok := conf.Float64Vals[key]
		return val, ok
	}
	return 0.0, false
}

func (conf *Config) setFloat64(key string, val float64) {
	if conf.Float64Vals == nil {
		conf.Float64Vals = make(map[string]float64)
	}
	conf.Float64Vals[key] = val
}

func (conf *Config) getComplexMsg(msgType string, key string) (proto.Message, bool) {
	msgBytes, ok := conf.getComplexBytes(msgType, key)
	if ok {
		msg, err := ComplexConfigUnmarshal(msgType, msgBytes)
		if err != nil {
			log.Error().
				Err(err).
				Str("Type", msgType).
				Msg("Unable to ComplexConfigUnmarshal")
			return nil, false
		}
		return msg, ok
	}
	return nil, false
}

func (conf *Config) setComplexMsg(msgType string, key string, msg proto.Message) {
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		log.Error().
			Err(err).
			Str("Type", msgType).
			Msg("Unable to marshal complex config")
		return
	}
	conf.setComplexBytes(msgType, key, msgBytes)
}

func (conf *Config) getComplexBytes(msgType string, key string) ([]byte, bool) {
	if conf.ComplexVals != nil {
		confCmplx, ok := conf.ComplexVals[msgType]
		if ok && confCmplx.MessageVals != nil {
			val, ok := confCmplx.MessageVals[key]
			return val, ok
		}
	}
	return nil, false
}

func (conf *Config) setComplexBytes(msgType string, key string, val []byte) {
	if conf.ComplexVals == nil {
		conf.ComplexVals = make(map[string]*Config_Complex)
	}
	confCmplx, ok := conf.ComplexVals[msgType]
	if !ok {
		confCmplx = &Config_Complex{
			MessageVals: make(map[string][]byte),
		}
		conf.ComplexVals[msgType] = confCmplx
	}
	confCmplx.MessageVals[key] = val
}

func (confMgr *ConfigMgr) GetString(key string) (string, bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config != nil {
		return confMgr.config.getString(key)
	}
	return "", false
}

func (confMgr *ConfigMgr) SetString(key string, val string) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config == nil {
		confMgr.config = newEmptyConfig()
	}
	confMgr.config.setString(key, val)
}

func (confMgr *ConfigMgr) GetBool(key string) (bool, bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config != nil {
		return confMgr.config.getBool(key)
	}
	return false, false
}

func (confMgr *ConfigMgr) SetBool(key string, val bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config == nil {
		confMgr.config = newEmptyConfig()
	}
	confMgr.config.setBool(key, val)
}

func (confMgr *ConfigMgr) GetFloat64(key string) (float64, bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config != nil {
		return confMgr.config.getFloat64(key)
	}
	return 0.0, false
}

func (confMgr *ConfigMgr) SetFloat64(key string, val float64) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config == nil {
		confMgr.config = newEmptyConfig()
	}
	confMgr.config.setFloat64(key, val)
}

func (confMgr *ConfigMgr) GetComplexMsg(msgType string, key string) (proto.Message, bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config != nil {
		return confMgr.config.getComplexMsg(msgType, key)
	}
	return nil, false
}

func (confMgr *ConfigMgr) SetComplexMsg(msgType string, key string, msg proto.Message) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config == nil {
		confMgr.config = newEmptyConfig()
	}
	confMgr.config.setComplexMsg(msgType, key, msg)
}

func (confMgr *ConfigMgr) GetComplexBytes(msgType string, key string) ([]byte, bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config != nil {
		return confMgr.config.getComplexBytes(msgType, key)
	}
	return nil, false
}

func (confMgr *ConfigMgr) SetComplexBytes(msgType string, key string, val []byte) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config == nil {
		confMgr.config = newEmptyConfig()
	}
	confMgr.config.setComplexBytes(msgType, key, val)
}

func (confMgr *ConfigMgr) manageConfigTopic(
	ctx context.Context,
	adminBrokers string,
	platformName string,
	environment string,
	wg *sync.WaitGroup,
) {
	confPubCh := make(chan *ConfigPublishMessage)

	consumeConfigTopic(
		ctx,
		confPubCh,
		adminBrokers,
		platformName,
		environment,
		wg,
	)

	for {
		select {
		case <-ctx.Done():
			return
		case confPubMsg := <-confPubCh:
			confMgr.mtx.Lock()
			confMgr.config = confPubMsg.Config
			confMgr.lastChanged = confPubMsg.Timestamp
			confMgr.lastChangedOffset = confPubMsg.Offset
			confMgr.mtx.Unlock()
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
			conf.setString(k, v)
		case bool:
			conf.setBool(k, v)
		case float64:
			conf.setFloat64(k, v)
		case []interface{}:
			msgType := strings.TrimSuffix(k, "List")
			if msgType == k || len(msgType) == 0 {
				return nil, fmt.Errorf("Complex config list not named with form {TypeName}List: %s", k)
			}
			// captialize first character since our type will be as such
			msgType = strings.ToUpper(string(msgType[0])) + msgType[1:]
			for idx, itm := range v {
				itmMap, ok := itm.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("Complex config not an object, %s idx %d", k, idx)
				}
				itmJsonBytes, err := json.Marshal(itmMap)
				if err != nil {
					return nil, fmt.Errorf("Error re-marshalling complex config to json, %s idx %d, err: '%s'", k, idx, err.Error())
				}
				confHdlr, ok := getComplexConfigHandler(msgType)
				if !ok {
					return nil, fmt.Errorf("Complex config type not registered: %s", msgType)
				}
				msg, err := confHdlr.UnmarshalJson(itmJsonBytes)
				key := confHdlr.GetKey(msg)
				conf.setComplexMsg(msgType, key, msg)
			}
		default:
			return nil, fmt.Errorf("Failed to parse config value: %+v", i)
		}
	}
	return conf, nil
}

func config2json(conf *Config) (string, error) {
	// LORRNOTE: This is only to be called when the conf has
	// exclusive access, caller must lock the config mgr before
	// calling this if it is passing in the config mgr's
	// config

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

	for msgType, cmplxConf := range conf.ComplexVals {
		list := make([]interface{}, 0, len(cmplxConf.MessageVals))
		for _, msgBytes := range cmplxConf.MessageVals {
			msg, err := ComplexConfigUnmarshal(msgType, msgBytes)
			if err != nil {
				return "", err
			}
			pjOpts := protojson.MarshalOptions{EmitUnpopulated: true}
			msgJsonBytes, err := pjOpts.Marshal(msg)
			if err != nil {
				return "", err
			}
			var msgObj map[string]interface{}
			err = json.Unmarshal(msgJsonBytes, &msgObj)
			if err != nil {
				return "", err
			}
			list = append(list, msgObj)
		}
		// captialize first character since our type will be as such
		keyName := strings.ToLower(string(msgType[0])) + msgType[1:] + "List"
		jmap[keyName] = list
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
	err = createPlatformTopics(context.Background(), gSettings.AdminBrokers, PlatformName(), Environment())
	if err != nil {
		span.SetStatus(otel_codes.Error, err.Error())
		slog.Fatal().
			Err(err).
			Str("Platform", PlatformName()).
			Msg("Failed to create platform topics")
	}

	configTopic := ConfigTopic(PlatformName(), Environment())
	slog = slog.With().
		Str("Topic", configTopic).
		Logger()

	// At this point we are guaranteed to have a platform admin topic
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	kafkaLogCh := make(chan kafka.LogEvent)
	go printKafkaLogs(ctx, kafkaLogCh)

	prod, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":  gSettings.AdminBrokers,
		"acks":               -1,     // acks required from all in-sync replicas
		"message.timeout.ms": 600000, // 10 minutes

		"go.logs.channel.enable": true,
		"go.logs.channel":        kafkaLogCh,
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
