// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/golang/protobuf/ptypes/timestamp"
)

const (
	CREATE = "Create"
	READ   = "Read"
	UPDATE = "Update"
	DELETE = "Delete"

	VALIDATE_CREATE = "ValidateCreate"
	VALIDATE_UPDATE = "ValidateUpdate"

	REFRESH_INSTANCE = "RefreshInstance"
	FLUSH_INSTANCE   = "FlushInstance"

	UPDATE_RELATED  = "UpdateRelated"
	REFRESH_RELATED = "RefreshRelated"
)

var gReservedCommandNames map[string]bool
var gTxnProhibitedCommandNames map[string]bool

func IsReservedCommandName(s string) bool {
	if gReservedCommandNames == nil {
		gReservedCommandNames = make(map[string]bool)
		gReservedCommandNames[CREATE] = true
		gReservedCommandNames[READ] = true
		gReservedCommandNames[UPDATE] = true
		gReservedCommandNames[DELETE] = true

		gReservedCommandNames[VALIDATE_CREATE] = true
		gReservedCommandNames[VALIDATE_UPDATE] = true

		gReservedCommandNames[REFRESH_INSTANCE] = true
		gReservedCommandNames[FLUSH_INSTANCE] = true

		gReservedCommandNames[UPDATE_RELATED] = true
		gReservedCommandNames[REFRESH_RELATED] = true
	}
	if gReservedCommandNames[s] {
		return true
	}
	if IsRelatedCommand(s) {
		return true
	}
	return false
}

func IsTxnProhibitedCommandName(s string) bool {
	if gTxnProhibitedCommandNames == nil {
		gTxnProhibitedCommandNames = make(map[string]bool)

		gTxnProhibitedCommandNames[VALIDATE_CREATE] = true
		gTxnProhibitedCommandNames[VALIDATE_UPDATE] = true

		gTxnProhibitedCommandNames[REFRESH_INSTANCE] = true
		gTxnProhibitedCommandNames[FLUSH_INSTANCE] = true

		gTxnProhibitedCommandNames[UPDATE_RELATED] = true
		gTxnProhibitedCommandNames[REFRESH_RELATED] = true
	}
	if gTxnProhibitedCommandNames[s] {
		return true
	}
	if IsRelatedCommand(s) {
		return true
	}
	return false
}

var gConcernHandlers map[string]ConcernHandler = make(map[string]ConcernHandler)

func IsRelatedCommand(command string) bool {
	return strings.HasPrefix(command, UPDATE_RELATED) ||
		strings.HasPrefix(command, REFRESH_RELATED)
}

func BuildUpdateRelatedCommand(srcConcern string, tgtField string) string {
	return fmt.Sprintf("%s_%s_%s", UPDATE_RELATED, srcConcern, tgtField)
}

func BuildRefreshRelatedCommand(srcConcern string, tgtField string) string {
	return fmt.Sprintf("%s_%s_%s", REFRESH_RELATED, srcConcern, tgtField)
}

func ParseRelatedCommand(command string) (string, string, string, error) {
	if !IsRelatedCommand(command) {
		return "", "", "", fmt.Errorf("Related command with wrong prefix: %s", command)
	}
	parts := strings.Split(command, "_")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("Wrong number of parts of related command: %s", command)
	}
	return parts[0], parts[1], parts[2], nil
}

type ConcernHandler interface {
	ConcernName() string
	HandleCommand(
		ctx context.Context,
		system System,
		command string,
		direction Direction,
		args *StepArgs,
		instanceStore *InstanceStore,
		confRdr *ConfigRdr,
		storageSystem string,
	) (*ApecsTxn_Step_Result, []*ApecsTxn_Step)
	DecodeInstance(ctx context.Context, buffer []byte) (proto.Message, error)
	DecodeRelated(ctx context.Context, buffer []byte) (proto.Message, error)
	DecodeArg(ctx context.Context, system System, command string, buffer []byte) (*ResultProto, error)
	DecodeResult(ctx context.Context, system System, command string, buffer []byte) (*ResultProto, error)

	SetProcessCommands(commands interface{}) error
	SetStorageCommands(storageSystem string, commands interface{}) error
	ValidateCommands() bool
}

func validateConcernHandlers() bool {
	retval := true
	for concern, cncHdlr := range gConcernHandlers {
		if !cncHdlr.ValidateCommands() {
			log.Error().
				Str("Concern", concern).
				Msgf("Invalid commands for ConcernHandler")
			retval = false
		}
	}
	return retval
}

func RegisterProcessCommands(concern string, commands interface{}) {
	cncHdlr, ok := gConcernHandlers[concern]
	if !ok {
		panic(fmt.Sprintf("%s concern handler not registered registered", concern))
	}
	err := cncHdlr.SetProcessCommands(commands)
	if err != nil {
		panic(err.Error())
	}
}

func RegisterStorageCommands(storageSystem string, concern string, commands interface{}) {
	cncHdlr, ok := gConcernHandlers[concern]
	if !ok {
		panic(fmt.Sprintf("%s concern handler not registered registered", concern))
	}
	err := cncHdlr.SetStorageCommands(storageSystem, commands)
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

func decodeInstance(ctx context.Context, concern string, buffer []byte) (proto.Message, error) {
	concernHandler, ok := gConcernHandlers[concern]
	if !ok {
		return nil, fmt.Errorf("decodeInstance invalid concern: %s", concern)
	}
	return concernHandler.DecodeInstance(ctx, buffer)
}

func decodeInstance64(ctx context.Context, concern string, buffer64 string) (proto.Message, error) {
	buffer, err := base64.StdEncoding.DecodeString(buffer64)
	if err != nil {
		return nil, err
	}
	return decodeInstance(ctx, concern, buffer)
}

func decodeInstanceJson(ctx context.Context, concern string, buffer []byte) ([]byte, error) {
	instMsg, err := decodeInstance(ctx, concern, buffer)
	if err != nil {
		return nil, err
	}
	instJson, err := protojson.Marshal(instMsg)
	if err != nil {
		return nil, err
	}
	return instJson, nil
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
) (*ResultProto, error) {
	concernHandler, ok := gConcernHandlers[concern]
	if !ok {
		return nil, fmt.Errorf("decodeArgPayload invalid concern: %s", concern)
	}
	return concernHandler.DecodeArg(ctx, system, command, buffer)
}

func decodeArgPayload64(
	ctx context.Context,
	concern string,
	system System,
	command string,
	buffer64 string,
) (*ResultProto, error) {
	buffer, err := base64.StdEncoding.DecodeString(buffer64)
	if err != nil {
		return nil, err
	}
	return decodeArgPayload(ctx, concern, system, command, buffer)
}

func newResultJson(resProto *ResultProto) (*ResultJson, error) {
	instanceJson, err := protojson.Marshal(resProto.Instance)
	if err != nil {
		return nil, err
	}
	var relatedJson []byte
	if resProto.Related != nil {
		relatedJson, err = protojson.Marshal(resProto.Related)
		if err != nil {
			return nil, err
		}
	}
	return &ResultJson{
		Type:     resProto.Type,
		Instance: instanceJson,
		Related:  relatedJson,
	}, nil
}

func decodeArgPayloadJson(
	ctx context.Context,
	concern string,
	system System,
	command string,
	buffer []byte,
) (*ResultJson, error) {
	resProto, err := decodeArgPayload(ctx, concern, system, command, buffer)
	if err != nil {
		return nil, err
	}
	return newResultJson(resProto)
}

func decodeArgPayload64Json(
	ctx context.Context,
	concern string,
	system System,
	command string,
	buffer64 string,
) (*ResultJson, error) {
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
) (*ResultProto, error) {
	concernHandler, ok := gConcernHandlers[concern]
	if !ok {
		return nil, fmt.Errorf("decodeResultPayload invalid concern: %s", concern)
	}
	return concernHandler.DecodeResult(ctx, system, command, buffer)
}

func decodeResultPayload64(
	ctx context.Context,
	concern string,
	system System,
	command string,
	buffer64 string,
) (*ResultProto, error) {
	buffer, err := base64.StdEncoding.DecodeString(buffer64)
	if err != nil {
		return nil, err
	}
	return decodeResultPayload(ctx, concern, system, command, buffer)
}

func decodeResultPayloadJson(
	ctx context.Context,
	concern string,
	system System,
	command string,
	buffer []byte,
) (*ResultJson, error) {
	resProto, err := decodeResultPayload(ctx, concern, system, command, buffer)
	if err != nil {
		return nil, err
	}
	return newResultJson(resProto)
}

func decodeResultPayload64Json(
	ctx context.Context,
	concern string,
	system System,
	command string,
	buffer64 string,
) (*ResultJson, error) {
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

	concernHandler, ok := gConcernHandlers[concern]
	if !ok {
		rslt := &ApecsTxn_Step_Result{}
		rslt.SetResult(fmt.Errorf("No handler for concern: '%s'", concern))
		return rslt, nil
	}

	return concernHandler.HandleCommand(
		ctx,
		system,
		command,
		direction,
		args,
		gInstanceStore,
		confRdr,
		"postgresql",
	)
}

type StepArgs struct {
	TxnId         string
	ProcessedTime *timestamp.Timestamp
	Key           string
	Instance      []byte
	Payload       []byte
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

func (e *Error) Error() string {
	return fmt.Sprintf("%d: %s", Code_name[int32(e.Code)], e.Msg)
}

func NewError(code Code, msg string) *Error {
	return &Error{Code: code, Msg: msg}
}

type ConcernInstance interface {
	Type() string
	Key() string
	SetKey(key string)
}

func PackPayloads(payload0 []byte, payload1 []byte) []byte {
	packed := make([]byte, 8+len(payload0)+len(payload1))
	copy(packed, "rkcy")
	offset := 8 + len(payload0)
	binary.LittleEndian.PutUint32(packed[4:8], uint32(offset))
	copy(packed[8:], payload0)
	copy(packed[offset:], payload1)
	return packed
}

func UnpackPayloads(packed []byte) ([][]byte, error) {
	if packed == nil || len(packed) < 8 {
		return nil, fmt.Errorf("Not a packed payload, too small: %s", base64.StdEncoding.EncodeToString(packed))
	}
	if string(packed[:4]) != "rkcy" {
		return nil, fmt.Errorf("Not a packed payload, missing rkcy: %s", base64.StdEncoding.EncodeToString(packed))
	}
	offset := binary.LittleEndian.Uint32(packed[4:8])
	if offset < uint32(8) || offset > uint32(len(packed)) {
		return nil, fmt.Errorf("Not a packed payload, invalid offset %d: %s", offset, base64.StdEncoding.EncodeToString(packed))
	}
	ret := make([][]byte, 2)
	ret[0] = packed[8:offset]
	ret[1] = packed[offset:]

	// Return nils if there is no data in slices
	if len(ret[0]) == 0 {
		ret[0] = nil
	}
	if len(ret[1]) == 0 {
		ret[1] = nil
	}

	return ret, nil
}
