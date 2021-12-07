// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package pb

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

// -----------------------------------------------------------------------------
// Concern Character
// -----------------------------------------------------------------------------
func init() {
	rkcy.RegisterGlobalConcernHandler(&CharacterConcernHandler{})
}

func (inst *Character) Key() string {
	return inst.Id
}

func MarshalCharacterOrPanic(inst *Character) []byte {
	b, err := proto.Marshal(inst)
	if err != nil {
		panic(err.Error())
	}
	return b
}

func (inst *Character) PreValidateCreate(ctx context.Context) error {
	if inst.Key() != "" {
		return rkcy.NewError(rkcypb.Code_INVALID_ARGUMENT, "Empty Key during PreValidateCreate for Character")
	}
	return nil
}

func (inst *Character) PreValidateUpdate(ctx context.Context, updated *Character) error {
	if updated.Key() == "" || inst.Key() != updated.Key() {
		return rkcy.NewError(rkcypb.Code_INVALID_ARGUMENT, "Mismatched Keys during PreValidateUpdate for Character")
	}
	return nil
}

// LogicHandler Interface
type CharacterLogicHandler interface {
	ValidateCreate(ctx context.Context, inst *Character) (*Character, error)
	ValidateUpdate(ctx context.Context, original *Character, updated *Character) (*Character, error)

	Fund(ctx context.Context, inst *Character, rel *CharacterRelated, payload *FundingRequest) (*Character, error)
	DebitFunds(ctx context.Context, inst *Character, rel *CharacterRelated, payload *FundingRequest) (*Character, error)
	CreditFunds(ctx context.Context, inst *Character, rel *CharacterRelated, payload *FundingRequest) (*Character, error)
}

// CrudHandler Interface
type CharacterCrudHandler interface {
	Read(ctx context.Context, key string) (*Character, *CharacterRelated, *rkcypb.CompoundOffset, error)
	Create(ctx context.Context, inst *Character, cmpdOffset *rkcypb.CompoundOffset) (*Character, error)
	Update(ctx context.Context, inst *Character, rel *CharacterRelated, cmpdOffset *rkcypb.CompoundOffset) error
	Delete(ctx context.Context, key string, cmpdOffset *rkcypb.CompoundOffset) error
}

// Concern Handler
type CharacterConcernHandler struct {
	logicHandler      CharacterLogicHandler
	crudHandlers      map[string]CharacterCrudHandler
	storageTargets    map[string]*rkcy.StorageTargetInit
	storageInitHasRun map[string]bool
}

func (*CharacterConcernHandler) ConcernName() string {
	return "Character"
}

func (cncHdlr *CharacterConcernHandler) SetLogicHandler(handler interface{}) error {
	if cncHdlr.logicHandler != nil {
		return fmt.Errorf("ProcessHandler already registered for CharacterConcernHandler")
	}
	var ok bool
	cncHdlr.logicHandler, ok = handler.(CharacterLogicHandler)
	if !ok {
		return fmt.Errorf("Invalid interface for CharacterConcernHandler.SetLogicHandler: %T", handler)
	}
	return nil
}

func (cncHdlr *CharacterConcernHandler) SetCrudHandler(storageType string, handler interface{}) error {
	cmds, ok := handler.(CharacterCrudHandler)
	if !ok {
		return fmt.Errorf("Invalid interface for CharacterConcernHandler.SetCrudHandler: %T", handler)
	}
	if cncHdlr.crudHandlers == nil {
		cncHdlr.crudHandlers = make(map[string]CharacterCrudHandler)
	}
	_, ok = cncHdlr.crudHandlers[storageType]
	if ok {
		return fmt.Errorf("CrudHandler already registered for %s/CharacterConcernHandler", storageType)
	}
	cncHdlr.crudHandlers[storageType] = cmds
	return nil
}

func (cncHdlr *CharacterConcernHandler) SetStorageTargets(storageTargets map[string]*rkcy.StorageTargetInit) {
	cncHdlr.storageTargets = storageTargets
	cncHdlr.storageInitHasRun = make(map[string]bool)
}

func (cncHdlr *CharacterConcernHandler) ValidateHandlers() bool {
	if cncHdlr.logicHandler == nil {
		return false
	}
	if cncHdlr.crudHandlers == nil || len(cncHdlr.crudHandlers) == 0 {
		return false
	}
	return true
}

// -----------------------------------------------------------------------------
// Related handling
// -----------------------------------------------------------------------------
func (rel *CharacterRelated) Key() string {
	return rel.Id
}

func CharacterGetRelated(instKey string, instanceStore *rkcy.InstanceStore) (*CharacterRelated, []byte, error) {
	rel := &CharacterRelated{}
	relBytes := instanceStore.GetRelated(instKey)
	if relBytes != nil {
		err := proto.Unmarshal(relBytes, rel)
		if err != nil {
			return nil, nil, err
		}
	}
	return rel, relBytes, nil
}

func (inst *Character) RequestRelatedSteps(instBytes []byte, relBytes []byte) ([]*rkcypb.ApecsTxn_Step, error) {
	var steps []*rkcypb.ApecsTxn_Step

	payload := rkcy.PackPayloads(instBytes, relBytes)

	// Field: Player
	{
		relReq := &rkcypb.RelatedRequest{
			Concern: "Character",
			Key:     inst.Key(),
			Field:   "Player",
			Payload: payload,
		}
		relReqBytes, err := proto.Marshal(relReq)
		if err != nil {
			return nil, err
		}
		steps = append(steps, &rkcypb.ApecsTxn_Step{
			System:  rkcypb.System_PROCESS,
			Concern: "Player",
			Command: rkcy.REQUEST_RELATED,
			Key:     inst.PlayerId,
			Payload: relReqBytes,
		})
	}

	return steps, nil
}

func (rel *CharacterRelated) RefreshRelatedSteps(instKey string, instBytes []byte, relBytes []byte) ([]*rkcypb.ApecsTxn_Step, error) {
	var steps []*rkcypb.ApecsTxn_Step

	payload := rkcy.PackPayloads(instBytes, relBytes)

	if rel.Player != nil {
		relRsp := &rkcypb.RelatedResponse{
			Concern: "Character",
			Field:   "Characters",
			Payload: payload,
		}
		relRspBytes, err := proto.Marshal(relRsp)
		if err != nil {
			return nil, err
		}
		steps = append(steps, &rkcypb.ApecsTxn_Step{
			System:  rkcypb.System_PROCESS,
			Concern: "Player",
			Command: rkcy.REFRESH_RELATED,
			Key:     rel.Player.Key(),
			Payload: relRspBytes,
		})
	}

	// Inject a read step to ensure we keep a consistent
	// result payload type with this concern
	if len(steps) > 0 {
		steps = append(steps, &rkcypb.ApecsTxn_Step{
			System:  rkcypb.System_PROCESS,
			Concern: "Character",
			Command: rkcy.READ,
			Key:     instKey,
		})
	}

	return steps, nil
}

func (rel *CharacterRelated) HandleRequestRelated(relReq *rkcypb.RelatedRequest) (bool, error) {
	// No 'pure' reverse relations, so no need to capture the contents of this request
	return false, nil
}

func (rel *CharacterRelated) HandleRefreshRelated(relRsp *rkcypb.RelatedResponse) (bool, error) {
	switch relRsp.Field {
	case "Player":
		if relRsp.Concern == "Player" {
			instBytes, _, err := rkcy.UnpackPayloads(relRsp.Payload)
			if err != nil {
				return false, err
			}
			inst := &Player{}
			err = proto.Unmarshal(instBytes, inst)
			if err != nil {
				return false, err
			}

			rel.Player = inst
			return true, nil
		} else {
			return false, fmt.Errorf("HandleRefreshRelated: Invalid type '%s' for field Player, expecting 'Player'", relRsp.Concern)
		}
	default:
		return false, fmt.Errorf("HandleRefreshRelated: Invalid field '%s''", relRsp.Field)
	}
}

// -----------------------------------------------------------------------------
// Related handling (END)
// -----------------------------------------------------------------------------

func (cncHdlr *CharacterConcernHandler) HandleLogicCommand(
	ctx context.Context,
	system rkcypb.System,
	command string,
	direction rkcypb.Direction,
	args *rkcy.StepArgs,
	instanceStore *rkcy.InstanceStore,
	confRdr *rkcy.ConfigRdr,
) (*rkcypb.ApecsTxn_Step_Result, []*rkcypb.ApecsTxn_Step) {
	var err error
	rslt := &rkcypb.ApecsTxn_Step_Result{}
	var addSteps []*rkcypb.ApecsTxn_Step // additional steps we want to be run after us

	if direction == rkcypb.Direction_REVERSE && args.ForwardResult == nil {
		rkcy.SetStepResult(rslt, fmt.Errorf("Unable to reverse step with nil ForwardResult"))
		return rslt, nil
	}

	if system != rkcypb.System_PROCESS {
		rkcy.SetStepResult(rslt, fmt.Errorf("Invalid system: %s", system.String()))
		return rslt, nil
	}

	switch command {
	// process handlers
	case rkcy.CREATE:
		rkcy.SetStepResult(rslt, fmt.Errorf("Invalid process command: %s", command))
		return rslt, nil
	case rkcy.READ:
		if direction == rkcypb.Direction_FORWARD {
			related := instanceStore.GetRelated(args.Key)
			rslt.Payload = rkcy.PackPayloads(args.Instance, related)
		} else {
			log.Error().
				Str("Command", command).
				Msg("Invalid command to reverse, no-op")
		}
	case rkcy.UPDATE:
		var payload []byte
		related := instanceStore.GetRelated(args.Key)
		payload = rkcy.PackPayloads(args.Payload, related)
		addSteps = append(
			addSteps,
			&rkcypb.ApecsTxn_Step{
				System:  rkcypb.System_STORAGE,
				Concern: "Character",
				Command: command,
				Key:     args.Key,
				Payload: payload,
			},
		)
	case rkcy.DELETE:
		addSteps = append(
			addSteps,
			&rkcypb.ApecsTxn_Step{
				System:  rkcypb.System_STORAGE,
				Concern: "Character",
				Command: command,
				Key:     args.Key,
				Payload: instanceStore.GetPacked(args.Key),
			},
		)
	case rkcy.REFRESH_INSTANCE:
		// Same operation for both directions
		instBytes, relBytes, err := rkcy.ParsePayload(args.Payload)
		if err != nil {
			rkcy.SetStepResult(rslt, err)
			return rslt, nil
		}
		rel := &CharacterRelated{}
		if relBytes != nil && len(relBytes) > 0 {
			err := proto.Unmarshal(relBytes, rel)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
		} else {
			relBytes = instanceStore.GetRelated(args.Key)
		}
		if args.Instance != nil {
			// refresh related
			relSteps, err := rel.RefreshRelatedSteps(args.Key, instBytes, relBytes)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			if relSteps != nil {
				addSteps = append(addSteps, relSteps...)
			}
		}
		instanceStore.Set(args.Key, instBytes, relBytes, args.CmpdOffset)
		// Re-Get in case Set didn't take due to offset violation
		rslt.Payload = instanceStore.GetPacked(args.Key)
	case rkcy.FLUSH_INSTANCE:
		// Same operation for both directions
		packed := instanceStore.GetPacked(args.Key)
		if packed != nil {
			instanceStore.Remove(args.Key)
		}
		rslt.Payload = packed
	case rkcy.VALIDATE_CREATE:
		if direction == rkcypb.Direction_FORWARD {
			payloadIn := &Character{}
			err = proto.Unmarshal(args.Payload, payloadIn)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			err = payloadIn.PreValidateCreate(ctx)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			payloadOut, err := cncHdlr.logicHandler.ValidateCreate(ctx, payloadIn)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			rslt.Payload, err = proto.Marshal(payloadOut)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
		} else {
			log.Error().
				Str("Command", command).
				Msg("Invalid command to reverse, no-op")
		}
	case rkcy.VALIDATE_UPDATE:
		if direction == rkcypb.Direction_FORWARD {
			if args.Instance == nil {
				rkcy.SetStepResult(rslt, fmt.Errorf("No instance exists during VALIDATE_UPDATE"))
				return rslt, nil
			}
			inst := &Character{}
			err = proto.Unmarshal(args.Instance, inst)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}

			payloadIn := &Character{}
			err = proto.Unmarshal(args.Payload, payloadIn)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			err = inst.PreValidateUpdate(ctx, payloadIn)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			payloadOut, err := cncHdlr.logicHandler.ValidateUpdate(ctx, inst, payloadIn)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			rslt.Payload, err = proto.Marshal(payloadOut)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
		} else {
			log.Error().
				Str("Command", command).
				Msg("Invalid command to reverse, no-op")
		}
	case rkcy.REQUEST_RELATED:
		if direction == rkcypb.Direction_FORWARD {
			relReq := &rkcypb.RelatedRequest{}
			err := proto.Unmarshal(args.Payload, relReq)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			// Get our related concerns from instanceStore
			rel, _, err := CharacterGetRelated(args.Key, instanceStore)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			// Handle refresh related request to see if it fulfills any of our related items
			changed, err := rel.HandleRequestRelated(relReq)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			relBytes, err := proto.Marshal(rel)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			relPayload := rkcy.PackPayloads(args.Instance, relBytes)
			if changed {
				err = instanceStore.SetRelated(args.Key, relBytes, args.CmpdOffset)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
			}
			// Send response containing our instance
			relRsp := &rkcypb.RelatedResponse{
				Concern: "Character",
				Field:   relReq.Field,
				Payload: relPayload,
			}
			relRspBytes, err := proto.Marshal(relRsp)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			rslt.Payload = relRspBytes
			addSteps = append(
				addSteps,
				&rkcypb.ApecsTxn_Step{
					System:        rkcypb.System_PROCESS,
					Concern:       relReq.Concern,
					Command:       rkcy.REFRESH_RELATED,
					Key:           relReq.Key,
					Payload:       rslt.Payload,
					EffectiveTime: timestamppb.Now(),
				},
			)
		} else {
			log.Error().
				Str("Command", command).
				Msg("Invalid command to reverse, no-op")
		}
	case rkcy.REFRESH_RELATED:
		if direction == rkcypb.Direction_FORWARD {
			relRsp := &rkcypb.RelatedResponse{}
			err := proto.Unmarshal(args.Payload, relRsp)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			// Get our related concerns from instanceStore
			rel, _, err := CharacterGetRelated(args.Key, instanceStore)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			// Handle refresh related response
			changed, err := rel.HandleRefreshRelated(relRsp) // dummy
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			relBytes, err := proto.Marshal(rel)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			if changed {
				err = instanceStore.SetRelated(args.Key, relBytes, args.CmpdOffset)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
			}
			rslt.Payload = instanceStore.GetPacked(args.Key)
		} else {
			log.Error().
				Str("Command", command).
				Msg("Invalid command to reverse, no-op")
		}
	case "Fund":
		if direction == rkcypb.Direction_FORWARD {
			if args.Instance == nil {
				rkcy.SetStepResult(rslt, fmt.Errorf("No instance exists during HandleLogicCommand"))
				return rslt, nil
			}
			inst := &Character{}
			err = proto.Unmarshal(args.Instance, inst)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			rel, _, err := CharacterGetRelated(inst.Key(), instanceStore)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			payloadIn := &FundingRequest{}
			err = proto.Unmarshal(args.Payload, payloadIn)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			payloadOut, err := cncHdlr.logicHandler.Fund(ctx, inst, rel, payloadIn)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			rslt.Payload, err = proto.Marshal(payloadOut)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}

			// compare inst to see if it has changed
			instSer, err := proto.Marshal(inst)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			if !bytes.Equal(instSer, args.Instance) {
				// GetRelated again in case command handler has altered it
				rel, relBytes, err := CharacterGetRelated(args.Key, instanceStore)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				relSteps, err := rel.RefreshRelatedSteps(args.Key, instSer, relBytes)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				if relSteps != nil {
					addSteps = append(addSteps, relSteps...)
				}
				rslt.Instance = instSer
				instanceStore.SetInstance(args.Key, rslt.Instance, args.CmpdOffset)
			}
		} else {
			rkcy.SetStepResult(rslt, fmt.Errorf("Reverse not supported for concern command: %s", command))
			return rslt, nil
		}
	case "DebitFunds":
		if direction == rkcypb.Direction_FORWARD {
			if args.Instance == nil {
				rkcy.SetStepResult(rslt, fmt.Errorf("No instance exists during HandleLogicCommand"))
				return rslt, nil
			}
			inst := &Character{}
			err = proto.Unmarshal(args.Instance, inst)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			rel, _, err := CharacterGetRelated(inst.Key(), instanceStore)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			payloadIn := &FundingRequest{}
			err = proto.Unmarshal(args.Payload, payloadIn)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			payloadOut, err := cncHdlr.logicHandler.DebitFunds(ctx, inst, rel, payloadIn)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			rslt.Payload, err = proto.Marshal(payloadOut)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}

			// compare inst to see if it has changed
			instSer, err := proto.Marshal(inst)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			if !bytes.Equal(instSer, args.Instance) {
				// GetRelated again in case command handler has altered it
				rel, relBytes, err := CharacterGetRelated(args.Key, instanceStore)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				relSteps, err := rel.RefreshRelatedSteps(args.Key, instSer, relBytes)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				if relSteps != nil {
					addSteps = append(addSteps, relSteps...)
				}
				rslt.Instance = instSer
				instanceStore.SetInstance(args.Key, rslt.Instance, args.CmpdOffset)
			}
		} else {
			rkcy.SetStepResult(rslt, fmt.Errorf("Reverse not supported for concern command: %s", command))
			return rslt, nil
		}
	case "CreditFunds":
		if direction == rkcypb.Direction_FORWARD {
			if args.Instance == nil {
				rkcy.SetStepResult(rslt, fmt.Errorf("No instance exists during HandleLogicCommand"))
				return rslt, nil
			}
			inst := &Character{}
			err = proto.Unmarshal(args.Instance, inst)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			rel, _, err := CharacterGetRelated(inst.Key(), instanceStore)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			payloadIn := &FundingRequest{}
			err = proto.Unmarshal(args.Payload, payloadIn)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			payloadOut, err := cncHdlr.logicHandler.CreditFunds(ctx, inst, rel, payloadIn)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			rslt.Payload, err = proto.Marshal(payloadOut)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}

			// compare inst to see if it has changed
			instSer, err := proto.Marshal(inst)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
			if !bytes.Equal(instSer, args.Instance) {
				// GetRelated again in case command handler has altered it
				rel, relBytes, err := CharacterGetRelated(args.Key, instanceStore)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				relSteps, err := rel.RefreshRelatedSteps(args.Key, instSer, relBytes)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				if relSteps != nil {
					addSteps = append(addSteps, relSteps...)
				}
				rslt.Instance = instSer
				instanceStore.SetInstance(args.Key, rslt.Instance, args.CmpdOffset)
			}
		} else {
			rkcy.SetStepResult(rslt, fmt.Errorf("Reverse not supported for concern command: %s", command))
			return rslt, nil
		}
	default:
		rkcy.SetStepResult(rslt, fmt.Errorf("Invalid process command: %s", command))
		return rslt, nil
	}

	if rslt.Code == rkcypb.Code_UNDEFINED {
		rslt.Code = rkcypb.Code_OK
	}
	return rslt, addSteps
}

func (cncHdlr *CharacterConcernHandler) HandleCrudCommand(
	ctx context.Context,
	system rkcypb.System,
	command string,
	direction rkcypb.Direction,
	args *rkcy.StepArgs,
	storageTarget string,
	wg *sync.WaitGroup,
) (*rkcypb.ApecsTxn_Step_Result, []*rkcypb.ApecsTxn_Step) {
	var err error
	rslt := &rkcypb.ApecsTxn_Step_Result{}
	var addSteps []*rkcypb.ApecsTxn_Step // additional steps we want to be run after us

	if direction == rkcypb.Direction_REVERSE && args.ForwardResult == nil {
		rkcy.SetStepResult(rslt, fmt.Errorf("Unable to reverse step with nil ForwardResult"))
		return rslt, nil
	}

	if !rkcy.IsStorageSystem(system) {
		rkcy.SetStepResult(rslt, fmt.Errorf("Invalid system: %s", system.String()))
		return rslt, nil
	}

	stgTgt, ok := cncHdlr.storageTargets[storageTarget]
	if !ok {
		rkcy.SetStepResult(rslt, fmt.Errorf("No matching StorageTarget: %s, %+v", storageTarget, cncHdlr.storageTargets))
		return rslt, nil
	}

	handler, ok := cncHdlr.crudHandlers[stgTgt.Type]
	if !ok {
		rkcy.SetStepResult(rslt, fmt.Errorf("No CrudHandler for %s", stgTgt.Type))
		return rslt, nil
	}

	if !cncHdlr.storageInitHasRun[storageTarget] {
		if stgTgt.Init != nil {
			err := stgTgt.Init(ctx, stgTgt.Config, wg)
			if err != nil {
				rkcy.SetStepResult(rslt, err)
				return rslt, nil
			}
		}
		cncHdlr.storageInitHasRun[storageTarget] = true
	}

	switch system {
	case rkcypb.System_STORAGE:
		if !stgTgt.IsPrimary {
			rkcy.SetStepResult(rslt, fmt.Errorf("Storage target is not primary in STORAGE target: %s, command: %s", stgTgt.Name, command))
			return rslt, nil
		}
		switch command {
		// storage handlers
		case rkcy.CREATE:
			{
				instBytes, relBytes, err := rkcy.ParsePayload(args.Payload)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				inst := &Character{}
				err = proto.Unmarshal(instBytes, inst)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				instOut, err := handler.Create(ctx, inst, args.CmpdOffset)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				// NOTE: From this point on any error handler must attempt to
				// Delete from the datastore, if this step fails, a reversal
				// step will not be run, so we must ensure we do it here.
				if instOut.Key() == "" {
					err := fmt.Errorf("No Key set in CREATE Character")
					errDel := handler.Delete(ctx, instOut.Key(), args.CmpdOffset)
					if errDel != nil {
						rkcy.SetStepResult(rslt, fmt.Errorf("DELETE error cleaning up CREATE failure, orig err: '%s', DELETE err: '%s'", err.Error(), errDel.Error()))
						return rslt, nil
					}
					rkcy.SetStepResult(rslt, fmt.Errorf("No Key set in CREATE Character"))
					return rslt, nil
				}
				instOutBytes, err := proto.Marshal(instOut)
				if err != nil {
					errDel := handler.Delete(ctx, instOut.Key(), args.CmpdOffset)
					if errDel != nil {
						rkcy.SetStepResult(rslt, fmt.Errorf("DELETE error cleaning up CREATE failure, orig err: '%s', DELETE err: '%s'", err.Error(), errDel.Error()))
						return rslt, nil
					}
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				rslt.Key = instOut.Key()
				rslt.CmpdOffset = args.CmpdOffset // for possible delete in rollback
				rslt.Payload = rkcy.PackPayloads(instOutBytes, relBytes)
				// Add a refresh step which will update the instance cache
				addSteps = append(
					addSteps,
					&rkcypb.ApecsTxn_Step{
						System:        rkcypb.System_PROCESS,
						Concern:       "Character",
						Command:       rkcy.REFRESH_INSTANCE,
						Key:           rslt.Key,
						Payload:       rslt.Payload,
						EffectiveTime: timestamppb.New(args.EffectiveTime),
					},
				)
				// Request related refresh messages from our related concerns
				relSteps, err := instOut.RequestRelatedSteps(instOutBytes, nil)
				if err != nil {
					errDel := handler.Delete(ctx, instOut.Key(), args.CmpdOffset)
					if errDel != nil {
						rkcy.SetStepResult(rslt, fmt.Errorf("DELETE error cleaning up CREATE failure, orig err: '%s', DELETE err: '%s'", err.Error(), errDel.Error()))
						return rslt, nil
					}
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				if relSteps != nil {
					addSteps = append(addSteps, relSteps...)
				}
			}
		case rkcy.READ:
			{
				var inst *Character
				var rel *CharacterRelated
				inst, rel, rslt.CmpdOffset, err = handler.Read(ctx, args.Key)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				instBytes, err := proto.Marshal(inst)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				var relBytes []byte
				relBytes, err = proto.Marshal(rel)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				rslt.Key = inst.Key()
				rslt.Payload = rkcy.PackPayloads(instBytes, relBytes)
				addSteps = append(
					addSteps,
					&rkcypb.ApecsTxn_Step{
						System:        rkcypb.System_PROCESS,
						Concern:       "Character",
						Command:       rkcy.REFRESH_INSTANCE,
						Key:           rslt.Key,
						Payload:       rslt.Payload,
						EffectiveTime: timestamppb.New(args.EffectiveTime),
					},
				)
			}
		case rkcy.UPDATE:
			fallthrough
		case rkcy.UPDATE_ASYNC:
			{
				instBytes, relBytes, err := rkcy.ParsePayload(args.Payload)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				inst := &Character{}
				err = proto.Unmarshal(instBytes, inst)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				rel := &CharacterRelated{}
				if relBytes != nil && len(relBytes) > 0 {
					err = proto.Unmarshal(relBytes, rel)
					if err != nil {
						rkcy.SetStepResult(rslt, err)
						return rslt, nil
					}
				}
				err = handler.Update(ctx, inst, rel, args.CmpdOffset)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				if command == rkcy.UPDATE {
					// Only do a refresh after an UPDATE, not an UPDATE_ASYNC
					rslt.Key = inst.Key()
					rslt.Instance = args.Instance
					rslt.Payload = rkcy.PackPayloads(instBytes, relBytes)
					// Add a refresh step which will update the instance cache
					addSteps = append(
						addSteps,
						&rkcypb.ApecsTxn_Step{
							System:        rkcypb.System_PROCESS,
							Concern:       "Character",
							Command:       rkcy.REFRESH_INSTANCE,
							Key:           rslt.Key,
							Payload:       rslt.Payload,
							EffectiveTime: timestamppb.New(args.EffectiveTime),
						},
					)
				}
			}
		case rkcy.DELETE:
			{
				err = handler.Delete(ctx, args.Key, args.CmpdOffset)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				rslt.Key = args.Key
				addSteps = append(
					addSteps,
					&rkcypb.ApecsTxn_Step{
						System:        rkcypb.System_PROCESS,
						Concern:       "Character",
						Command:       rkcy.FLUSH_INSTANCE,
						Key:           rslt.Key,
						EffectiveTime: timestamppb.New(args.EffectiveTime),
					},
				)
			}
		default:
			rkcy.SetStepResult(rslt, fmt.Errorf("Invalid storage command: %s", command))
			return rslt, nil
		}
	case rkcypb.System_STORAGE_SCND:
		if stgTgt.IsPrimary {
			rkcy.SetStepResult(rslt, fmt.Errorf("Storage target is primary in STORAGE_SCND handler: %s, command: %s", stgTgt.Name, command))
			return rslt, nil
		}
		if direction != rkcypb.Direction_FORWARD {
			rkcy.SetStepResult(rslt, fmt.Errorf("Reverse direction not supported in STORAGE_SCND handler: %s, command: %s", stgTgt.Name, command))
			return rslt, nil
		}
		switch command {
		case rkcy.CREATE:
			{
				instBytes, _, err := rkcy.ParsePayload(args.Payload)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				inst := &Character{}
				err = proto.Unmarshal(instBytes, inst)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				if inst.Key() == "" {
					rkcy.SetStepResult(rslt, fmt.Errorf("Empty key in CREATE STORAGE_SCND"))
					return rslt, nil
				}
				_, err = handler.Create(ctx, inst, args.CmpdOffset)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				rslt.Key = inst.Key()
				// ignore Marshal error, this step is just to get the data into secondary
				rslt.Payload = args.Payload
			}
		case rkcy.UPDATE:
			fallthrough
		case rkcy.UPDATE_ASYNC:
			{
				instBytes, relBytes, err := rkcy.ParsePayload(args.Payload)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				inst := &Character{}
				err = proto.Unmarshal(instBytes, inst)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				rel := &CharacterRelated{}
				if relBytes != nil && len(relBytes) > 0 {
					err = proto.Unmarshal(relBytes, rel)
					if err != nil {
						rkcy.SetStepResult(rslt, err)
						return rslt, nil
					}
				}
				err = handler.Update(ctx, inst, rel, args.CmpdOffset)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
			}
		case rkcy.DELETE:
			{
				err = handler.Delete(ctx, args.Key, args.CmpdOffset)
				if err != nil {
					rkcy.SetStepResult(rslt, err)
					return rslt, nil
				}
				rslt.Key = args.Key
			}
		default:
			rkcy.SetStepResult(rslt, fmt.Errorf("Invalid storage command: %s", command))
			return rslt, nil
		}
	}

	if rslt.Code == rkcypb.Code_UNDEFINED {
		rslt.Code = rkcypb.Code_OK
	}
	return rslt, addSteps
}

func (*CharacterConcernHandler) DecodeInstance(
	ctx context.Context,
	buffer []byte,
) (*rkcy.ResultProto, error) {
	resProto := &rkcy.ResultProto{
		Type: "Character",
	}

	if !rkcy.IsPackedPayload(buffer) {
		inst := &Character{}
		err := proto.Unmarshal(buffer, inst)
		if err != nil {
			return nil, err
		}
		resProto.Instance = inst
	} else {
		instBytes, relBytes, err := rkcy.UnpackPayloads(buffer)
		if err != nil {
			return nil, err
		}

		inst := &Character{}
		err = proto.Unmarshal(instBytes, inst)
		if err != nil {
			return nil, err
		}
		resProto.Instance = inst

		if relBytes != nil && len(relBytes) > 0 {
			rel := &CharacterRelated{}
			err := proto.Unmarshal(relBytes, rel)
			if err != nil {
				return nil, err
			}
			resProto.Related = rel
		} else {
			resProto.Related = &CharacterRelated{}
		}
	}

	return resProto, nil
}

func (cncHdlr *CharacterConcernHandler) DecodeArg(
	ctx context.Context,
	system rkcypb.System,
	command string,
	buffer []byte,
) (*rkcy.ResultProto, error) {
	switch system {
	case rkcypb.System_STORAGE:
		fallthrough
	case rkcypb.System_STORAGE_SCND:
		switch command {
		case rkcy.CREATE:
			fallthrough
		case rkcy.READ:
			fallthrough
		case rkcy.UPDATE:
			fallthrough
		case rkcy.UPDATE_ASYNC:
			return cncHdlr.DecodeInstance(ctx, buffer)
		default:
			return nil, fmt.Errorf("ArgDecoder invalid command: %d %s", system, command)
		}
	case rkcypb.System_PROCESS:
		switch command {
		case rkcy.VALIDATE_CREATE:
			fallthrough
		case rkcy.VALIDATE_UPDATE:
			fallthrough
		case rkcy.REFRESH_INSTANCE:
			return cncHdlr.DecodeInstance(ctx, buffer)
		case rkcy.REQUEST_RELATED:
			{
				relReq := &rkcypb.RelatedRequest{}
				err := proto.Unmarshal(buffer, relReq)
				if err != nil {
					return nil, err
				}
				return &rkcy.ResultProto{Type: "RelatedRequest", Instance: relReq}, nil
			}
		case rkcy.REFRESH_RELATED:
			{
				relRsp := &rkcypb.RelatedResponse{}
				err := proto.Unmarshal(buffer, relRsp)
				if err != nil {
					return nil, err
				}
				return &rkcy.ResultProto{Type: "RelatedResponse", Instance: relRsp}, nil
			}
		case "Fund":
			{
				pb := &FundingRequest{}
				err := proto.Unmarshal(buffer, pb)
				if err != nil {
					return nil, err
				}
				return &rkcy.ResultProto{Type: "FundingRequest", Instance: pb}, nil
			}
		case "DebitFunds":
			{
				pb := &FundingRequest{}
				err := proto.Unmarshal(buffer, pb)
				if err != nil {
					return nil, err
				}
				return &rkcy.ResultProto{Type: "FundingRequest", Instance: pb}, nil
			}
		case "CreditFunds":
			{
				pb := &FundingRequest{}
				err := proto.Unmarshal(buffer, pb)
				if err != nil {
					return nil, err
				}
				return &rkcy.ResultProto{Type: "FundingRequest", Instance: pb}, nil
			}
		default:
			return nil, fmt.Errorf("ArgDecoder invalid command: %d %s", system, command)
		}
	default:
		return nil, fmt.Errorf("ArgDecoder invalid system: %s", system.String())
	}
}

func (cncHdlr *CharacterConcernHandler) DecodeResult(
	ctx context.Context,
	system rkcypb.System,
	command string,
	buffer []byte,
) (*rkcy.ResultProto, error) {
	switch system {
	case rkcypb.System_STORAGE:
		fallthrough
	case rkcypb.System_STORAGE_SCND:
		switch command {
		case rkcy.READ:
			fallthrough
		case rkcy.CREATE:
			fallthrough
		case rkcy.UPDATE:
			fallthrough
		case rkcy.UPDATE_ASYNC:
			return cncHdlr.DecodeInstance(ctx, buffer)
		default:
			return nil, fmt.Errorf("ResultDecoder invalid command: %d %s", system, command)
		}
	case rkcypb.System_PROCESS:
		switch command {
		case rkcy.READ:
			fallthrough
		case rkcy.VALIDATE_CREATE:
			fallthrough
		case rkcy.VALIDATE_UPDATE:
			fallthrough
		case rkcy.REFRESH_INSTANCE:
			fallthrough
		case rkcy.REFRESH_RELATED:
			return cncHdlr.DecodeInstance(ctx, buffer)
		case rkcy.REQUEST_RELATED:
			{
				relRsp := &rkcypb.RelatedResponse{}
				err := proto.Unmarshal(buffer, relRsp)
				if err != nil {
					return nil, err
				}
				return &rkcy.ResultProto{Type: "RelatedResponse", Instance: relRsp}, nil
			}
		case "Fund":
			{
				pb := &Character{}
				err := proto.Unmarshal(buffer, pb)
				if err != nil {
					return nil, err
				}
				return &rkcy.ResultProto{Type: "Character", Instance: pb}, nil
			}
		case "DebitFunds":
			{
				pb := &Character{}
				err := proto.Unmarshal(buffer, pb)
				if err != nil {
					return nil, err
				}
				return &rkcy.ResultProto{Type: "Character", Instance: pb}, nil
			}
		case "CreditFunds":
			{
				pb := &Character{}
				err := proto.Unmarshal(buffer, pb)
				if err != nil {
					return nil, err
				}
				return &rkcy.ResultProto{Type: "Character", Instance: pb}, nil
			}
		default:
			return nil, fmt.Errorf("ResultDecoder invalid command: %d %s", system, command)
		}
	default:
		return nil, fmt.Errorf("ResultDecoder invalid system: %s", system.String())
	}
}

func (cncHdlr *CharacterConcernHandler) DecodeRelatedRequest(
	ctx context.Context,
	relReq *rkcypb.RelatedRequest,
) (*rkcy.ResultProto, error) {
	// No 'pure' reverse relations, so we should never be decoding
	return nil, fmt.Errorf("DecodeRelatedRequest: Invalid concern")
}

// -----------------------------------------------------------------------------
// Concern Character END
// -----------------------------------------------------------------------------
