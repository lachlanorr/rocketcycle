// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package pb

import (
	"bytes"
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

// TODO: Config stuff for Limits

// -----------------------------------------------------------------------------
// Concern Player
// -----------------------------------------------------------------------------
func init() {
	rkcy.RegisterConcernHandler(&PlayerConcernHandler{})
}

// helper routines on protobuf
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
	Read(ctx context.Context, key string) (*Player, *rkcy.CompoundOffset, error)
	Create(ctx context.Context, inst *Player, cmpdOffset *rkcy.CompoundOffset) (*Player, error)
	Update(ctx context.Context, inst *Player, cmpdOffset *rkcy.CompoundOffset) error
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
	storageSystem string,
) *rkcy.ApecsTxn_Step_Result {
	var err error
	rslt := &rkcy.ApecsTxn_Step_Result{}

	if direction == rkcy.Direction_REVERSE && args.ForwardResult == nil {
		rslt.SetResult(fmt.Errorf("Unable to reverse step with nil ForwardResult"))
		return rslt
	}

	if system == rkcy.System_PROCESS {
		switch command {
		// process handlers
		case rkcy.VALIDATE_CREATE:
			{
				payloadIn := &Player{}
				err = proto.Unmarshal(args.Payload, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
				err = payloadIn.PreValidateCreate(ctx)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
				payloadOut, err := cncHdlr.processCmds.ValidateCreate(ctx, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
				rslt.Payload, err = proto.Marshal(payloadOut)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
			}
		case rkcy.VALIDATE_UPDATE:
			{
				instBytes := instanceStore.GetInstance(args.Key)
				if instBytes == nil {
					rslt.SetResult(fmt.Errorf("No instance exists during VALIDATE_UPDATE"))
					return rslt
				}
				inst := &Player{}
				err = proto.Unmarshal(instBytes, inst)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}

				payloadIn := &Player{}
				err = proto.Unmarshal(args.Payload, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
				err = inst.PreValidateUpdate(ctx, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
				payloadOut, err := cncHdlr.processCmds.ValidateUpdate(ctx, inst, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
				rslt.Payload, err = proto.Marshal(payloadOut)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}

				// compare inst to see if it has changed
				instSer, err := proto.Marshal(inst)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
				if !bytes.Equal(instSer, args.Instance) {
					rslt.Instance = instSer
				}
			}
		default:
			rslt.SetResult(fmt.Errorf("Invalid process command: %s", command))
			return rslt
		}
	} else if system == rkcy.System_STORAGE {
		// Quick out for REVERSE mode on non CREATE commands, this should never happen
		if direction == rkcy.Direction_REVERSE && command != rkcy.CREATE {
			rslt.SetResult(fmt.Errorf("Unable to reverse non CREATE storage commands"))
			return rslt
		}

		cmds, ok := cncHdlr.storageCmds[storageSystem]
		if !ok {
			rslt.SetResult(fmt.Errorf("No storage commands for %s", storageSystem))
			return rslt
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
						return rslt
					}
					payloadOut, err := cmds.Create(ctx, payloadIn, args.CmpdOffset)
					if err != nil {
						rslt.SetResult(err)
						return rslt
					}
					if payloadOut.Key() == "" {
						rslt.SetResult(fmt.Errorf("No Key set in CREATE Player"))
						return rslt
					}
					rslt.Key = payloadOut.Key()
					rslt.CmpdOffset = args.CmpdOffset // for possible delete in rollback
					rslt.Payload, err = proto.Marshal(payloadOut)
					if err != nil {
						rslt.SetResult(err)
						return rslt
					}
				} else { // Direction_REVERSE
					err = cmds.Delete(ctx, args.Key, args.ForwardResult.CmpdOffset)
					if err != nil {
						rslt.SetResult(err)
						return rslt
					}
				}
			}
		case rkcy.READ:
			{
				inst := &Player{}
				inst, rslt.CmpdOffset, err = cmds.Read(ctx, args.Key)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
				rslt.Payload, err = proto.Marshal(inst)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
			}
		case rkcy.UPDATE:
			{
				payloadIn := &Player{}
				err = proto.Unmarshal(args.Payload, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
				err = cmds.Update(ctx, payloadIn, args.CmpdOffset)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
			}
		case rkcy.DELETE:
			{
				err = cmds.Delete(ctx, args.Key, args.CmpdOffset)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
			}
		default:
			rslt.SetResult(fmt.Errorf("Invalid storage command: %s", command))
			return rslt
		}
	} else {
		rslt.SetResult(fmt.Errorf("Invalid system: %d", system))
		return rslt
	}

	return rslt
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

func (cncHdlr *PlayerConcernHandler) DecodeArg(
	ctx context.Context,
	system rkcy.System,
	command string,
	buffer []byte,
) (proto.Message, error) {
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
		case rkcy.REFRESH:
			fallthrough
		case rkcy.READ:
			fallthrough
		case rkcy.VALIDATE_CREATE:
			fallthrough
		case rkcy.VALIDATE_UPDATE:
			return cncHdlr.DecodeInstance(ctx, buffer)
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
) (proto.Message, error) {
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
			return nil, fmt.Errorf("ResultDecoder invalid command: %d %s", system, command)
		}
	case rkcy.System_PROCESS:
		switch command {
		case rkcy.READ:
			fallthrough
		case rkcy.REFRESH:
			fallthrough
		case rkcy.VALIDATE_CREATE:
			fallthrough
		case rkcy.VALIDATE_UPDATE:
			return cncHdlr.DecodeInstance(ctx, buffer)
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
