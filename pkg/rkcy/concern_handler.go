// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type StepArgs struct {
	TxnId         string
	Key           string
	Instance      []byte
	Payload       []byte
	EffectiveTime time.Time

	CmpdOffset    *rkcypb.CompoundOffset
	ForwardResult *rkcypb.ApecsTxn_Step_Result
}

type ResultProto struct {
	Type     string
	Instance proto.Message
	Related  proto.Message
}

type StorageInit func(ctx context.Context, config map[string]string, wg *sync.WaitGroup) error
type StorageTargetInit struct {
	*rkcypb.StorageTarget
	Init StorageInit
}

type ConcernHandlers map[string]ConcernHandler

type ConcernHandler interface {
	ConcernName() string
	HandleLogicCommand(
		ctx context.Context,
		system rkcypb.System,
		command string,
		direction rkcypb.Direction,
		args *StepArgs,
		instanceStore *InstanceStore,
		confRdr ConfigRdr,
	) (*rkcypb.ApecsTxn_Step_Result, []*rkcypb.ApecsTxn_Step)
	HandleCrudCommand(
		ctx context.Context,
		system rkcypb.System,
		command string,
		direction rkcypb.Direction,
		args *StepArgs,
		storageType string,
		wg *sync.WaitGroup,
	) (*rkcypb.ApecsTxn_Step_Result, []*rkcypb.ApecsTxn_Step)
	DecodeInstance(ctx context.Context, buffer []byte) (*ResultProto, error)
	DecodeArg(ctx context.Context, system rkcypb.System, command string, buffer []byte) (*ResultProto, error)
	DecodeResult(ctx context.Context, system rkcypb.System, command string, buffer []byte) (*ResultProto, error)
	DecodeRelatedRequest(ctx context.Context, relReq *rkcypb.RelatedRequest) (*ResultProto, error)

	SetLogicHandler(commands interface{}) error
	SetCrudHandler(storageType string, commands interface{}) error
	ValidateHandlers() bool
	SetStorageTargets(storageTargets map[string]*StorageTargetInit)
}

func IsStorageSystem(system rkcypb.System) bool {
	return system == rkcypb.System_STORAGE || system == rkcypb.System_STORAGE_SCND
}

func (concernHandlers ConcernHandlers) ValidateConcernHandlers() bool {
	retval := true
	for concern, cncHdlr := range concernHandlers {
		if !cncHdlr.ValidateHandlers() {
			log.Error().
				Str("Concern", concern).
				Msgf("Invalid commands for ConcernHandler")
			retval = false
		}
	}
	return retval
}

func (concernHandlers ConcernHandlers) RegisterLogicHandler(concern string, handler interface{}) {
	cncHdlr, ok := concernHandlers[concern]
	if !ok {
		panic(fmt.Sprintf("%s concern handler not registered", concern))
	}
	err := cncHdlr.SetLogicHandler(handler)
	if err != nil {
		panic(err.Error())
	}
}

func (concernHandlers ConcernHandlers) RegisterCrudHandler(storageType string, concern string, handler interface{}) {
	cncHdlr, ok := concernHandlers[concern]
	if !ok {
		panic(fmt.Sprintf("%s concern handler not registered", concern))
	}
	err := cncHdlr.SetCrudHandler(storageType, handler)
	if err != nil {
		panic(err.Error())
	}
}
