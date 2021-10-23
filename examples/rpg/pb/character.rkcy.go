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

// -----------------------------------------------------------------------------
// Concern Character
// -----------------------------------------------------------------------------
func init() {
	rkcy.RegisterConcernHandler(&CharacterConcernHandler{})
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

// StorageCommands Interface
type CharacterStorageCommands interface {
	Read(ctx context.Context, key string) (*Character, *CharacterRelatedConcerns, *rkcy.CompoundOffset, error)
	Create(ctx context.Context, inst *Character, cmpdOffset *rkcy.CompoundOffset) (*Character, error)
	Update(ctx context.Context, inst *Character, relCnc *CharacterRelatedConcerns, cmpdOffset *rkcy.CompoundOffset) error
	Delete(ctx context.Context, key string, cmpdOffset *rkcy.CompoundOffset) error
}

// ProcessCommands Interface
type CharacterProcessCommands interface {
	ValidateCreate(ctx context.Context, inst *Character) (*Character, error)
	ValidateUpdate(ctx context.Context, original *Character, updated *Character) (*Character, error)

	Fund(ctx context.Context, inst *Character, relCnc *CharacterRelatedConcerns, payload *FundingRequest) (*Character, error)
	DebitFunds(ctx context.Context, inst *Character, relCnc *CharacterRelatedConcerns, payload *FundingRequest) (*Character, error)
	CreditFunds(ctx context.Context, inst *Character, relCnc *CharacterRelatedConcerns, payload *FundingRequest) (*Character, error)
}

// Concern Handler
type CharacterConcernHandler struct {
	storageCmds map[string]CharacterStorageCommands
	processCmds CharacterProcessCommands
}

func (*CharacterConcernHandler) ConcernName() string {
	return "Character"
}

func (cncHdlr *CharacterConcernHandler) SetStorageCommands(storageSystem string, commands interface{}) error {
	cmds, ok := commands.(CharacterStorageCommands)
	if !ok {
		return fmt.Errorf("Invalid interface for CharacterConcernHandler.SetStorageCommands: %T", commands)
	}
	if cncHdlr.storageCmds == nil {
		cncHdlr.storageCmds = make(map[string]CharacterStorageCommands)
	}
	_, ok = cncHdlr.storageCmds[storageSystem]
	if ok {
		return fmt.Errorf("StorageCommands already registered for %s/CharacterConcernHandler", storageSystem)
	}
	cncHdlr.storageCmds[storageSystem] = cmds
	return nil
}

func (cncHdlr *CharacterConcernHandler) SetProcessCommands(commands interface{}) error {
	if cncHdlr.processCmds != nil {
		return fmt.Errorf("ProcessCommands already registered for CharacterConcernHandler")
	}
	var ok bool
	cncHdlr.processCmds, ok = commands.(CharacterProcessCommands)
	if !ok {
		return fmt.Errorf("Invalid interface for CharacterConcernHandler.SetProcessCommands: %T", commands)
	}
	return nil
}

func (cncHdlr *CharacterConcernHandler) ValidateCommands() bool {
	if cncHdlr.processCmds == nil {
		return false
	}
	if cncHdlr.storageCmds == nil || len(cncHdlr.storageCmds) == 0 {
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
				payloadIn := &Character{}
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
				if args.Instance == nil {
					rslt.SetResult(fmt.Errorf("No instance exists during VALIDATE_UPDATE"))
					return rslt
				}
				inst := &Character{}
				err = proto.Unmarshal(args.Instance, inst)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}

				payloadIn := &Character{}
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
		case "Fund":
			{
				if args.Instance == nil {
					rslt.SetResult(fmt.Errorf("No instance exists during HandleCommand"))
					return rslt
				}
				inst := &Character{}
				err = proto.Unmarshal(args.Instance, inst)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
				var relCnc *CharacterRelatedConcerns
				relCncBytes := instanceStore.GetRelated(args.Key)
				if relCncBytes != nil {
					err = proto.Unmarshal(relCncBytes, relCnc)
				}
				payloadIn := &FundingRequest{}
				err = proto.Unmarshal(args.Payload, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
				payloadOut, err := cncHdlr.processCmds.Fund(ctx, inst, relCnc, payloadIn)
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
		case "DebitFunds":
			{
				if args.Instance == nil {
					rslt.SetResult(fmt.Errorf("No instance exists during HandleCommand"))
					return rslt
				}
				inst := &Character{}
				err = proto.Unmarshal(args.Instance, inst)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
				var relCnc *CharacterRelatedConcerns
				relCncBytes := instanceStore.GetRelated(args.Key)
				if relCncBytes != nil {
					err = proto.Unmarshal(relCncBytes, relCnc)
				}
				payloadIn := &FundingRequest{}
				err = proto.Unmarshal(args.Payload, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
				payloadOut, err := cncHdlr.processCmds.DebitFunds(ctx, inst, relCnc, payloadIn)
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
		case "CreditFunds":
			{
				if args.Instance == nil {
					rslt.SetResult(fmt.Errorf("No instance exists during HandleCommand"))
					return rslt
				}
				inst := &Character{}
				err = proto.Unmarshal(args.Instance, inst)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
				var relCnc *CharacterRelatedConcerns
				relCncBytes := instanceStore.GetRelated(args.Key)
				if relCncBytes != nil {
					err = proto.Unmarshal(relCncBytes, relCnc)
				}
				payloadIn := &FundingRequest{}
				err = proto.Unmarshal(args.Payload, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
				payloadOut, err := cncHdlr.processCmds.CreditFunds(ctx, inst, relCnc, payloadIn)
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
					payloadIn := &Character{}
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
						rslt.SetResult(fmt.Errorf("No Key set in CREATE Character"))
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
				var inst *Character
				var relCnc *CharacterRelatedConcerns
				inst, relCnc, rslt.CmpdOffset, err = cmds.Read(ctx, args.Key)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
				rslt.Payload, err = proto.Marshal(inst)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
				rslt.Related, err = proto.Marshal(relCnc)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
			}
		case rkcy.UPDATE:
			{
				payloadIn := &Character{}
				err = proto.Unmarshal(args.Payload, payloadIn)
				if err != nil {
					rslt.SetResult(err)
					return rslt
				}
				relCncBytes := instanceStore.GetRelated(args.Key)
				relCnc := &CharacterRelatedConcerns{}
				if relCncBytes != nil {
					err = proto.Unmarshal(relCncBytes, relCnc)
				}
				err = cmds.Update(ctx, payloadIn, relCnc, args.CmpdOffset)
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

func (*CharacterConcernHandler) DecodeInstance(
	ctx context.Context,
	buffer []byte,
) (proto.Message, error) {
	pb := &Character{}
	err := proto.Unmarshal(buffer, pb)
	if err != nil {
		return nil, err
	}
	return pb, nil
}

func (cncHdlr *CharacterConcernHandler) DecodeArg(
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
		case "Fund":
			{
				pb := &FundingRequest{}
				err := proto.Unmarshal(buffer, pb)
				if err != nil {
					return nil, err
				}
				return pb, nil
			}
		case "DebitFunds":
			{
				pb := &FundingRequest{}
				err := proto.Unmarshal(buffer, pb)
				if err != nil {
					return nil, err
				}
				return pb, nil
			}
		case "CreditFunds":
			{
				pb := &FundingRequest{}
				err := proto.Unmarshal(buffer, pb)
				if err != nil {
					return nil, err
				}
				return pb, nil
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
		case "Fund":
			return cncHdlr.DecodeInstance(ctx, buffer)
		case "DebitFunds":
			return cncHdlr.DecodeInstance(ctx, buffer)
		case "CreditFunds":
			return cncHdlr.DecodeInstance(ctx, buffer)
		default:
			return nil, fmt.Errorf("ResultDecoder invalid command: %d %s", system, command)
		}
	default:
		return nil, fmt.Errorf("ResultDecoder invalid system: %d", system)
	}
}

// -----------------------------------------------------------------------------
// Concern Character END
// -----------------------------------------------------------------------------
