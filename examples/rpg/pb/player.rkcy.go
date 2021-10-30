// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package pb

import (
	"bytes"
	"context"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
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
	rkcy.RegisterConcernHandler(&PlayerConcernHandler{})
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
		return rkcy.NewError(rkcy.Code_INVALID_ARGUMENT, "Empty Key during PreValidateCreate for Player")
	}
	return nil
}

func (inst *Player) PreValidateUpdate(ctx context.Context, updated *Player) error {
	if updated.Key() == "" || inst.Key() != updated.Key() {
		return rkcy.NewError(rkcy.Code_INVALID_ARGUMENT, "Mismatched Keys during PreValidateUpdate for Player")
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
	Read(ctx context.Context, key string) (*Player, *PlayerRelated, *rkcy.CompoundOffset, error)
	Create(ctx context.Context, inst *Player, cmpdOffset *rkcy.CompoundOffset) (*Player, error)
	Update(ctx context.Context, inst *Player, related *PlayerRelated, cmpdOffset *rkcy.CompoundOffset) error
	Delete(ctx context.Context, key string, cmpdOffset *rkcy.CompoundOffset) error
}

// Concern Handler
type PlayerConcernHandler struct {
	logicHandler PlayerLogicHandler
	crudHandlers map[string]PlayerCrudHandler
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

func (inst *Player) RequestRelatedSteps(instBytes []byte, relBytes []byte) ([]*rkcy.ApecsTxn_Step, error) {
	// No forward related Concerns
	return nil, nil
}

func (rel *PlayerRelated) RefreshRelatedSteps(instBytes []byte, relBytes []byte) ([]*rkcy.ApecsTxn_Step, error) {
	var steps []*rkcy.ApecsTxn_Step

	payload := rkcy.PackPayloads(instBytes, relBytes)

	// Field: Characters
	for _, itm := range rel.Characters {
		relRsp := &rkcy.RelatedResponse{
			Concern: "Player",
			Field:   "Player",
			Payload: payload,
		}
		relRspBytes, err := proto.Marshal(relRsp)
		if err != nil {
			return nil, err
		}
		steps = append(steps, &rkcy.ApecsTxn_Step{
			System:  rkcy.System_PROCESS,
			Concern: "Character",
			Command: rkcy.REFRESH_RELATED,
			Key:     itm.Key(),
			Payload: relRspBytes,
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

func (rel *PlayerRelated) HandleRequestRelated(relReq *rkcy.RelatedRequest) (bool, error) {
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

func (rel *PlayerRelated) HandleRefreshRelated(relRsp *rkcy.RelatedResponse) (bool, error) {
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

func (cncHdlr *PlayerConcernHandler) HandleCommand(
	ctx context.Context,
	system rkcy.System,
	command string,
	direction rkcy.Direction,
	args *rkcy.StepArgs,
	instanceStore *rkcy.InstanceStore,
	confRdr *rkcy.ConfigRdr,
	storageType string,
) (*rkcy.ApecsTxn_Step_Result, []*rkcy.ApecsTxn_Step) {
	var err error
	rslt := &rkcy.ApecsTxn_Step_Result{}
	var addSteps []*rkcy.ApecsTxn_Step // additional steps we want to be run after us

	if direction == rkcy.Direction_REVERSE && args.ForwardResult == nil {
		rslt.SetResult(fmt.Errorf("Unable to reverse step with nil ForwardResult"))
		return rslt, nil
	}

	if system == rkcy.System_PROCESS {
		switch command {
		// process handlers
		case rkcy.CREATE:
			rslt.SetResult(fmt.Errorf("Invalid process command: %s", command))
			return rslt, nil
		case rkcy.READ:
			if direction == rkcy.Direction_FORWARD {
				related := instanceStore.GetRelated(args.Key)
				rslt.Payload = rkcy.PackPayloads(args.Instance, related)
			} else { // Direction_REVERSE
				panic("REVERSE not implemented")
			}
		case rkcy.UPDATE:
			if direction == rkcy.Direction_FORWARD {
				if !bytes.Equal(args.Payload, args.Instance) {
					rel, relBytes, err := PlayerGetRelated(args.Key, instanceStore)
					if err != nil {
						rslt.SetResult(err)
						return rslt, nil
					}
					relSteps, err := rel.RefreshRelatedSteps(args.Payload, relBytes) // dummy
					if err != nil {
						rslt.SetResult(err)
						return rslt, nil
					}
					if relSteps != nil {
						addSteps = append(addSteps, relSteps...)
					}
					rslt.Instance = args.Payload
				}
			} else { // Direction_REVERSE
				panic("REVERSE not implemented")
			}
		case rkcy.DELETE:
			if direction == rkcy.Direction_FORWARD {
				instanceStore.Remove(args.Key)
			} else { // Direction_REVERSE
				panic("REVERSE not implemented")
			}
		case rkcy.VALIDATE_CREATE:
			{
				payloadIn := &Player{}
				err = proto.Unmarshal(args.Payload, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				err = payloadIn.PreValidateCreate(ctx)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				payloadOut, err := cncHdlr.logicHandler.ValidateCreate(ctx, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				rslt.Payload, err = proto.Marshal(payloadOut)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
			}
		case rkcy.VALIDATE_UPDATE:
			{
				if args.Instance == nil {
					rslt.SetResult(fmt.Errorf("No instance exists during VALIDATE_UPDATE"))
					return rslt, nil
				}
				inst := &Player{}
				err = proto.Unmarshal(args.Instance, inst)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}

				payloadIn := &Player{}
				err = proto.Unmarshal(args.Payload, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				err = inst.PreValidateUpdate(ctx, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				payloadOut, err := cncHdlr.logicHandler.ValidateUpdate(ctx, inst, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				rslt.Payload, err = proto.Marshal(payloadOut)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}

				// compare inst to see if it has changed
				instSer, err := proto.Marshal(inst)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				if !bytes.Equal(instSer, args.Instance) {
					rel, relBytes, err := PlayerGetRelated(args.Key, instanceStore)
					if err != nil {
						rslt.SetResult(err)
						return rslt, nil
					}
					relSteps, err := rel.RefreshRelatedSteps(instSer, relBytes) // dummy
					if err != nil {
						rslt.SetResult(err)
						return rslt, nil
					}
					if relSteps != nil {
						addSteps = append(addSteps, relSteps...)
					}
					rslt.Instance = instSer
				}
			}
		case rkcy.REQUEST_RELATED:
			{
				relReq := &rkcy.RelatedRequest{}
				err := proto.Unmarshal(args.Payload, relReq)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				// Get our related concerns from instanceStore
				rel, _, err := PlayerGetRelated(args.Key, instanceStore)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				// Handle refresh related request to see if it fulfills any of our related items
				changed, err := rel.HandleRequestRelated(relReq) // dummy
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				relBytes, err := proto.Marshal(rel)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				relPayload := rkcy.PackPayloads(args.Instance, relBytes)
				if changed {
					err = instanceStore.SetRelated(args.Key, relBytes, args.CmpdOffset)
					if err != nil {
						rslt.SetResult(err)
						return rslt, nil
					}
				}
				// Send response containing our instance
				relRsp := &rkcy.RelatedResponse{
					Concern: "Player",
					Field:   relReq.Field,
					Payload: relPayload,
				}
				relRspBytes, err := proto.Marshal(relRsp)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				addSteps = append(
					addSteps,
					&rkcy.ApecsTxn_Step{
						System:        rkcy.System_PROCESS,
						Concern:       relReq.Concern,
						Command:       rkcy.REFRESH_RELATED,
						Key:           relReq.Key,
						Payload:       relRspBytes,
						EffectiveTime: timestamppb.Now(),
					},
				)
				rslt.Payload = relRspBytes
			}
		case rkcy.REFRESH_RELATED:
			{
				relRsp := &rkcy.RelatedResponse{}
				err := proto.Unmarshal(args.Payload, relRsp)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				// Get our related concerns from instanceStore
				rel, _, err := PlayerGetRelated(args.Key, instanceStore)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				// Handle refresh related response
				changed, err := rel.HandleRefreshRelated(relRsp) // dummy
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				relBytes, err := proto.Marshal(rel)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				if changed {
					err = instanceStore.SetRelated(args.Key, relBytes, args.CmpdOffset)
					if err != nil {
						rslt.SetResult(err)
						return rslt, nil
					}
				}
				rslt.Payload = rkcy.PackPayloads(args.Instance, relBytes)
			}
		default:
			rslt.SetResult(fmt.Errorf("Invalid process command: %s", command))
			return rslt, nil
		}
	} else if system == rkcy.System_STORAGE {
		// Quick out for REVERSE mode on non CREATE commands, this should never happen
		if direction == rkcy.Direction_REVERSE && command != rkcy.CREATE {
			rslt.SetResult(fmt.Errorf("Unable to reverse non CREATE storage commands"))
			return rslt, nil
		}

		handler, ok := cncHdlr.crudHandlers[storageType]
		if !ok {
			rslt.SetResult(fmt.Errorf("No CrudHandler for %s", storageType))
			return rslt, nil
		}

		switch command {
		// storage handlers
		case rkcy.CREATE:
			{
				if direction == rkcy.Direction_FORWARD {
					payloadIn := &Player{}
					err = proto.Unmarshal(args.Payload, payloadIn)
					if err != nil {
						rslt.SetResult(err)
						return rslt, nil
					}
					payloadOut, err := handler.Create(ctx, payloadIn, args.CmpdOffset)
					if err != nil {
						rslt.SetResult(err)
						return rslt, nil
					}
					// NOTE: From this point on any error handler must attempt to
					// Delete from the datastore, if this step fails, a reversal
					// step will not be run, so we must ensure we do it here.
					if payloadOut.Key() == "" {
						err := fmt.Errorf("No Key set in CREATE Player")
						errDel := handler.Delete(ctx, payloadOut.Key(), args.CmpdOffset)
						if errDel != nil {
							rslt.SetResult(fmt.Errorf("DELETE error cleaning up CREATE failure, orig err: '%s', DELETE err: '%s'", err.Error(), errDel.Error()))
							return rslt, nil
						}
						rslt.SetResult(fmt.Errorf("No Key set in CREATE Player"))
						return rslt, nil
					}
					rslt.Key = payloadOut.Key()
					rslt.CmpdOffset = args.CmpdOffset // for possible delete in rollback
					rslt.Payload, err = proto.Marshal(payloadOut)
					if err != nil {
						errDel := handler.Delete(ctx, payloadOut.Key(), args.CmpdOffset)
						if errDel != nil {
							rslt.SetResult(fmt.Errorf("DELETE error cleaning up CREATE failure, orig err: '%s', DELETE err: '%s'", err.Error(), errDel.Error()))
							return rslt, nil
						}
						rslt.SetResult(err)
						return rslt, nil
					}
					// Add a refresh step which will update the instance cache
					addSteps = append(
						addSteps,
						&rkcy.ApecsTxn_Step{
							System:        rkcy.System_PROCESS,
							Concern:       "Player",
							Command:       rkcy.REFRESH_INSTANCE,
							Key:           rslt.Key,
							Payload:       rkcy.PackPayloads(rslt.Payload, nil /* should be related */),
							EffectiveTime: timestamppb.New(args.EffectiveTime),
						},
					)
					// Request related refresh messages from our related concerns
					relSteps, err := payloadOut.RequestRelatedSteps(rslt.Payload, nil) // dummy
					if err != nil {
						errDel := handler.Delete(ctx, payloadOut.Key(), args.CmpdOffset)
						if errDel != nil {
							rslt.SetResult(fmt.Errorf("DELETE error cleaning up CREATE failure, orig err: '%s', DELETE err: '%s'", err.Error(), errDel.Error()))
							return rslt, nil
						}
						rslt.SetResult(err)
						return rslt, nil
					}
					if relSteps != nil {
						addSteps = append(addSteps, relSteps...)
					}
				} else { // Direction_REVERSE
					err = handler.Delete(ctx, args.Key, args.ForwardResult.CmpdOffset)
					if err != nil {
						rslt.SetResult(err)
						return rslt, nil
					}
				}
			}
		case rkcy.READ:
			{
				var inst *Player
				var rel *PlayerRelated
				inst, rel, rslt.CmpdOffset, err = handler.Read(ctx, args.Key)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				instBytes, err := proto.Marshal(inst)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				var relBytes []byte
				relBytes, err = proto.Marshal(rel)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				rslt.Payload = rkcy.PackPayloads(instBytes, relBytes)
				addSteps = append(
					addSteps,
					&rkcy.ApecsTxn_Step{
						System:        rkcy.System_PROCESS,
						Concern:       "Player",
						Command:       rkcy.REFRESH_INSTANCE,
						Key:           inst.Key(),
						Payload:       rslt.Payload,
						EffectiveTime: timestamppb.New(args.EffectiveTime),
					},
				)
			}
		case rkcy.UPDATE:
			{
				payloadIn := &Player{}
				err = proto.Unmarshal(args.Payload, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				rel, _, err := PlayerGetRelated(payloadIn.Key(), instanceStore)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				err = handler.Update(ctx, payloadIn, rel, args.CmpdOffset)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
			}
		case rkcy.DELETE:
			{
				err = handler.Delete(ctx, args.Key, args.CmpdOffset)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
			}
		default:
			rslt.SetResult(fmt.Errorf("Invalid storage command: %s", command))
			return rslt, nil
		}
	} else {
		rslt.SetResult(fmt.Errorf("Invalid system: %d", system))
		return rslt, nil
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
		}
	}

	return resProto, nil
}

func (cncHdlr *PlayerConcernHandler) DecodeArg(
	ctx context.Context,
	system rkcy.System,
	command string,
	buffer []byte,
) (*rkcy.ResultProto, error) {
	switch system {
	case rkcy.System_STORAGE:
		switch command {
		case rkcy.CREATE:
			fallthrough
		case rkcy.READ:
			fallthrough
		case rkcy.UPDATE:
			return cncHdlr.DecodeInstance(ctx, buffer)
		default:
			return nil, fmt.Errorf("ArgDecoder invalid command: %d %s", system, command)
		}
	case rkcy.System_PROCESS:
		switch command {
		case rkcy.VALIDATE_CREATE:
			fallthrough
		case rkcy.VALIDATE_UPDATE:
			fallthrough
		case rkcy.REFRESH_INSTANCE:
			return cncHdlr.DecodeInstance(ctx, buffer)
		case rkcy.REQUEST_RELATED:
			{
				relReq := &rkcy.RelatedRequest{}
				err := proto.Unmarshal(buffer, relReq)
				if err != nil {
					return nil, err
				}
				return &rkcy.ResultProto{Type: "RelatedRequest", Instance: relReq}, nil
			}
		case rkcy.REFRESH_RELATED:
			{
				relRsp := &rkcy.RelatedResponse{}
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
		return nil, fmt.Errorf("ArgDecoder invalid system: %d", system)
	}
}

func (cncHdlr *PlayerConcernHandler) DecodeResult(
	ctx context.Context,
	system rkcy.System,
	command string,
	buffer []byte,
) (*rkcy.ResultProto, error) {
	switch system {
	case rkcy.System_STORAGE:
		switch command {
		case rkcy.READ:
			fallthrough
		case rkcy.CREATE:
			fallthrough
		case rkcy.UPDATE:
			return cncHdlr.DecodeInstance(ctx, buffer)
		default:
			return nil, fmt.Errorf("ResultDecoder invalid command: %d %s", system, command)
		}
	case rkcy.System_PROCESS:
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
				relRsp := &rkcy.RelatedResponse{}
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
		return nil, fmt.Errorf("ResultDecoder invalid system: %d", system)
	}
}

func (cncHdlr *PlayerConcernHandler) DecodeRelatedRequest(
	ctx context.Context,
	relReq *rkcy.RelatedRequest,
) (*rkcy.ResultProto, error) {
	return &rkcy.ResultProto{
		Type: "Player Note: need to add this to codegen!!!",
	}, nil
}

func (cncHdlr *PlayerConcernHandler) DecodeRelatedResponse(
	ctx context.Context,
	relRsp *rkcy.RelatedResponse,
) (*rkcy.ResultProto, error) {
	return &rkcy.ResultProto{
		Type: "Player Note: need to add this to codegen!!!",
	}, nil
}

// -----------------------------------------------------------------------------
// Concern Player END
// -----------------------------------------------------------------------------
