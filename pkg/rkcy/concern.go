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

var gConcernHandlers map[string]ConcernHandler = make(map[string]ConcernHandler)

type CommandHandler func(context.Context, System, string, Direction, *StepArgs) *ApecsTxn_Step_Result
type InstanceDecoder func(context.Context, []byte) (string, error)
type PayloadDecoder func(context.Context, System, string, []byte) (string, error)

type ConcernHandler struct {
	Handler         CommandHandler
	InstanceDecoder InstanceDecoder
	ArgDecoder      PayloadDecoder
	ResultDecoder   PayloadDecoder
}

func RegisterConcernHandler(
	concern string,
	handler CommandHandler,
	instanceDecoder InstanceDecoder,
	argDecoder PayloadDecoder,
	resultDecoder PayloadDecoder,
) {
	_, ok := gConcernHandlers[concern]
	if ok {
		panic(fmt.Sprintf("%s concern handler already registered", concern))
	}
	gConcernHandlers[concern] = ConcernHandler{
		Handler:         handler,
		InstanceDecoder: instanceDecoder,
		ArgDecoder:      argDecoder,
		ResultDecoder:   resultDecoder,
	}
}

func decodeInstance(ctx context.Context, concern string, instance []byte) (string, error) {
	concernHandler, ok := gConcernHandlers[concern]
	if !ok {
		return "", fmt.Errorf("decodeInstance invalid concern: %s", concern)
	}
	return concernHandler.InstanceDecoder(ctx, instance)
}

func decodeInstance64(ctx context.Context, concern string, instance64 string) (string, error) {
	instance, err := base64.StdEncoding.DecodeString(instance64)
	if err != nil {
		return "", err
	}
	return decodeInstance(ctx, concern, instance)
}

func decodeArgPayload(ctx context.Context, concern string, system System, command string, payload []byte) (string, error) {
	concernHandler, ok := gConcernHandlers[concern]
	if !ok {
		return "", fmt.Errorf("decodeArgPayload invalid concern: %s", concern)
	}
	return concernHandler.ArgDecoder(ctx, system, command, payload)
}

func decodeArgPayload64(ctx context.Context, concern string, system System, command string, payload64 string) (string, error) {
	payload, err := base64.StdEncoding.DecodeString(payload64)
	if err != nil {
		return "", err
	}
	return decodeArgPayload(ctx, concern, system, command, payload)
}

func decodeResultPayload(ctx context.Context, concern string, system System, command string, payload []byte) (string, error) {
	concernHandler, ok := gConcernHandlers[concern]
	if !ok {
		return "", fmt.Errorf("decodeResultPayload invalid concern: %s", concern)
	}
	return concernHandler.ResultDecoder(ctx, system, command, payload)
}

func decodeResultPayload64(ctx context.Context, concern string, system System, command string, payload64 string) (string, error) {
	payload, err := base64.StdEncoding.DecodeString(payload64)
	if err != nil {
		return "", err
	}
	return decodeResultPayload(ctx, concern, system, command, payload)
}

func handleCommand(
	ctx context.Context,
	concern string,
	system System,
	command string,
	direction Direction,
	args *StepArgs,
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
				Code:     Code_OK,
				Payload:  args.Payload,
				Instance: args.Payload,
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
			gInstanceCache.Remove(args.Key)
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

	return concernHandler.Handler(ctx, system, command, direction, args)
}

type StepArgs struct {
	TraceId       string
	ProcessedTime *timestamp.Timestamp
	Key           string
	Instance      []byte
	Payload       []byte
	Offset        *Offset
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
