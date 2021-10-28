// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package pb

import (
	"bytes"
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

// -----------------------------------------------------------------------------
// Concern Character
// -----------------------------------------------------------------------------
func init() {
	rkcy.RegisterConcernHandler(&CharacterConcernHandler{})
}

func (inst *Character) Key() string {
	return inst.Id
}

func CharacterGetRelated(instKey string, instanceStore *rkcy.InstanceStore) (*CharacterRelatedConcerns, error) {
	relCnc := &CharacterRelatedConcerns{}
	relCncBytes := instanceStore.GetRelated(instKey)
	if relCncBytes != nil {
		err := proto.Unmarshal(relCncBytes, relCnc)
		if err != nil {
			return nil, err
		}
	}
    return relCnc, nil
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
		return rkcy.NewError(rkcy.Code_INVALID_ARGUMENT, "Empty Key during PreValidateCreate for Character")
	}
	return nil
}

func (inst *Character) PreValidateUpdate(ctx context.Context, updated *Character) error {
	if updated.Key() == "" || inst.Key() != updated.Key() {
		return rkcy.NewError(rkcy.Code_INVALID_ARGUMENT, "Mismatched Keys during PreValidateUpdate for Character")
	}
	return nil
}

// LogicHandler Interface
type CharacterLogicHandler interface {
	ValidateCreate(ctx context.Context, inst *Character) (*Character, error)
	ValidateUpdate(ctx context.Context, original *Character, updated *Character) (*Character, error)

	Fund(ctx context.Context, inst *Character, relCnc *CharacterRelatedConcerns, payload *FundingRequest) (*Character, error)
	DebitFunds(ctx context.Context, inst *Character, relCnc *CharacterRelatedConcerns, payload *FundingRequest) (*Character, error)
	CreditFunds(ctx context.Context, inst *Character, relCnc *CharacterRelatedConcerns, payload *FundingRequest) (*Character, error)
}

// CrudHandler Interface
type CharacterCrudHandler interface {
	Read(ctx context.Context, key string) (*Character, *CharacterRelatedConcerns, *rkcy.CompoundOffset, error)
	Create(ctx context.Context, inst *Character, cmpdOffset *rkcy.CompoundOffset) (*Character, error)
	Update(ctx context.Context, inst *Character, relCnc *CharacterRelatedConcerns, cmpdOffset *rkcy.CompoundOffset) error
	Delete(ctx context.Context, key string, cmpdOffset *rkcy.CompoundOffset) error
}

// Concern Handler
type CharacterConcernHandler struct {
	logicHandler CharacterLogicHandler
	crudHandlers map[string]CharacterCrudHandler
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

func (cncHdlr *CharacterConcernHandler) ValidateHandlers() bool {
	if cncHdlr.logicHandler == nil {
		return false
	}
	if cncHdlr.crudHandlers == nil || len(cncHdlr.crudHandlers) == 0 {
		return false
	}
	return true
}

func (cncHdlr *CharacterConcernHandler) HandleCommand(
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
					relCnc, err := CharacterGetRelated(args.Key, instanceStore)
	                if err != nil {
	                    rslt.SetResult(err)
	                    return rslt, nil
	                }
                    relSteps, err := relCnc.RefreshRelatedStepsDUMMY(args.Payload)
	                if err != nil {
	                    rslt.SetResult(err)
	                    return rslt, nil
	                }
					addSteps = append(addSteps, relSteps...)
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
				payloadIn := &Character{}
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
				inst := &Character{}
				err = proto.Unmarshal(args.Instance, inst)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}

				payloadIn := &Character{}
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
					relCnc, err := CharacterGetRelated(args.Key, instanceStore)
	                if err != nil {
	                    rslt.SetResult(err)
	                    return rslt, nil
	                }
                    relSteps, err := relCnc.RefreshRelatedStepsDUMMY(instSer)
	                if err != nil {
	                    rslt.SetResult(err)
	                    return rslt, nil
	                }
					addSteps = append(addSteps, relSteps...)
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
				relCnc, err := CharacterGetRelated(args.Key, instanceStore)
                if err != nil {
                    rslt.SetResult(err)
                    return rslt, nil
                }
                // Handle refresh related request to see if it fulfills any of our related items
                changed, err := relCnc.HandleRequestRelatedDUMMY(relReq)
                if err != nil {
                    rslt.SetResult(err)
                    return rslt, nil
                }
   				relBytes, err := proto.Marshal(relCnc)
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
                // Send response containing our instance
				relRsp := &rkcy.RelatedResponse{
                    Concern: "Character",
					Field:   relReq.Field,
					Payload: args.Instance,
				}
				relRspBytes, err := proto.Marshal(relRsp)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				addSteps = append(
					addSteps,
					&rkcy.ApecsTxn_Step{
						System:		   rkcy.System_PROCESS,
						Concern:	   relReq.Concern,
						Command:	   rkcy.REFRESH_RELATED,
						Key:		   relReq.Key,
						Payload:	   relRspBytes,
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
				relCnc, err := CharacterGetRelated(args.Key, instanceStore)
                if err != nil {
                    rslt.SetResult(err)
                    return rslt, nil
                }
                // Handle refresh related response
                err = relCnc.HandleRefreshRelatedDUMMY(relRsp)
                if err != nil {
                    rslt.SetResult(err)
                    return rslt, nil
                }
   				relBytes, err := proto.Marshal(relCnc)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
                err = instanceStore.SetRelated(args.Key, relBytes, args.CmpdOffset)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				rslt.Payload = rkcy.PackPayloads(args.Instance, relBytes)
			}
		case "Fund":
			{
				if args.Instance == nil {
					rslt.SetResult(fmt.Errorf("No instance exists during HandleCommand"))
					return rslt, nil
				}
				inst := &Character{}
				err = proto.Unmarshal(args.Instance, inst)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				relCnc, err := CharacterGetRelated(inst.Key(), instanceStore)
                if err != nil {
                    rslt.SetResult(err)
                    return rslt, nil
                }
				payloadIn := &FundingRequest{}
				err = proto.Unmarshal(args.Payload, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				payloadOut, err := cncHdlr.logicHandler.Fund(ctx, inst, relCnc, payloadIn)
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
					rslt.Instance = instSer
				}
			}
		case "DebitFunds":
			{
				if args.Instance == nil {
					rslt.SetResult(fmt.Errorf("No instance exists during HandleCommand"))
					return rslt, nil
				}
				inst := &Character{}
				err = proto.Unmarshal(args.Instance, inst)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				relCnc, err := CharacterGetRelated(inst.Key(), instanceStore)
                if err != nil {
                    rslt.SetResult(err)
                    return rslt, nil
                }
				payloadIn := &FundingRequest{}
				err = proto.Unmarshal(args.Payload, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				payloadOut, err := cncHdlr.logicHandler.DebitFunds(ctx, inst, relCnc, payloadIn)
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
					rslt.Instance = instSer
				}
			}
		case "CreditFunds":
			{
				if args.Instance == nil {
					rslt.SetResult(fmt.Errorf("No instance exists during HandleCommand"))
					return rslt, nil
				}
				inst := &Character{}
				err = proto.Unmarshal(args.Instance, inst)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				relCnc, err := CharacterGetRelated(inst.Key(), instanceStore)
                if err != nil {
                    rslt.SetResult(err)
                    return rslt, nil
                }
				payloadIn := &FundingRequest{}
				err = proto.Unmarshal(args.Payload, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				payloadOut, err := cncHdlr.logicHandler.CreditFunds(ctx, inst, relCnc, payloadIn)
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
					rslt.Instance = instSer
				}
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
					payloadIn := &Character{}
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
                        err := fmt.Errorf("No Key set in CREATE Character")
                        errDel := handler.Delete(ctx, payloadOut.Key(), args.CmpdOffset)
                        if errDel != nil {
                            rslt.SetResult(fmt.Errorf("DELETE error cleaning up CREATE failure, orig err: '%s', DELETE err: '%s'", err.Error(), errDel.Error()))
                            return rslt, nil
                        }
						rslt.SetResult(fmt.Errorf("No Key set in CREATE Character"))
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
							System:		   rkcy.System_PROCESS,
							Concern:	   "Character",
							Command:	   rkcy.REFRESH_INSTANCE,
							Key:		   rslt.Key,
							Payload:	   rkcy.PackPayloads(rslt.Payload, nil /* should be related */),
							EffectiveTime: timestamppb.New(args.EffectiveTime),
						},
					)
                    // Request related refresh messages from our related concerns
                    reqRfshSteps, err := payloadOut.RequestRelatedStepsDUMMY(rslt.Payload)
                    if err != nil {
                        errDel := handler.Delete(ctx, payloadOut.Key(), args.CmpdOffset)
                        if errDel != nil {
                            rslt.SetResult(fmt.Errorf("DELETE error cleaning up CREATE failure, orig err: '%s', DELETE err: '%s'", err.Error(), errDel.Error()))
                            return rslt, nil
                        }
                        rslt.SetResult(err)
                        return rslt, nil
                    }
					addSteps = append(addSteps, reqRfshSteps...)
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
				var inst *Character
				var relCnc *CharacterRelatedConcerns
				inst, relCnc, rslt.CmpdOffset, err = handler.Read(ctx, args.Key)
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
				relBytes, err = proto.Marshal(relCnc)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				rslt.Payload = rkcy.PackPayloads(instBytes, relBytes)
				addSteps = append(
					addSteps,
					&rkcy.ApecsTxn_Step{
						System:		   rkcy.System_PROCESS,
						Concern:	   "Character",
						Command:	   rkcy.REFRESH_INSTANCE,
						Key:		   inst.Key(),
						Payload:	   rslt.Payload,
						EffectiveTime: timestamppb.New(args.EffectiveTime),
					},
				)
			}
		case rkcy.UPDATE:
			{
				payloadIn := &Character{}
				err = proto.Unmarshal(args.Payload, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
				relCnc, err := CharacterGetRelated(payloadIn.Key(), instanceStore)
                if err != nil {
                    rslt.SetResult(err)
                    return rslt, nil
                }
				err = handler.Update(ctx, payloadIn, relCnc, args.CmpdOffset)
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
        unpacked, err := rkcy.UnpackPayloads(buffer)
        if err != nil {
            return nil, err
        }

		inst := &Character{}
		err = proto.Unmarshal(unpacked[0], inst)
		if err != nil {
			return nil, err
		}
        resProto.Instance = inst

        if unpacked[1] != nil && len(unpacked[1]) > 0 {
    		rel := &CharacterRelatedConcerns{}
			err := proto.Unmarshal(unpacked[1], rel)
			if err != nil {
				return nil, err
			}
	        resProto.Related = rel
        }
    }

	return resProto, nil
}

func (cncHdlr *CharacterConcernHandler) DecodeArg(
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
		return nil, fmt.Errorf("ArgDecoder invalid system: %d", system)
	}
}

func (cncHdlr *CharacterConcernHandler) DecodeResult(
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
		return nil, fmt.Errorf("ResultDecoder invalid system: %d", system)
	}
}

func (cncHdlr *CharacterConcernHandler) DecodeRelatedRequest(
	ctx context.Context,
	relReq *rkcy.RelatedRequest,
) (*rkcy.ResultProto, error) {
	return &rkcy.ResultProto{
		Type: "Character Note: need to add this to codegen!!!",
	}, nil
}

func (cncHdlr *CharacterConcernHandler) DecodeRelatedResponse(
	ctx context.Context,
	relRsp *rkcy.RelatedResponse,
) (*rkcy.ResultProto, error) {
	return &rkcy.ResultProto{
		Type: "Character Note: need to add this to codegen!!!",
	}, nil
}

// -----------------------------------------------------------------------------
// Concern Character END
// -----------------------------------------------------------------------------
