// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"encoding/base64"
	"fmt"
	"runtime/debug"

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

	REFRESH = "Refresh"
)

var gReservedCommandNames map[string]bool

func IsReservedCommandName(s string) bool {
	if gReservedCommandNames == nil {
		gReservedCommandNames = make(map[string]bool)
		gReservedCommandNames[CREATE] = true
		gReservedCommandNames[READ] = true
		gReservedCommandNames[UPDATE] = true
		gReservedCommandNames[DELETE] = true

		gReservedCommandNames[VALIDATE_CREATE] = true
		gReservedCommandNames[VALIDATE_UPDATE] = true

		gReservedCommandNames[REFRESH] = true
	}
	return gReservedCommandNames[s]
}

var gConcernHandlers map[string]ConcernHandler = make(map[string]ConcernHandler)

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
	) *ApecsTxn_Step_Result
	DecodeInstance(ctx context.Context, buffer []byte) (proto.Message, error)
	DecodeArg(ctx context.Context, system System, command string, buffer []byte) (proto.Message, error)
	DecodeResult(ctx context.Context, system System, command string, buffer []byte) (proto.Message, error)

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

func decodeInstance(ctx context.Context, concern string, instance []byte) (proto.Message, error) {
	concernHandler, ok := gConcernHandlers[concern]
	if !ok {
		return nil, fmt.Errorf("decodeInstance invalid concern: %s", concern)
	}
	return concernHandler.DecodeInstance(ctx, instance)
}

func decodeInstance64(ctx context.Context, concern string, instance64 string) (proto.Message, error) {
	instance, err := base64.StdEncoding.DecodeString(instance64)
	if err != nil {
		return nil, err
	}
	return decodeInstance(ctx, concern, instance)
}

func decodeInstanceJson(ctx context.Context, concern string, instance []byte) (string, error) {
	msg, err := decodeInstance(ctx, concern, instance)
	if err != nil {
		return "", err
	}
	msgJson, err := protojson.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(msgJson), nil
}

func decodeInstance64Json(ctx context.Context, concern string, instance64 string) (string, error) {
	instance, err := base64.StdEncoding.DecodeString(instance64)
	if err != nil {
		return "", err
	}
	return decodeInstanceJson(ctx, concern, instance)
}

func decodeArgPayload(ctx context.Context, concern string, system System, command string, payload []byte) (proto.Message, error) {
	concernHandler, ok := gConcernHandlers[concern]
	if !ok {
		return nil, fmt.Errorf("decodeArgPayload invalid concern: %s", concern)
	}
	return concernHandler.DecodeArg(ctx, system, command, payload)
}

func decodeArgPayload64(ctx context.Context, concern string, system System, command string, payload64 string) (proto.Message, error) {
	payload, err := base64.StdEncoding.DecodeString(payload64)
	if err != nil {
		return nil, err
	}
	return decodeArgPayload(ctx, concern, system, command, payload)
}

func decodeArgPayloadJson(ctx context.Context, concern string, system System, command string, payload []byte) (string, error) {
	msg, err := decodeArgPayload(ctx, concern, system, command, payload)
	if err != nil {
		return "", err
	}
	msgJson, err := protojson.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(msgJson), nil
}

func decodeArgPayload64Json(ctx context.Context, concern string, system System, command string, payload64 string) (string, error) {
	payload, err := base64.StdEncoding.DecodeString(payload64)
	if err != nil {
		return "", err
	}
	return decodeArgPayloadJson(ctx, concern, system, command, payload)
}

func decodeResultPayload(ctx context.Context, concern string, system System, command string, payload []byte) (proto.Message, error) {
	concernHandler, ok := gConcernHandlers[concern]
	if !ok {
		return nil, fmt.Errorf("decodeResultPayload invalid concern: %s", concern)
	}
	return concernHandler.DecodeResult(ctx, system, command, payload)
}

func decodeResultPayload64(ctx context.Context, concern string, system System, command string, payload64 string) (proto.Message, error) {
	payload, err := base64.StdEncoding.DecodeString(payload64)
	if err != nil {
		return nil, err
	}
	return decodeResultPayload(ctx, concern, system, command, payload)
}

func decodeResultPayloadJson(ctx context.Context, concern string, system System, command string, payload []byte) (string, error) {
	msg, err := decodeResultPayload(ctx, concern, system, command, payload)
	if err != nil {
		return "", err
	}
	msgJson, err := protojson.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(msgJson), nil
}

func decodeResultPayload64Json(ctx context.Context, concern string, system System, command string, payload64 string) (string, error) {
	payload, err := base64.StdEncoding.DecodeString(payload64)
	if err != nil {
		return "", err
	}
	return decodeResultPayloadJson(ctx, concern, system, command, payload)
}

func handleCommand(
	ctx context.Context,
	concern string,
	system System,
	command string,
	direction Direction,
	args *StepArgs,
	confRdr *ConfigRdr,
) *ApecsTxn_Step_Result {
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Str("TraceId", args.TraceId).
				Str("Concern", concern).
				Str("System", system.String()).
				Str("Command", command).
				Str("Direction", direction.String()).
				Msgf("panic during handleCommand, %s, args: %+v", r, args)
			debug.PrintStack()
		}
	}()

	if system == System_PROCESS {
		switch command {
		case CREATE:
			fallthrough
		case UPDATE:
			if direction == Direction_REVERSE {
				panic("REVERSE NOT IMPLEMENTED")
			}
			return &ApecsTxn_Step_Result{
				Code:    Code_OK,
				Payload: args.Payload,
			}
		case READ:
			if direction == Direction_REVERSE {
				panic("REVERSE NOT IMPLEMENTED")
			}
			return &ApecsTxn_Step_Result{
				Code:    Code_OK,
				Payload: args.Instance,
			}
		case DELETE:
			if direction == Direction_REVERSE {
				panic("REVERSE NOT IMPLEMENTED")
			}
			gInstanceStore.Remove(args.Key)
			return &ApecsTxn_Step_Result{
				Code: Code_OK,
			}
		}
	}

	concernHandler, ok := gConcernHandlers[concern]
	if !ok {
		rslt := &ApecsTxn_Step_Result{}
		rslt.SetResult(fmt.Errorf("No handler for concern: '%s'", concern))
		return rslt
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
	TraceId       string
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
