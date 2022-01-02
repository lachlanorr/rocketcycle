// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.//
//message Trade {
//option (rkcy.pb.is_concern) = true;
//
//string id = 1 [(rkcy.pb.is_key) = true];
//string username = 2;
//bool active = 3;
//
//Limits limits = 4;
//}

package pb

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

// -----------------------------------------------------------------------------
// Config TradeCategory
// -----------------------------------------------------------------------------
type TradeCategoryConfigHandler struct{}

func (conf *TradeCategory) Key() string {
	return conf.Id
}

func (*TradeCategoryConfigHandler) GetKey(msg proto.Message) string {
	conf := msg.(*TradeCategory)
	return conf.Key()
}

func (*TradeCategoryConfigHandler) Unmarshal(b []byte) (proto.Message, error) {
	conf := &TradeCategory{}
	err := proto.Unmarshal(b, conf)
	if err != nil {
		return nil, err
	}
	return proto.Message(conf), nil
}

func (*TradeCategoryConfigHandler) UnmarshalJson(b []byte) (proto.Message, error) {
	conf := &TradeCategory{}
	err := protojson.Unmarshal(b, conf)
	if err != nil {
		return nil, err
	}
	return proto.Message(conf), nil
}

func init() {
	rkcy.RegisterComplexConfigHandler("TradeCategory", &TradeCategoryConfigHandler{})
}

// -----------------------------------------------------------------------------
// Config TradeCategory END
// -----------------------------------------------------------------------------
