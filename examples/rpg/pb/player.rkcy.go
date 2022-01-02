// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package pb

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"github.com/lachlanorr/rocketcycle/pkg/apecs"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

// -----------------------------------------------------------------------------
// Config Limits
// -----------------------------------------------------------------------------
type LimitsConfigHandler struct{}

func (conf *Limits) Key() string {
	return conf.Id
}

func (*LimitsConfigHandler) GetKey(msg proto.Message) string {
	conf := msg.(*Limits)
	return conf.Key()
}

func (*LimitsConfigHandler) Unmarshal(b []byte) (proto.Message, error) {
	conf := &Limits{}
	err := proto.Unmarshal(b, conf)
	if err != nil {
		return nil, err
	}
	return proto.Message(conf), nil
}

func (*LimitsConfigHandler) UnmarshalJson(b []byte) (proto.Message, error) {
	conf := &Limits{}
	err := protojson.Unmarshal(b, conf)
	if err != nil {
		return nil, err
	}
	return proto.Message(conf), nil
}

func init() {
	rkcy.RegisterComplexConfigHandler("Limits", &LimitsConfigHandler{})
}

// -----------------------------------------------------------------------------
// Config Limits END
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// Concern Player
// -----------------------------------------------------------------------------
func init() {
	rkcy.RegisterGlobalConcernHandler(&PlayerConcernHandler{})
}

func (inst *Player) Key() string {
	return inst.Id
}

func MarshalPlayerOrPanic(inst *Player) []byte {
	b, err := proto.Marshal(inst)
	if err != nil {
		panic(err.Error())
	}
	return b
}

func (inst *Player) PreValidateCreate(ctx context.Context) error {
	if inst.Key() != "" {
		return rkcy.NewError(rkcypb.Code_INVALID_ARGUMENT, "Empty Key during PreValidateCreate for Player")
	}
	return nil
}

func (inst *Player) PreValidateUpdate(ctx context.Context, updated *Player) error {
	if updated.Key() == "" || inst.Key() != updated.Key() {
		return rkcy.NewError(rkcypb.Code_INVALID_ARGUMENT, "Mismatched Keys during PreValidateUpdate for Player")
	}
	return nil
}

// LogicHandler Interface
type PlayerLogicHandler interface {
	ValidateCreate(ctx context.Context, inst *Player) (*Player, error)
	ValidateUpdate(ctx context.Context, original *Player, updated *Player) (*Player, error)
}

// CrudHandler Interface
type PlayerCrudHandler interface {
	Read(ctx context.Context, key string) (*Player, *PlayerRelated, *rkcypb.CompoundOffset, error)
	Create(ctx context.Context, inst *Player, cmpdOffset *rkcypb.CompoundOffset) (*Player, error)
	Update(ctx context.Context, inst *Player, rel *PlayerRelated, cmpdOffset *rkcypb.CompoundOffset) error
	Delete(ctx context.Context, key string, cmpdOffset *rkcypb.CompoundOffset) error
}

// Concern Handler
type PlayerConcernHandler struct {
	logicHandler      PlayerLogicHandler
	crudHandlers      map[string]PlayerCrudHandler
	storageTargets    map[string]*rkcy.StorageTargetInit
	storageInitHasRun map[string]bool
}

func (*PlayerConcernHandler) ConcernName() string {
	return "Player"
}

func (cncHdlr *PlayerConcernHandler) SetLogicHandler(handler interface{}) error {
	if cncHdlr.logicHandler != nil {
		return fmt.Errorf("ProcessHandler already registered for PlayerConcernHandler")
	}
	var ok bool
	cncHdlr.logicHandler, ok = handler.(PlayerLogicHandler)
	if !ok {
		return fmt.Errorf("Invalid interface for PlayerConcernHandler.SetLogicHandler: %T", handler)
	}
	return nil
}

func (cncHdlr *PlayerConcernHandler) SetCrudHandler(storageType string, handler interface{}) error {
	cmds, ok := handler.(PlayerCrudHandler)
	if !ok {
		return fmt.Errorf("Invalid interface for PlayerConcernHandler.SetCrudHandler: %T", handler)
	}
	if cncHdlr.crudHandlers == nil {
		cncHdlr.crudHandlers = make(map[string]PlayerCrudHandler)
	}
	_, ok = cncHdlr.crudHandlers[storageType]
	if ok {
		return fmt.Errorf("CrudHandler already registered for %s/PlayerConcernHandler", storageType)
	}
	cncHdlr.crudHandlers[storageType] = cmds
	return nil
}

func (cncHdlr *PlayerConcernHandler) SetStorageTargets(storageTargets map[string]*rkcy.StorageTargetInit) {
	cncHdlr.storageTargets = storageTargets
	cncHdlr.storageInitHasRun = make(map[string]bool)
}

func (cncHdlr *PlayerConcernHandler) ValidateHandlers() bool {
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
func (rel *PlayerRelated) Key() string {
	return rel.Id
}

func PlayerGetRelated(instKey string, instanceStore *rkcy.InstanceStore) (*PlayerRelated, []byte, error) {
	rel := &PlayerRelated{}
	relBytes := instanceStore.GetRelated(instKey)
	if relBytes != nil {
		err := proto.Unmarshal(relBytes, rel)
		if err != nil {
			return nil, nil, err
		}
	}
	return rel, relBytes, nil
}

func (inst *Player) RequestRelatedSteps(instBytes []byte, relBytes []byte) ([]*rkcypb.ApecsTxn_Step, error) {
	// No forward related Concerns
	return nil, nil
}

func (rel *PlayerRelated) RefreshRelatedSteps(instKey string, instBytes []byte, relBytes []byte) ([]*rkcypb.ApecsTxn_Step, error) {
	var steps []*rkcypb.ApecsTxn_Step

	payload := rkcy.PackPayloads(instBytes, relBytes)

	// Field: Characters
	for _, itm := range rel.Characters {
		relRsp := &rkcypb.RelatedResponse{
			Concern: "Player",
			Field:   "Player",
			Payload: payload,
		}
		relRspBytes, err := proto.Marshal(relRsp)
		if err != nil {
			return nil, err
		}
		steps = append(steps, &rkcypb.ApecsTxn_Step{
			System:  rkcypb.System_PROCESS,
			Concern: "Character",
			Command: rkcy.REFRESH_RELATED,
			Key:     itm.Key(),
			Payload: relRspBytes,
		})
	}

	// Inject a read step to ensure we keep a consistent
	// result payload type with this concern
	if len(steps) > 0 {
		steps = append(steps, &rkcypb.ApecsTxn_Step{
			System:  rkcypb.System_PROCESS,
			Concern: "Player",
			Command: rkcy.READ,
			Key:     instKey,
		})
	}

	return steps, nil
}

func (rel *PlayerRelated) UpsertRelatedCharacters(msg *Character) {
	idx := -1
	// check to see if it's already here
	for i, itm := range rel.Characters {
		if itm.Key() == msg.Key() {
			idx = i
			break
		}
	}
	if idx == -1 {
		rel.Characters = append(rel.Characters, msg)
	} else {
		rel.Characters[idx] = msg
	}
}

func (rel *PlayerRelated) HandleRequestRelated(relReq *rkcypb.RelatedRequest) (bool, error) {
	switch relReq.Field {
	case "Player":
		if relReq.Concern == "Character" {
			instBytes, _, err := rkcy.UnpackPayloads(relReq.Payload)
			if err != nil {
				return false, err
			}
			inst := &Character{}
			err = proto.Unmarshal(instBytes, inst)
			if err != nil {
				return false, err
			}

			rel.UpsertRelatedCharacters(inst)
			return true, nil
		} else {
			return false, fmt.Errorf("HandleRequestRelated: Invalid type '%s' for field Characters, expecting 'Character'", relReq.Concern)
		}
	default:
		return false, fmt.Errorf("HandleRequestRelated: Invalid field '%s''", relReq.Field)
	}
	return false, nil
}

func (rel *PlayerRelated) HandleRefreshRelated(relRsp *rkcypb.RelatedResponse) (bool, error) {
	switch relRsp.Field {
	case "Characters":
		if relRsp.Concern == "Character" {
			instBytes, _, err := rkcy.UnpackPayloads(relRsp.Payload)
			if err != nil {
				return false, err
			}
			inst := &Character{}
			err = proto.Unmarshal(instBytes, inst)
			if err != nil {
				return false, err
			}

			rel.UpsertRelatedCharacters(inst)
			return true, nil
		} else {
			return false, fmt.Errorf("HandleRefreshRelated: Invalid type '%s' for field Characters, expecting 'Character'", relRsp.Concern)
		}
	default:
		return false, fmt.Errorf("HandleRefreshRelated: Invalid field '%s''", relRsp.Field)
	}
}

// -----------------------------------------------------------------------------
// Related handling (END)
// -----------------------------------------------------------------------------

func (cncHdlr *PlayerConcernHandler) HandleLogicCommand(
	ctx context.Context,
	system rkcypb.System,
	command string,
	direction rkcypb.Direction,
	args *rkcy.StepArgs,
	instanceStore *rkcy.InstanceStore,
	confRdr rkcy.ConfigRdr,
) (*rkcypb.ApecsTxn_Step_Result, []*rkcypb.ApecsTxn_Step) {
	var err error
	rslt := &rkcypb.ApecsTxn_Step_Result{}
	var addSteps []*rkcypb.ApecsTxn_Step // additional steps we want to be run after us

	if direction == rkcypb.Direction_REVERSE && args.ForwardResult == nil {
		apecs.SetStepResult(rslt, fmt.Errorf("Unable to reverse step with nil ForwardResult"))
		return rslt, nil
	}

	if system != rkcypb.System_PROCESS {
		apecs.SetStepResult(rslt, fmt.Errorf("Invalid system: %s", system.String()))
		return rslt, nil
	}

	switch command {
	// process handlers
	case rkcy.CREATE:
		apecs.SetStepResult(rslt, fmt.Errorf("Invalid process command: %s", command))
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
				Concern: "Player",
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
				Concern: "Player",
				Command: command,
				Key:     args.Key,
				Payload: instanceStore.GetPacked(args.Key),
			},
		)
	case rkcy.REFRESH_INSTANCE:
		// Same operation for both directions
		instBytes, relBytes, err := rkcy.ParsePayload(args.Payload)
		if err != nil {
			apecs.SetStepResult(rslt, err)
			return rslt, nil
		}
		rel := &PlayerRelated{}
		if relBytes != nil && len(relBytes) > 0 {
			err := proto.Unmarshal(relBytes, rel)
			if err != nil {
				apecs.SetStepResult(rslt, err)
				return rslt, nil
			}
		} else {
			relBytes = instanceStore.GetRelated(args.Key)
		}
		if args.Instance != nil {
			// refresh related
			relSteps, err := rel.RefreshRelatedSteps(args.Key, instBytes, relBytes)
			if err != nil {
				apecs.SetStepResult(rslt, err)
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
			payloadIn := &Player{}
			err = proto.Unmarshal(args.Payload, payloadIn)
			if err != nil {
				apecs.SetStepResult(rslt, err)
				return rslt, nil
			}
			err = payloadIn.PreValidateCreate(ctx)
			if err != nil {
				apecs.SetStepResult(rslt, err)
				return rslt, nil
			}
			payloadOut, err := cncHdlr.logicHandler.ValidateCreate(ctx, payloadIn)
			if err != nil {
				apecs.SetStepResult(rslt, err)
				return rslt, nil
			}
			rslt.Payload, err = proto.Marshal(payloadOut)
			if err != nil {
				apecs.SetStepResult(rslt, err)
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
				apecs.SetStepResult(rslt, fmt.Errorf("No instance exists during VALIDATE_UPDATE"))
				return rslt, nil
			}
			inst := &Player{}
			err = proto.Unmarshal(args.Instance, inst)
			if err != nil {
				apecs.SetStepResult(rslt, err)
				return rslt, nil
			}

			payloadIn := &Player{}
			err = proto.Unmarshal(args.Payload, payloadIn)
			if err != nil {
				apecs.SetStepResult(rslt, err)
				return rslt, nil
			}
			err = inst.PreValidateUpdate(ctx, payloadIn)
			if err != nil {
				apecs.SetStepResult(rslt, err)
				return rslt, nil
			}
			payloadOut, err := cncHdlr.logicHandler.ValidateUpdate(ctx, inst, payloadIn)
			if err != nil {
				apecs.SetStepResult(rslt, err)
				return rslt, nil
			}
			rslt.Payload, err = proto.Marshal(payloadOut)
			if err != nil {
				apecs.SetStepResult(rslt, err)
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
				apecs.SetStepResult(rslt, err)
				return rslt, nil
			}
			// Get our related concerns from instanceStore
			rel, _, err := PlayerGetRelated(args.Key, instanceStore)
			if err != nil {
				apecs.SetStepResult(rslt, err)
				return rslt, nil
			}
			// Handle refresh related request to see if it fulfills any of our related items
			changed, err := rel.HandleRequestRelated(relReq)
			if err != nil {
				apecs.SetStepResult(rslt, err)
				return rslt, nil
			}
			relBytes, err := proto.Marshal(rel)
			if err != nil {
				apecs.SetStepResult(rslt, err)
				return rslt, nil
			}
			relPayload := rkcy.PackPayloads(args.Instance, relBytes)
			if changed {
				err = instanceStore.SetRelated(args.Key, relBytes, args.CmpdOffset)
				if err != nil {
					apecs.SetStepResult(rslt, err)
					return rslt, nil
				}
			}
			// Send response containing our instance
			relRsp := &rkcypb.RelatedResponse{
				Concern: "Player",
				Field:   relReq.Field,
				Payload: relPayload,
			}
			relRspBytes, err := proto.Marshal(relRsp)
			if err != nil {
				apecs.SetStepResult(rslt, err)
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
				apecs.SetStepResult(rslt, err)
				return rslt, nil
			}
			// Get our related concerns from instanceStore
			rel, _, err := PlayerGetRelated(args.Key, instanceStore)
			if err != nil {
				apecs.SetStepResult(rslt, err)
				return rslt, nil
			}
			// Handle refresh related response
			changed, err := rel.HandleRefreshRelated(relRsp) // dummy
			if err != nil {
				apecs.SetStepResult(rslt, err)
				return rslt, nil
			}
			relBytes, err := proto.Marshal(rel)
			if err != nil {
				apecs.SetStepResult(rslt, err)
				return rslt, nil
			}
			if changed {
				err = instanceStore.SetRelated(args.Key, relBytes, args.CmpdOffset)
				if err != nil {
					apecs.SetStepResult(rslt, err)
					return rslt, nil
				}
			}
			rslt.Payload = instanceStore.GetPacked(args.Key)
		} else {
			log.Error().
				Str("Command", command).
				Msg("Invalid command to reverse, no-op")
		}
	default:
		apecs.SetStepResult(rslt, fmt.Errorf("Invalid process command: %s", command))
		return rslt, nil
	}

	if rslt.Code == rkcypb.Code_UNDEFINED {
		rslt.Code = rkcypb.Code_OK
	}
	return rslt, addSteps
}

func (cncHdlr *PlayerConcernHandler) HandleCrudCommand(
	ctx context.Context,
	wg *sync.WaitGroup,
	system rkcypb.System,
	command string,
	direction rkcypb.Direction,
	args *rkcy.StepArgs,
	storageTarget string,
) (*rkcypb.ApecsTxn_Step_Result, []*rkcypb.ApecsTxn_Step) {
	var err error
	rslt := &rkcypb.ApecsTxn_Step_Result{}
	var addSteps []*rkcypb.ApecsTxn_Step // additional steps we want to be run after us

	if direction == rkcypb.Direction_REVERSE && args.ForwardResult == nil {
		apecs.SetStepResult(rslt, fmt.Errorf("Unable to reverse step with nil ForwardResult"))
		return rslt, nil
	}

	if !rkcy.IsStorageSystem(system) {
		apecs.SetStepResult(rslt, fmt.Errorf("Invalid system: %s", system.String()))
		return rslt, nil
	}

	stgTgt, ok := cncHdlr.storageTargets[storageTarget]
	if !ok {
		apecs.SetStepResult(rslt, fmt.Errorf("No matching StorageTarget: %s, %+v", storageTarget, cncHdlr.storageTargets))
		return rslt, nil
	}

	handler, ok := cncHdlr.crudHandlers[stgTgt.Type]
	if !ok {
		apecs.SetStepResult(rslt, fmt.Errorf("No CrudHandler for %s", stgTgt.Type))
		return rslt, nil
	}

	if !cncHdlr.storageInitHasRun[storageTarget] {
		if stgTgt.Init != nil {
			err := stgTgt.Init(ctx, wg, stgTgt.Config)
			if err != nil {
				apecs.SetStepResult(rslt, err)
				return rslt, nil
			}
		}
		cncHdlr.storageInitHasRun[storageTarget] = true
	}

	switch system {
	case rkcypb.System_STORAGE:
		if !stgTgt.IsPrimary {
			apecs.SetStepResult(rslt, fmt.Errorf("Storage target is not primary in STORAGE target: %s, command: %s", stgTgt.Name, command))
			return rslt, nil
		}
		switch command {
		// storage handlers
		case rkcy.CREATE:
			{
				instBytes, relBytes, err := rkcy.ParsePayload(args.Payload)
				if err != nil {
					apecs.SetStepResult(rslt, err)
					return rslt, nil
				}
				inst := &Player{}
				err = proto.Unmarshal(instBytes, inst)
				if err != nil {
					apecs.SetStepResult(rslt, err)
					return rslt, nil
				}
				instOut, err := handler.Create(ctx, inst, args.CmpdOffset)
				if err != nil {
					apecs.SetStepResult(rslt, err)
					return rslt, nil
				}
				// NOTE: From this point on any error handler must attempt to
				// Delete from the datastore, if this step fails, a reversal
				// step will not be run, so we must ensure we do it here.
				if instOut.Key() == "" {
					err := fmt.Errorf("No Key set in CREATE Player")
					errDel := handler.Delete(ctx, instOut.Key(), args.CmpdOffset)
					if errDel != nil {
						apecs.SetStepResult(rslt, fmt.Errorf("DELETE error cleaning up CREATE failure, orig err: '%s', DELETE err: '%s'", err.Error(), errDel.Error()))
						return rslt, nil
					}
					apecs.SetStepResult(rslt, fmt.Errorf("No Key set in CREATE Player"))
					return rslt, nil
				}
				instOutBytes, err := proto.Marshal(instOut)
				if err != nil {
					errDel := handler.Delete(ctx, instOut.Key(), args.CmpdOffset)
					if errDel != nil {
						apecs.SetStepResult(rslt, fmt.Errorf("DELETE error cleaning up CREATE failure, orig err: '%s', DELETE err: '%s'", err.Error(), errDel.Error()))
						return rslt, nil
					}
					apecs.SetStepResult(rslt, err)
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
						Concern:       "Player",
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
						apecs.SetStepResult(rslt, fmt.Errorf("DELETE error cleaning up CREATE failure, orig err: '%s', DELETE err: '%s'", err.Error(), errDel.Error()))
						return rslt, nil
					}
					apecs.SetStepResult(rslt, err)
					return rslt, nil
				}
				if relSteps != nil {
					addSteps = append(addSteps, relSteps...)
				}
			}
		case rkcy.READ:
			{
				var inst *Player
				var rel *PlayerRelated
				inst, rel, rslt.CmpdOffset, err = handler.Read(ctx, args.Key)
				if err != nil {
					apecs.SetStepResult(rslt, err)
					return rslt, nil
				}
				instBytes, err := proto.Marshal(inst)
				if err != nil {
					apecs.SetStepResult(rslt, err)
					return rslt, nil
				}
				var relBytes []byte
				relBytes, err = proto.Marshal(rel)
				if err != nil {
					apecs.SetStepResult(rslt, err)
					return rslt, nil
				}
				rslt.Key = inst.Key()
				rslt.Payload = rkcy.PackPayloads(instBytes, relBytes)
				addSteps = append(
					addSteps,
					&rkcypb.ApecsTxn_Step{
						System:        rkcypb.System_PROCESS,
						Concern:       "Player",
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
					apecs.SetStepResult(rslt, err)
					return rslt, nil
				}
				inst := &Player{}
				err = proto.Unmarshal(instBytes, inst)
				if err != nil {
					apecs.SetStepResult(rslt, err)
					return rslt, nil
				}
				rel := &PlayerRelated{}
				if relBytes != nil && len(relBytes) > 0 {
					err = proto.Unmarshal(relBytes, rel)
					if err != nil {
						apecs.SetStepResult(rslt, err)
						return rslt, nil
					}
				}
				err = handler.Update(ctx, inst, rel, args.CmpdOffset)
				if err != nil {
					apecs.SetStepResult(rslt, err)
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
							Concern:       "Player",
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
					apecs.SetStepResult(rslt, err)
					return rslt, nil
				}
				rslt.Key = args.Key
				addSteps = append(
					addSteps,
					&rkcypb.ApecsTxn_Step{
						System:        rkcypb.System_PROCESS,
						Concern:       "Player",
						Command:       rkcy.FLUSH_INSTANCE,
						Key:           rslt.Key,
						EffectiveTime: timestamppb.New(args.EffectiveTime),
					},
				)
			}
		default:
			apecs.SetStepResult(rslt, fmt.Errorf("Invalid storage command: %s", command))
			return rslt, nil
		}
	case rkcypb.System_STORAGE_SCND:
		if stgTgt.IsPrimary {
			apecs.SetStepResult(rslt, fmt.Errorf("Storage target is primary in STORAGE_SCND handler: %s, command: %s", stgTgt.Name, command))
			return rslt, nil
		}
		if direction != rkcypb.Direction_FORWARD {
			apecs.SetStepResult(rslt, fmt.Errorf("Reverse direction not supported in STORAGE_SCND handler: %s, command: %s", stgTgt.Name, command))
			return rslt, nil
		}
		switch command {
		case rkcy.CREATE:
			{
				instBytes, _, err := rkcy.ParsePayload(args.Payload)
				if err != nil {
					apecs.SetStepResult(rslt, err)
					return rslt, nil
				}
				inst := &Player{}
				err = proto.Unmarshal(instBytes, inst)
				if err != nil {
					apecs.SetStepResult(rslt, err)
					return rslt, nil
				}
				if inst.Key() == "" {
					apecs.SetStepResult(rslt, fmt.Errorf("Empty key in CREATE STORAGE_SCND"))
					return rslt, nil
				}
				_, err = handler.Create(ctx, inst, args.CmpdOffset)
				if err != nil {
					apecs.SetStepResult(rslt, err)
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
					apecs.SetStepResult(rslt, err)
					return rslt, nil
				}
				inst := &Player{}
				err = proto.Unmarshal(instBytes, inst)
				if err != nil {
					apecs.SetStepResult(rslt, err)
					return rslt, nil
				}
				rel := &PlayerRelated{}
				if relBytes != nil && len(relBytes) > 0 {
					err = proto.Unmarshal(relBytes, rel)
					if err != nil {
						apecs.SetStepResult(rslt, err)
						return rslt, nil
					}
				}
				err = handler.Update(ctx, inst, rel, args.CmpdOffset)
				if err != nil {
					apecs.SetStepResult(rslt, err)
					return rslt, nil
				}
			}
		case rkcy.DELETE:
			{
				err = handler.Delete(ctx, args.Key, args.CmpdOffset)
				if err != nil {
					apecs.SetStepResult(rslt, err)
					return rslt, nil
				}
				rslt.Key = args.Key
			}
		default:
			apecs.SetStepResult(rslt, fmt.Errorf("Invalid storage command: %s", command))
			return rslt, nil
		}
	}

	if rslt.Code == rkcypb.Code_UNDEFINED {
		rslt.Code = rkcypb.Code_OK
	}
	return rslt, addSteps
}

func (*PlayerConcernHandler) DecodeInstance(
	ctx context.Context,
	buffer []byte,
) (*rkcy.ResultProto, error) {
	resProto := &rkcy.ResultProto{
		Type: "Player",
	}

	if !rkcy.IsPackedPayload(buffer) {
		inst := &Player{}
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

		inst := &Player{}
		err = proto.Unmarshal(instBytes, inst)
		if err != nil {
			return nil, err
		}
		resProto.Instance = inst

		if relBytes != nil && len(relBytes) > 0 {
			rel := &PlayerRelated{}
			err := proto.Unmarshal(relBytes, rel)
			if err != nil {
				return nil, err
			}
			resProto.Related = rel
		} else {
			resProto.Related = &PlayerRelated{}
		}
	}

	return resProto, nil
}

func (cncHdlr *PlayerConcernHandler) DecodeArg(
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
		default:
			return nil, fmt.Errorf("ArgDecoder invalid command: %d %s", system, command)
		}
	default:
		return nil, fmt.Errorf("ArgDecoder invalid system: %s", system.String())
	}
}

func (cncHdlr *PlayerConcernHandler) DecodeResult(
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
		default:
			return nil, fmt.Errorf("ResultDecoder invalid command: %d %s", system, command)
		}
	default:
		return nil, fmt.Errorf("ResultDecoder invalid system: %s", system.String())
	}
}

func (cncHdlr *PlayerConcernHandler) DecodeRelatedRequest(
	ctx context.Context,
	relReq *rkcypb.RelatedRequest,
) (*rkcy.ResultProto, error) {
	resProto := &rkcy.ResultProto{}

	switch relReq.Field {
	case "Player":
		if relReq.Concern == "Character" {
			resProto.Type = "Character"

			instBytes, relBytes, err := rkcy.UnpackPayloads(relReq.Payload)
			if err != nil {
				return nil, err
			}
			inst := &Character{}
			err = proto.Unmarshal(instBytes, inst)
			if err != nil {
				return nil, err
			}
			resProto.Instance = inst

			if relBytes != nil {
				rel := &CharacterRelated{}
				err = proto.Unmarshal(relBytes, rel)
				if err != nil {
					return nil, err
				}
				resProto.Related = rel
			}
		} else {
			return nil, fmt.Errorf("DecodeRelatedRequest: Invalid type '%s' for field Characters, expecting 'Character'", relReq.Concern)
		}
	default:
		return nil, fmt.Errorf("DecodeRelatedRequest: Invalid field '%s''", relReq.Field)
	}
	return resProto, nil
}

// -----------------------------------------------------------------------------
// Concern Player END
// -----------------------------------------------------------------------------
