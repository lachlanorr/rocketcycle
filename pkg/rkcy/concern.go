// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy/jsonutils"
)

const (
	CREATE       = "Create"
	READ         = "Read"
	UPDATE       = "Update"
	UPDATE_ASYNC = "UpdateAsync"
	DELETE       = "Delete"

	VALIDATE_CREATE = "ValidateCreate"
	VALIDATE_UPDATE = "ValidateUpdate"

	REFRESH_INSTANCE = "RefreshInstance"
	FLUSH_INSTANCE   = "FlushInstance"

	REQUEST_RELATED = "RequestRelated"
	REFRESH_RELATED = "RefreshRelated"
)

var gReservedCommands map[string]bool
var gTxnProhibitedCommands map[string]bool
var gSecondaryStorageCommands map[string]bool

func IsReservedCommand(cmd string) bool {
	if gReservedCommands == nil {
		gReservedCommands = make(map[string]bool)
		gReservedCommands[CREATE] = true
		gReservedCommands[READ] = true
		gReservedCommands[UPDATE] = true
		gReservedCommands[UPDATE_ASYNC] = true
		gReservedCommands[DELETE] = true

		gReservedCommands[VALIDATE_CREATE] = true
		gReservedCommands[VALIDATE_UPDATE] = true

		gReservedCommands[REFRESH_INSTANCE] = true
		gReservedCommands[FLUSH_INSTANCE] = true

		gReservedCommands[REQUEST_RELATED] = true
		gReservedCommands[REFRESH_RELATED] = true
	}
	if gReservedCommands[cmd] {
		return true
	}
	return false
}

func IsTxnProhibitedCommand(cmd string) bool {
	if gTxnProhibitedCommands == nil {
		gTxnProhibitedCommands = make(map[string]bool)

		gTxnProhibitedCommands[VALIDATE_CREATE] = true
		gTxnProhibitedCommands[VALIDATE_UPDATE] = true

		gTxnProhibitedCommands[REFRESH_INSTANCE] = true
		gTxnProhibitedCommands[FLUSH_INSTANCE] = true

		gTxnProhibitedCommands[REQUEST_RELATED] = true
		gTxnProhibitedCommands[REFRESH_RELATED] = true
	}
	if gTxnProhibitedCommands[cmd] {
		return true
	}
	return false
}

var gConcernHandlers map[string]ConcernHandler = make(map[string]ConcernHandler)

type StorageTarget struct {
	Name      string
	Type      string
	IsPrimary bool
	Config    map[string]string
	Init      StorageInit
}

type ConcernHandler interface {
	ConcernName() string
	HandleLogicCommand(
		ctx context.Context,
		system System,
		command string,
		direction Direction,
		args *StepArgs,
		instanceStore *InstanceStore,
		confRdr *ConfigRdr,
	) (*ApecsTxn_Step_Result, []*ApecsTxn_Step)
	HandleCrudCommand(
		ctx context.Context,
		system System,
		command string,
		direction Direction,
		args *StepArgs,
		instanceStore *InstanceStore,
		storageType string,
		wg *sync.WaitGroup,
	) (*ApecsTxn_Step_Result, []*ApecsTxn_Step)
	DecodeInstance(ctx context.Context, buffer []byte) (*ResultProto, error)
	DecodeArg(ctx context.Context, system System, command string, buffer []byte) (*ResultProto, error)
	DecodeResult(ctx context.Context, system System, command string, buffer []byte) (*ResultProto, error)
	DecodeRelatedRequest(ctx context.Context, relReq *RelatedRequest) (*ResultProto, error)
	DecodeRelatedResponse(ctx context.Context, relRsp *RelatedResponse) (*ResultProto, error)

	SetLogicHandler(commands interface{}) error
	SetCrudHandler(storageType string, commands interface{}) error
	ValidateHandlers() bool
	SetStorageTargets(storageTargets map[string]*StorageTarget)
}

func validateConcernHandlers() bool {
	retval := true
	for concern, cncHdlr := range gConcernHandlers {
		if !cncHdlr.ValidateHandlers() {
			log.Error().
				Str("Concern", concern).
				Msgf("Invalid commands for ConcernHandler")
			retval = false
		}
	}
	return retval
}

func RegisterLogicHandler(concern string, handler interface{}) {
	cncHdlr, ok := gConcernHandlers[concern]
	if !ok {
		panic(fmt.Sprintf("%s concern handler not registered", concern))
	}
	err := cncHdlr.SetLogicHandler(handler)
	if err != nil {
		panic(err.Error())
	}
}

func RegisterCrudHandler(storageType string, concern string, handler interface{}) {
	cncHdlr, ok := gConcernHandlers[concern]
	if !ok {
		panic(fmt.Sprintf("%s concern handler not registered", concern))
	}
	err := cncHdlr.SetCrudHandler(storageType, handler)
	if err != nil {
		panic(err.Error())
	}
}

func RegisterConcernHandler(cncHandler ConcernHandler) {
	_, ok := gConcernHandlers[cncHandler.ConcernName()]
	if ok {
		panic(fmt.Sprintf("%s concern handler already registered", cncHandler.ConcernName()))
	}
	gConcernHandlers[cncHandler.ConcernName()] = cncHandler
}

func decodeInstance(ctx context.Context, concern string, buffer []byte) (*ResultProto, error) {
	concernHandler, ok := gConcernHandlers[concern]
	if !ok {
		return nil, fmt.Errorf("decodeInstance invalid concern: %s", concern)
	}
	return concernHandler.DecodeInstance(ctx, buffer)
}

func decodeInstance64(ctx context.Context, concern string, buffer64 string) (*ResultProto, error) {
	buffer, err := base64.StdEncoding.DecodeString(buffer64)
	if err != nil {
		return nil, err
	}
	return decodeInstance(ctx, concern, buffer)
}

func decodeInstanceJson(ctx context.Context, concern string, buffer []byte) ([]byte, error) {
	resProto, err := decodeInstance(ctx, concern, buffer)
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

func decodeInstance64Json(ctx context.Context, concern string, buffer64 string) ([]byte, error) {
	buffer, err := base64.StdEncoding.DecodeString(buffer64)
	if err != nil {
		return nil, err
	}
	return decodeInstanceJson(ctx, concern, buffer)
}

func decodeArgPayload(
	ctx context.Context,
	concern string,
	system System,
	command string,
	buffer []byte,
) (*ResultProto, ConcernHandler, error) {
	cncHdlr, ok := gConcernHandlers[concern]
	if !ok {
		return nil, nil, fmt.Errorf("decodeArgPayload invalid concern: %s", concern)
	}
	resProto, err := cncHdlr.DecodeArg(ctx, system, command, buffer)
	return resProto, cncHdlr, err
}

func decodeArgPayload64(
	ctx context.Context,
	concern string,
	system System,
	command string,
	buffer64 string,
) (*ResultProto, ConcernHandler, error) {
	buffer, err := base64.StdEncoding.DecodeString(buffer64)
	if err != nil {
		return nil, nil, err
	}
	return decodeArgPayload(ctx, concern, system, command, buffer)
}

func resultProto2OrderedMap(
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
			relReqDec, err := cncHdlr.DecodeRelatedRequest(ctx, resProto.Instance.(*RelatedRequest))
			if err != nil {
				return nil, err
			}
			relReqDecOmap, err := resultProto2OrderedMap(ctx, cncHdlr, relReqDec)
			if err != nil {
				return nil, err
			}
			instOmap.SetAfter("payloadDec", relReqDecOmap, "payload")
		case "RelatedResponse":
			relRspDec, err := cncHdlr.DecodeRelatedResponse(ctx, resProto.Instance.(*RelatedResponse))
			if err != nil {
				return nil, err
			}
			relRspDecOmap, err := resultProto2OrderedMap(ctx, cncHdlr, relRspDec)
			if err != nil {
				return nil, err
			}
			instOmap.SetAfter("payloadDec", relRspDecOmap, "payload")
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

func resultProto2Json(ctx context.Context, cncHdlr ConcernHandler, resProto *ResultProto) ([]byte, error) {
	resOmap, err := resultProto2OrderedMap(ctx, cncHdlr, resProto)
	if err != nil {
		return nil, err
	}
	return jsonutils.MarshalOrdered(resOmap)
}

func decodeArgPayloadJson(
	ctx context.Context,
	concern string,
	system System,
	command string,
	buffer []byte,
) ([]byte, error) {
	resProto, cncHdlr, err := decodeArgPayload(ctx, concern, system, command, buffer)
	if err != nil {
		return nil, err
	}
	return resultProto2Json(ctx, cncHdlr, resProto)
}

func decodeArgPayload64Json(
	ctx context.Context,
	concern string,
	system System,
	command string,
	buffer64 string,
) ([]byte, error) {
	buffer, err := base64.StdEncoding.DecodeString(buffer64)
	if err != nil {
		return nil, err
	}
	return decodeArgPayloadJson(ctx, concern, system, command, buffer)
}

func decodeResultPayload(
	ctx context.Context,
	concern string,
	system System,
	command string,
	buffer []byte,
) (*ResultProto, ConcernHandler, error) {
	cncHdlr, ok := gConcernHandlers[concern]
	if !ok {
		return nil, nil, fmt.Errorf("decodeResultPayload invalid concern: %s", concern)
	}
	resProto, err := cncHdlr.DecodeResult(ctx, system, command, buffer)
	return resProto, cncHdlr, err
}

func decodeResultPayload64(
	ctx context.Context,
	concern string,
	system System,
	command string,
	buffer64 string,
) (*ResultProto, ConcernHandler, error) {
	buffer, err := base64.StdEncoding.DecodeString(buffer64)
	if err != nil {
		return nil, nil, err
	}
	return decodeResultPayload(ctx, concern, system, command, buffer)
}

func decodeResultPayloadJson(
	ctx context.Context,
	concern string,
	system System,
	command string,
	buffer []byte,
) ([]byte, error) {
	resProto, cncHdlr, err := decodeResultPayload(ctx, concern, system, command, buffer)
	if err != nil {
		return nil, err
	}
	return resultProto2Json(ctx, cncHdlr, resProto)
}

func decodeResultPayload64Json(
	ctx context.Context,
	concern string,
	system System,
	command string,
	buffer64 string,
) ([]byte, error) {
	buffer, err := base64.StdEncoding.DecodeString(buffer64)
	if err != nil {
		return nil, err
	}
	return decodeResultPayloadJson(ctx, concern, system, command, buffer)
}

func handleCommand(
	ctx context.Context,
	concern string,
	system System,
	command string,
	direction Direction,
	args *StepArgs,
	confRdr *ConfigRdr,
	wg *sync.WaitGroup,
) (*ApecsTxn_Step_Result, []*ApecsTxn_Step) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Str("TxnId", args.TxnId).
				Str("Concern", concern).
				Str("System", system.String()).
				Str("Command", command).
				Str("Direction", direction.String()).
				Msgf("panic during handleCommand, %s, args: %+v", r, args)
			debug.PrintStack()
		}
	}()

	cncHdlr, ok := gConcernHandlers[concern]
	if !ok {
		rslt := &ApecsTxn_Step_Result{}
		rslt.SetResult(fmt.Errorf("No handler for concern: '%s'", concern))
		return rslt, nil
	}

	switch system {
	case System_PROCESS:
		return cncHdlr.HandleLogicCommand(
			ctx,
			system,
			command,
			direction,
			args,
			gInstanceStore,
			confRdr,
		)
	case System_STORAGE:
		fallthrough
	case System_STORAGE_SCND:
		return cncHdlr.HandleCrudCommand(
			ctx,
			system,
			command,
			direction,
			args,
			gInstanceStore,
			gSettings.StorageTarget,
			wg,
		)
	default:
		rslt := &ApecsTxn_Step_Result{}
		rslt.SetResult(fmt.Errorf("Invalid system: %s", system.String()))
		return rslt, nil
	}
}

type StepArgs struct {
	TxnId         string
	Key           string
	Instance      []byte
	Payload       []byte
	EffectiveTime time.Time

	CmpdOffset    *CompoundOffset
	ForwardResult *ApecsTxn_Step_Result
}

type Error struct {
	Code Code
	Msg  string
}

func (rslt *ApecsTxn_Step_Result) SetResult(err error) {
	if err == nil {
		rslt.Code = Code_OK
	} else {
		rkcyErr, ok := err.(*Error)
		if ok {
			rslt.Code = rkcyErr.Code
			rslt.LogEvents = append(rslt.LogEvents, &LogEvent{Sev: Severity_ERR, Msg: rkcyErr.Msg})
		} else {
			rslt.Code = Code_INTERNAL
			rslt.LogEvents = append(rslt.LogEvents, &LogEvent{Sev: Severity_ERR, Msg: err.Error()})
		}
	}
}

func IsStorageSystem(system System) bool {
	return system == System_STORAGE || system == System_STORAGE_SCND
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", Code_name[int32(e.Code)], e.Msg)
}

func NewError(code Code, msg string) *Error {
	return &Error{Code: code, Msg: msg}
}

type ConcernInstance interface {
	Type() string
	Key() string
	SetKey(key string)
}

func IsPackedPayload(payload []byte) bool {
	return payload != nil && len(payload) > 4 && string(payload[:4]) == "rkcy"
}

func ParsePayload(payload []byte) ([]byte, []byte, error) {
	var (
		instBytes []byte
		relBytes  []byte
		err       error
	)
	if IsPackedPayload(payload) {
		instBytes, relBytes, err = UnpackPayloads(payload)
		if err != nil {
			return nil, nil, err
		}
	} else {
		instBytes = payload
	}
	return instBytes, relBytes, nil
}

func PackPayloads(payload0 []byte, payload1 []byte) []byte {
	if IsPackedPayload(payload0) || IsPackedPayload(payload1) {
		panic(fmt.Sprintf("PackPayloads of already packed payload payload0=%s payload1=%s", base64.StdEncoding.EncodeToString(payload0), base64.StdEncoding.EncodeToString(payload1)))
	}
	packed := make([]byte, 8+len(payload0)+len(payload1))
	copy(packed, "rkcy")
	offset := 8 + len(payload0)
	binary.LittleEndian.PutUint32(packed[4:8], uint32(offset))
	copy(packed[8:], payload0)
	copy(packed[offset:], payload1)
	return packed
}

func UnpackPayloads(packed []byte) ([]byte, []byte, error) {
	if packed == nil || len(packed) < 8 {
		return nil, nil, fmt.Errorf("Not a packed payload, too small: %s", base64.StdEncoding.EncodeToString(packed))
	}
	if string(packed[:4]) != "rkcy" {
		return nil, nil, fmt.Errorf("Not a packed payload, missing rkcy: %s", base64.StdEncoding.EncodeToString(packed))
	}
	offset := binary.LittleEndian.Uint32(packed[4:8])
	if offset < uint32(8) || offset > uint32(len(packed)) {
		return nil, nil, fmt.Errorf("Not a packed payload, invalid offset %d: %s", offset, base64.StdEncoding.EncodeToString(packed))
	}

	instBytes := packed[8:offset]
	relBytes := packed[offset:]

	// Return nils if there is no data in slices
	if len(instBytes) == 0 {
		return nil, nil, fmt.Errorf("No instance contained in packed payload: %s", base64.StdEncoding.EncodeToString(packed))
	}
	if len(relBytes) == 0 {
		relBytes = nil
	}

	return instBytes, relBytes, nil
}
