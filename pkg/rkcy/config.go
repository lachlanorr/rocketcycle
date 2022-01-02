// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type Config struct {
	rkcypb.Config
}

func NewConfig() *Config {
	conf := &Config{}
	conf.StringVals = make(map[string]string)
	conf.BoolVals = make(map[string]bool)
	conf.Float64Vals = make(map[string]float64)
	conf.ComplexVals = make(map[string]*rkcypb.Config_Complex)
	return conf
}

func (conf *Config) GetString(key string) (string, bool) {
	if conf.StringVals != nil {
		val, ok := conf.StringVals[key]
		return val, ok
	}
	return "", false
}

func (conf *Config) SetString(key string, val string) {
	if conf.StringVals == nil {
		conf.StringVals = make(map[string]string)
	}
	conf.StringVals[key] = val
}

func (conf *Config) GetBool(key string) (bool, bool) {
	if conf.BoolVals != nil {
		val, ok := conf.BoolVals[key]
		return val, ok
	}
	return false, false
}

func (conf *Config) SetBool(key string, val bool) {
	if conf.BoolVals == nil {
		conf.BoolVals = make(map[string]bool)
	}
	conf.BoolVals[key] = val
}

func (conf *Config) GetFloat64(key string) (float64, bool) {
	if conf.Float64Vals != nil {
		val, ok := conf.Float64Vals[key]
		return val, ok
	}
	return 0.0, false
}

func (conf *Config) SetFloat64(key string, val float64) {
	if conf.Float64Vals == nil {
		conf.Float64Vals = make(map[string]float64)
	}
	conf.Float64Vals[key] = val
}

func (conf *Config) GetComplexMsg(msgType string, key string) (proto.Message, bool) {
	msgBytes, ok := conf.GetComplexBytes(msgType, key)
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

func (conf *Config) SetComplexMsg(msgType string, key string, msg proto.Message) {
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		log.Error().
			Err(err).
			Str("Type", msgType).
			Msg("Unable to marshal complex config")
		return
	}
	conf.SetComplexBytes(msgType, key, msgBytes)
}

func (conf *Config) GetComplexBytes(msgType string, key string) ([]byte, bool) {
	if conf.ComplexVals != nil {
		confCmplx, ok := conf.ComplexVals[msgType]
		if ok && confCmplx.MessageVals != nil {
			val, ok := confCmplx.MessageVals[key]
			return val, ok
		}
	}
	return nil, false
}

func (conf *Config) SetComplexBytes(msgType string, key string, val []byte) {
	if conf.ComplexVals == nil {
		conf.ComplexVals = make(map[string]*rkcypb.Config_Complex)
	}
	confCmplx, ok := conf.ComplexVals[msgType]
	if !ok {
		confCmplx = &rkcypb.Config_Complex{
			MessageVals: make(map[string][]byte),
		}
		conf.ComplexVals[msgType] = confCmplx
	}
	confCmplx.MessageVals[key] = val
}

func JsonToConfig(data []byte) (*Config, error) {
	var confj map[string]interface{}
	err := json.Unmarshal(data, &confj)
	if err != nil {
		return nil, err
	}

	conf := NewConfig()

	for k, i := range confj {
		switch v := i.(type) {
		case string:
			conf.SetString(k, v)
		case bool:
			conf.SetBool(k, v)
		case float64:
			conf.SetFloat64(k, v)
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
				conf.SetComplexMsg(msgType, key, msg)
			}
		default:
			return nil, fmt.Errorf("Failed to parse config value: %+v", i)
		}
	}
	return conf, nil
}

func ConfigToJson(conf *Config) (string, error) {
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
