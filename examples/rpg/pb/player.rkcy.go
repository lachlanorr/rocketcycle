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

// StorageCommands Interface
type PlayerStorageCommands interface {
	Read(ctx context.Context, key string) (*Player, *PlayerRelatedConcerns, *rkcy.CompoundOffset, error)
	Create(ctx context.Context, inst *Player, cmpdOffset *rkcy.CompoundOffset) (*Player, error)
	Update(ctx context.Context, inst *Player, relCnc *PlayerRelatedConcerns, cmpdOffset *rkcy.CompoundOffset) error
	Delete(ctx context.Context, key string, cmpdOffset *rkcy.CompoundOffset) error
}

// ProcessCommands Interface
type PlayerProcessCommands interface {
	ValidateCreate(ctx context.Context, inst *Player) (*Player, error)
	ValidateUpdate(ctx context.Context, original *Player, updated *Player) (*Player, error)
}

// Concern Handler
type PlayerConcernHandler struct {
	storageCmds map[string]PlayerStorageCommands
	processCmds PlayerProcessCommands
}

func (*PlayerConcernHandler) ConcernName() string {
	return "Player"
}

func (cncHdlr *PlayerConcernHandler) SetStorageCommands(storageSystem string, commands interface{}) error {
	cmds, ok := commands.(PlayerStorageCommands)
	if !ok {
		return fmt.Errorf("Invalid interface for PlayerConcernHandler.SetStorageCommands: %T", commands)
	}
	if cncHdlr.storageCmds == nil {
		cncHdlr.storageCmds = make(map[string]PlayerStorageCommands)
	}
	_, ok = cncHdlr.storageCmds[storageSystem]
	if ok {
		return fmt.Errorf("StorageCommands already registered for %s/PlayerConcernHandler", storageSystem)
	}
	cncHdlr.storageCmds[storageSystem] = cmds
	return nil
}

func (cncHdlr *PlayerConcernHandler) SetProcessCommands(commands interface{}) error {
	if cncHdlr.processCmds != nil {
		return fmt.Errorf("ProcessCommands already registered for PlayerConcernHandler")
	}
	var ok bool
	cncHdlr.processCmds, ok = commands.(PlayerProcessCommands)
	if !ok {
		return fmt.Errorf("Invalid interface for PlayerConcernHandler.SetProcessCommands: %T", commands)
	}
	return nil
}

func (cncHdlr *PlayerConcernHandler) ValidateCommands() bool {
	if cncHdlr.processCmds == nil {
		return false
	}
	if cncHdlr.storageCmds == nil || len(cncHdlr.storageCmds) == 0 {
		return false
	}
	return true
}

func (cncHdlr *PlayerConcernHandler) HandleCommand(
	ctx context.Context,
	system rkcy.System,
	command string,
	direction rkcy.Direction,
	args *rkcy.StepArgs,
	instanceStore *rkcy.InstanceStore,
	confRdr *rkcy.ConfigRdr,
	storageSystem string,
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
		case rkcy.UPDATE:
			if direction == rkcy.Direction_FORWARD {
				rslt.Instance = args.Payload
			} else { // Direction_REVERSE
				panic("REVERSE not implemented")
			}
		case rkcy.READ:
			if direction == rkcy.Direction_FORWARD {
				related := instanceStore.GetRelated(args.Key)
				rslt.Payload = rkcy.PackPayloads(args.Instance, related)
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
				payloadOut, err := cncHdlr.processCmds.ValidateCreate(ctx, payloadIn)
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
				payloadOut, err := cncHdlr.processCmds.ValidateUpdate(ctx, inst, payloadIn)
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

		cmds, ok := cncHdlr.storageCmds[storageSystem]
		if !ok {
			rslt.SetResult(fmt.Errorf("No storage commands for %s", storageSystem))
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
					payloadOut, err := cmds.Create(ctx, payloadIn, args.CmpdOffset)
					if err != nil {
						rslt.SetResult(err)
						return rslt, nil
					}
					if payloadOut.Key() == "" {
						rslt.SetResult(fmt.Errorf("No Key set in CREATE Player"))
						return rslt, nil
					}
					rslt.Key = payloadOut.Key()
					rslt.CmpdOffset = args.CmpdOffset // for possible delete in rollback
					rslt.Payload, err = proto.Marshal(payloadOut)
					if err != nil {
						rslt.SetResult(err)
						return rslt, nil
					}
					addSteps = append(
						addSteps,
						&rkcy.ApecsTxn_Step{
							System:	 rkcy.System_PROCESS,
							Concern: "Player",
							Command: rkcy.REFRESH_INSTANCE,
							Key:	 rslt.Key,
							Payload: rkcy.PackPayloads(rslt.Payload, nil /* should be related */),
						},
					)
				} else { // Direction_REVERSE
					err = cmds.Delete(ctx, args.Key, args.ForwardResult.CmpdOffset)
					if err != nil {
						rslt.SetResult(err)
						return rslt, nil
					}
				}
			}
		case rkcy.READ:
			{
				var inst *Player
				var relCnc *PlayerRelatedConcerns
				inst, relCnc, rslt.CmpdOffset, err = cmds.Read(ctx, args.Key)
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
						System:	 rkcy.System_PROCESS,
						Concern: "Player",
						Command: rkcy.REFRESH_INSTANCE,
						Key:	 inst.Key(),
						Payload: rslt.Payload,
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
				relCncBytes := instanceStore.GetRelated(args.Key)
				relCnc := &PlayerRelatedConcerns{}
				if relCncBytes != nil {
					err = proto.Unmarshal(relCncBytes, relCnc)
				}
				err = cmds.Update(ctx, payloadIn, relCnc, args.CmpdOffset)
				if err != nil {
					rslt.SetResult(err)
					return rslt, nil
				}
			}
		case rkcy.DELETE:
			{
				err = cmds.Delete(ctx, args.Key, args.CmpdOffset)
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
) (proto.Message, error) {
	pb := &Player{}
	err := proto.Unmarshal(buffer, pb)
	if err != nil {
		return nil, err
	}
	return pb, nil
}

func (*PlayerConcernHandler) DecodeRelated(
	ctx context.Context,
	buffer []byte,
) (proto.Message, error) {
	pb := &PlayerRelatedConcerns{}
	err := proto.Unmarshal(buffer, pb)
	if err != nil {
		return nil, err
	}
	return pb, nil
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
			{
				dec, err := cncHdlr.DecodeInstance(ctx, buffer)
				if err != nil {
					return nil, err
				}
				return &rkcy.ResultProto{Type: "Player", Instance: dec}, nil
			}
		default:
			return nil, fmt.Errorf("ArgDecoder invalid command: %d %s", system, command)
		}
	case rkcy.System_PROCESS:
		switch command {
		case rkcy.REFRESH_INSTANCE:
			{
				unpacked, err := rkcy.UnpackPayloads(buffer)
				if err != nil {
					return nil, err
				}
				instMsg, err := cncHdlr.DecodeInstance(ctx, unpacked[0])
				if err != nil {
					return nil, err
				}
				relatedMsg, err := cncHdlr.DecodeRelated(ctx, unpacked[1])
				if err != nil {
					return nil, err
				}
				return &rkcy.ResultProto{Type: "Player", Instance: instMsg, Related: relatedMsg}, nil
			}
		case rkcy.READ:
			return &rkcy.ResultProto{Type: "Player"}, nil
		case rkcy.VALIDATE_CREATE:
			fallthrough
		case rkcy.VALIDATE_UPDATE:
			{
				dec, err := cncHdlr.DecodeInstance(ctx, buffer)
				if err != nil {
					return nil, err
				}
				return &rkcy.ResultProto{Type: "Player", Instance: dec}, nil
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
			{
				unpacked, err := rkcy.UnpackPayloads(buffer)
				if err != nil {
					return nil, err
				}
				instMsg, err := cncHdlr.DecodeInstance(ctx, unpacked[0])
				if err != nil {
					return nil, err
				}
				relatedMsg, err := cncHdlr.DecodeRelated(ctx, unpacked[1])
				if err != nil {
					return nil, err
				}
				return &rkcy.ResultProto{Type: "Player", Instance: instMsg, Related: relatedMsg}, nil
			}
		case rkcy.CREATE:
			fallthrough
		case rkcy.UPDATE:
			{
				dec, err := cncHdlr.DecodeInstance(ctx, buffer)
				if err != nil {
					return nil, err
				}
				return &rkcy.ResultProto{Type: "Player", Instance: dec}, nil
			}
		default:
			return nil, fmt.Errorf("ResultDecoder invalid command: %d %s", system, command)
		}
	case rkcy.System_PROCESS:
		switch command {
		case rkcy.READ:
			fallthrough
		case rkcy.REFRESH_INSTANCE:
			{
				unpacked, err := rkcy.UnpackPayloads(buffer)
				if err != nil {
					return nil, err
				}
				instMsg, err := cncHdlr.DecodeInstance(ctx, unpacked[0])
				if err != nil {
					return nil, err
				}
				relatedMsg, err := cncHdlr.DecodeRelated(ctx, unpacked[1])
				if err != nil {
					return nil, err
				}
				return &rkcy.ResultProto{Type: "Player", Instance: instMsg, Related: relatedMsg}, nil
			}
		case rkcy.VALIDATE_CREATE:
			fallthrough
		case rkcy.VALIDATE_UPDATE:
			{
				dec, err := cncHdlr.DecodeInstance(ctx, buffer)
				if err != nil {
					return nil, err
				}
                return &rkcy.ResultProto{Type: "Player", Instance: dec}, nil
			}
		default:
			return nil, fmt.Errorf("ResultDecoder invalid command: %d %s", system, command)
		}
	default:
		return nil, fmt.Errorf("ResultDecoder invalid system: %d", system)
	}
}

// -----------------------------------------------------------------------------
// Concern Player END
// -----------------------------------------------------------------------------
