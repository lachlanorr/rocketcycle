// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
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

func (conf *Config) getDecimal(key string) (Decimal, bool) {
	if conf.DecimalVals != nil {
		val, ok := conf.DecimalVals[key]
		return *val, ok
	}
	return Decimal{}, false
}

func (conf *Config) setDecimal(key string, val Decimal) {
	if conf.DecimalVals == nil {
		conf.DecimalVals = make(map[string]*Decimal)
	}
	conf.DecimalVals[key] = &val
}

func (conf *Config) getComplex(msgType string, key string) ([]byte, bool) {
	if conf.ComplexVals != nil {
		confCmplx, ok := conf.ComplexVals[msgType]
		if ok && confCmplx.MessageVals != nil {
			val, ok := confCmplx.MessageVals[key]
			return val, ok
		}
	}
	return nil, false
}

func (conf *Config) setComplex(msgType string, key string, val []byte) {
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

func (confMgr *ConfigMgr) GetDecimal(key string) (Decimal, bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config != nil {
		return confMgr.config.getDecimal(key)
	}
	return Decimal{}, false
}

func (confMgr *ConfigMgr) SetDecimal(key string, val Decimal) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config == nil {
		confMgr.config = newEmptyConfig()
	}
	confMgr.config.setDecimal(key, val)
}

func (confMgr *ConfigMgr) GetComplex(msgType string, key string) ([]byte, bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config != nil {
		return confMgr.config.getComplex(msgType, key)
	}
	return nil, false
}

func (confMgr *ConfigMgr) SetComplex(msgType string, key string, val []byte) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config == nil {
		confMgr.config = newEmptyConfig()
	}
	confMgr.config.setComplex(msgType, key, val)
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

func translateFromAbbreviatedDecimalsList(list []interface{}) {
	for _, i := range list {
		switch v := i.(type) {
		case []interface{}:
			translateFromAbbreviatedDecimalsList(v)
		case map[string]interface{}:
			translateFromAbbreviatedDecimalsObj(v)
		}
	}
}

func translateFromAbbreviatedDecimalsObj(obj map[string]interface{}) {
	for k, i := range obj {
		switch v := i.(type) {
		case string:
			d, err := DecimalFromString(v)
			if err == nil {
				dmap := make(map[string]interface{})
				dmap["l"] = d.L
				dmap["r"] = d.R
				obj[k] = dmap
			}
		case []interface{}:
			translateFromAbbreviatedDecimalsList(v)
		case map[string]interface{}:
			translateFromAbbreviatedDecimalsObj(v)
		}
	}
}

func translateToAbbreviatedDecimalsList(list []interface{}) {
	for _, i := range list {
		switch v := i.(type) {
		case []interface{}:
			translateToAbbreviatedDecimalsList(v)
		case map[string]interface{}:
			translateToAbbreviatedDecimalsObj(v)
		}
	}
}

func translateToAbbreviatedDecimalsObj(obj map[string]interface{}) {
	for k, i := range obj {
		switch v := i.(type) {
		case map[string]interface{}:
			d, err := decimalFromMap(v)
			if err == nil {
				obj[k] = d.ToString()
			} else {
				translateToAbbreviatedDecimalsObj(v)
			}
		case []interface{}:
			translateToAbbreviatedDecimalsList(v)
		}
	}
}

func decimalDigitFromInterface(i interface{}) (int64, error) {
	vf, ok := i.(float64)
	if ok {
		i64 := int64(vf)
		if vf-float64(i64) != 0 {
			return 0, fmt.Errorf("float64 digit with fractional part: %f", vf)
		}
		return i64, nil
	}
	vs, ok := i.(string)
	if ok {
		return strconv.ParseInt(vs, 10, 64)
	}
	return 0, fmt.Errorf("invalid interface{} for digit: %+v, type: %T", i, i)
}

func decimalFromMap(obj map[string]interface{}) (Decimal, error) {
	lPart, lOk := obj["l"]
	rPart, rOk := obj["r"]
	if len(obj) <= 2 && (lOk || rOk) {
		var lI64, rI64 int64
		var err error
		if lOk {
			lI64, err = decimalDigitFromInterface(lPart)
			if err != nil {
				return Decimal{}, err
			}
		}
		if rOk {
			rI64, err = decimalDigitFromInterface(rPart)
			if err != nil {
				return Decimal{}, err
			}
		}
		return Decimal{L: lI64, R: rI64}, nil
	} else {
		return Decimal{}, fmt.Errorf("Error parsing Decimal: %+v", obj)
	}
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
				conf.setDecimal(k, d)
			} else {
				conf.setString(k, v)
			}
		case bool:
			conf.setBool(k, v)
		case float64:
			conf.setFloat64(k, v)
		case map[string]interface{}:
			d, err := decimalFromMap(v)
			if err != nil {
				return nil, err
			}
			conf.setDecimal(k, d)
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
				idI, ok := itmMap["id"]
				if !ok {
					return nil, fmt.Errorf("No 'id' provided in complex config, %s idx %d", k, idx)
				}
				id, ok := idI.(string)
				if !ok {
					return nil, fmt.Errorf("'id' not a string in complex config, %s idx %d", k, idx)
				}
				translateFromAbbreviatedDecimalsObj(itmMap)
				itmJsonBytes, err := json.Marshal(itmMap)
				if err != nil {
					return nil, fmt.Errorf("Error re-marshalling complex config to json, %s idx %d, err: '%s'", k, idx, err.Error())
				}
				itmMsg, err := ComplexConfigUnmarshalJson(msgType, itmJsonBytes)
				if err != nil {
					return nil, fmt.Errorf("Unable to unmarshal complex config json, %s idx %d, err: '%s'", k, idx, err.Error())
				}
				itmBytes, err := proto.Marshal(itmMsg)
				if err != nil {
					return nil, fmt.Errorf("Unable to marshal complex config msg, %s idx %d, err: '%s'", k, idx, err.Error())
				}
				conf.setComplex(msgType, id, itmBytes)
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

	translateToAbbreviatedDecimalsObj(jmap)
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
