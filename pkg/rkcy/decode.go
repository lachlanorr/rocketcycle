// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/lachlanorr/rocketcycle/pkg/jsonutils"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

func (concernHandlers ConcernHandlers) DecodeInstance(ctx context.Context, concern string, buffer []byte) (*ResultProto, error) {
	concernHandler, ok := concernHandlers[concern]
	if !ok {
		return nil, fmt.Errorf("decodeInstance invalid concern: %s", concern)
	}
	return concernHandler.DecodeInstance(ctx, buffer)
}

func (concernHandlers ConcernHandlers) DecodeInstance64(ctx context.Context, concern string, buffer64 string) (*ResultProto, error) {
	buffer, err := base64.StdEncoding.DecodeString(buffer64)
	if err != nil {
		return nil, err
	}
	return concernHandlers.DecodeInstance(ctx, concern, buffer)
}

func (concernHandlers ConcernHandlers) DecodeInstanceJson(ctx context.Context, concern string, buffer []byte) ([]byte, error) {
	resProto, err := concernHandlers.DecodeInstance(ctx, concern, buffer)
	if err != nil {
		return nil, err
	}

	jsonMap := make(map[string]interface{})
	jsonMap["type"] = resProto.Type

	pjOpts := protojson.MarshalOptions{EmitUnpopulated: true}

	instJson, err := pjOpts.Marshal(resProto.Instance)
	if err != nil {
		return nil, err
	}
	instMap := make(map[string]interface{})
	err = json.Unmarshal(instJson, &instMap)
	if err != nil {
		return nil, err
	}
	jsonMap["instance"] = instMap

	if resProto.Related != nil {
		relJson, err := pjOpts.Marshal(resProto.Related)
		if err != nil {
			return nil, err
		}
		relMap := make(map[string]interface{})
		err = json.Unmarshal(relJson, &relMap)
		if err != nil {
			return nil, err
		}
		jsonMap["related"] = relMap
	}

	jsonBytes, err := json.Marshal(jsonMap)
	if err != nil {
		return nil, err
	}

	return jsonBytes, nil
}

func (concernHandlers ConcernHandlers) DecodeInstance64Json(ctx context.Context, concern string, buffer64 string) ([]byte, error) {
	buffer, err := base64.StdEncoding.DecodeString(buffer64)
	if err != nil {
		return nil, err
	}
	return concernHandlers.DecodeInstanceJson(ctx, concern, buffer)
}

func (concernHandlers ConcernHandlers) DecodeArgPayload(
	ctx context.Context,
	concern string,
	system rkcypb.System,
	command string,
	buffer []byte,
) (*ResultProto, ConcernHandler, error) {
	cncHdlr, ok := concernHandlers[concern]
	if !ok {
		return nil, nil, fmt.Errorf("DecodeArgPayload invalid concern: %s", concern)
	}
	resProto, err := cncHdlr.DecodeArg(ctx, system, command, buffer)
	return resProto, cncHdlr, err
}

func (concernHandlers ConcernHandlers) DecodeArgPayload64(
	ctx context.Context,
	concern string,
	system rkcypb.System,
	command string,
	buffer64 string,
) (*ResultProto, ConcernHandler, error) {
	buffer, err := base64.StdEncoding.DecodeString(buffer64)
	if err != nil {
		return nil, nil, err
	}
	return concernHandlers.DecodeArgPayload(ctx, concern, system, command, buffer)
}

func (concernHandlers ConcernHandlers) resultProto2OrderedMap(
	ctx context.Context,
	cncHdlr ConcernHandler,
	resProto *ResultProto,
) (*jsonutils.OrderedMap, error) {
	pjOpts := protojson.MarshalOptions{EmitUnpopulated: true}

	var instOmap *jsonutils.OrderedMap
	if resProto.Instance != nil {
		instJson, err := pjOpts.Marshal(resProto.Instance)
		if err != nil {
			return nil, err
		}
		err = jsonutils.UnmarshalOrdered(instJson, &instOmap)
		if err != nil {
			return nil, err
		}

		switch resProto.Type {
		case "RelatedRequest":
			relReqDec, err := cncHdlr.DecodeRelatedRequest(ctx, resProto.Instance.(*rkcypb.RelatedRequest))
			if err != nil {
				return nil, err
			}
			relReqDecOmap, err := concernHandlers.resultProto2OrderedMap(ctx, cncHdlr, relReqDec)
			if err != nil {
				return nil, err
			}
			instOmap.SetAfter("payloadDec", relReqDecOmap, "payload")
		case "RelatedResponse":
			relRsp := resProto.Instance.(*rkcypb.RelatedResponse)
			relRspInst, err := concernHandlers.DecodeInstance(ctx, relRsp.Concern, relRsp.Payload)
			if err != nil {
				return nil, err
			}
			relRspInstOmap, err := concernHandlers.resultProto2OrderedMap(ctx, cncHdlr, relRspInst)
			if err != nil {
				return nil, err
			}
			instOmap.SetAfter("payloadDec", relRspInstOmap, "payload")
		}
	}

	var relOmap *jsonutils.OrderedMap
	if resProto.Related != nil {
		relJson, err := protojson.Marshal(resProto.Related)
		if err != nil {
			return nil, err
		}
		err = jsonutils.UnmarshalOrdered(relJson, &relOmap)
		if err != nil {
			return nil, err
		}
	}

	resOmap := jsonutils.NewOrderedMap()
	resOmap.Set("type", resProto.Type)
	if instOmap != nil {
		resOmap.Set("instance", instOmap)
	}
	if relOmap != nil {
		resOmap.Set("related", relOmap)
	}
	return resOmap, nil
}

func (concernHandlers ConcernHandlers) resultProto2Json(ctx context.Context, cncHdlr ConcernHandler, resProto *ResultProto) ([]byte, error) {
	resOmap, err := concernHandlers.resultProto2OrderedMap(ctx, cncHdlr, resProto)
	if err != nil {
		return nil, err
	}
	return jsonutils.MarshalOrdered(resOmap)
}

func (concernHandlers ConcernHandlers) DecodeArgPayloadJson(
	ctx context.Context,
	concern string,
	system rkcypb.System,
	command string,
	buffer []byte,
) ([]byte, error) {
	resProto, cncHdlr, err := concernHandlers.DecodeArgPayload(ctx, concern, system, command, buffer)
	if err != nil {
		return nil, err
	}
	return concernHandlers.resultProto2Json(ctx, cncHdlr, resProto)
}

func (concernHandlers ConcernHandlers) DecodeArgPayload64Json(
	ctx context.Context,
	concern string,
	system rkcypb.System,
	command string,
	buffer64 string,
) ([]byte, error) {
	buffer, err := base64.StdEncoding.DecodeString(buffer64)
	if err != nil {
		return nil, err
	}
	return concernHandlers.DecodeArgPayloadJson(ctx, concern, system, command, buffer)
}

func (concernHandlers ConcernHandlers) DecodeResultPayload(
	ctx context.Context,
	concern string,
	system rkcypb.System,
	command string,
	buffer []byte,
) (*ResultProto, ConcernHandler, error) {
	cncHdlr, ok := concernHandlers[concern]
	if !ok {
		return nil, nil, fmt.Errorf("DecodeResultPayload invalid concern: %s", concern)
	}
	resProto, err := cncHdlr.DecodeResult(ctx, system, command, buffer)
	return resProto, cncHdlr, err
}

func (concernHandlers ConcernHandlers) DecodeResultPayload64(
	ctx context.Context,
	concern string,
	system rkcypb.System,
	command string,
	buffer64 string,
) (*ResultProto, ConcernHandler, error) {
	buffer, err := base64.StdEncoding.DecodeString(buffer64)
	if err != nil {
		return nil, nil, err
	}
	return concernHandlers.DecodeResultPayload(ctx, concern, system, command, buffer)
}

func (concernHandlers ConcernHandlers) DecodeResultPayloadJson(
	ctx context.Context,
	concern string,
	system rkcypb.System,
	command string,
	buffer []byte,
) ([]byte, error) {
	resProto, cncHdlr, err := concernHandlers.DecodeResultPayload(ctx, concern, system, command, buffer)
	if err != nil {
		return nil, err
	}
	return concernHandlers.resultProto2Json(ctx, cncHdlr, resProto)
}

func (concernHandlers ConcernHandlers) DecodeResultPayload64Json(
	ctx context.Context,
	concern string,
	system rkcypb.System,
	command string,
	buffer64 string,
) ([]byte, error) {
	buffer, err := base64.StdEncoding.DecodeString(buffer64)
	if err != nil {
		return nil, err
	}
	return concernHandlers.DecodeResultPayloadJson(ctx, concern, system, command, buffer)
}
