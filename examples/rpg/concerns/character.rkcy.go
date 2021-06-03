// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

/*
// Implement the following functions to enable this concern:

package concerns

import (
	"context"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

// -----------------------------------------------------------------------------
// STORAGE CRUD Handlers
// -----------------------------------------------------------------------------
func (inst *Character) Read(ctx context.Context, key string) (*rkcy.Offset, error) {
    // Read Character instance from storage system and set in inst
    // Return Offset as well, as was presented on last Create/Update

    return nil, rkcy.NewError(rkcy.Code_NOT_IMPLEMENTED, "Command Not Implemented: Character.Read")
}

func (inst *Character) Create(ctx context.Context, offset *rkcy.Offset) error {
    // Create new Character instance in the storage system, store offset as well.
    // If storage offset is less than offset argument, do not create,
    // as this is indicative of a message duplicate.

    return rkcy.NewError(rkcy.Code_NOT_IMPLEMENTED, "Command Not Implemented: Character.Create")
}

func (inst *Character) Update(ctx context.Context, offset *rkcy.Offset) error {
    // Update existsing Character instance in the storage system,
    // store offset as well.
    // If storage offset is less than offset argument, do not update,
    // as this is indicative of a message duplicate.

    return rkcy.NewError(rkcy.Code_NOT_IMPLEMENTED, "Command Not Implemented: Character.Update")
}

func (inst *Character) Delete(ctx context.Context, key string, offset *rkcy.Offset) error {
    // Delete existsing Character instance in the storage system.
    // If storage offset is less than offset argument, do not delete,
    // as this is indicative of a message duplicate.

    return rkcy.NewError(rkcy.Code_NOT_IMPLEMENTED, "Command Not Implemented: Character.Delete")
}
// -----------------------------------------------------------------------------
// STORAGE CRUD Handlers (END)
// -----------------------------------------------------------------------------


// -----------------------------------------------------------------------------
// PROCESS Standard Handlers
// -----------------------------------------------------------------------------
func (*Character) ValidateCreate(ctx context.Context, payload *Character) (*Character, error) {
    // Validate contents of Character 'payload', make any changes appropriately, and return it.

    return nil, rkcy.NewError(rkcy.Code_NOT_IMPLEMENTED, "Command Not Implemented: Character.ValidateCreate")
}

func (inst *Character) ValidateUpdate(ctx context.Context, payload *Character) (*Character, error) {
    // Validate contents of Character 'payload', make any changes, and return it.
    // 'inst' contains current instance if that is important for validation.

    return nil, rkcy.NewError(rkcy.Code_NOT_IMPLEMENTED, "Command Not Implemented: Character.ValidateUpdate")
}
// -----------------------------------------------------------------------------
// PROCESS Standard Handlers (END)
// -----------------------------------------------------------------------------


// -----------------------------------------------------------------------------
// PROCESS Command Handlers
// -----------------------------------------------------------------------------
func (inst *Character) Fund(ctx context.Context, payload *FundingRequest) (*Character, error) {
    return nil, rkcy.NewError(rkcy.Code_NOT_IMPLEMENTED, "Command Not Implemented: Character.Fund")
}
// -----------------------------------------------------------------------------
// PROCESS Command Handlers (END)
// -----------------------------------------------------------------------------
*/

package concerns

import (
	"bytes"
	"context"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

func (*Character) Concern() string {
	return "Character"
}

func (inst *Character) Key() string {
	return inst.Id
}

func (inst *Character) SetKey(key string) {
	inst.Id = key
}

func init() {
	decodeInst := func(ctx context.Context, buffer []byte) (string, error) {
		pb := &Character{}
		err := proto.Unmarshal(buffer, pb)
		if err != nil {
			return "", err
		}
		decoded, err := protojson.Marshal(pb)
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	}

	rkcy.RegisterConcernHandler(
		"Character",
		// Handler
		func(ctx context.Context, system rkcy.System, command string, direction rkcy.Direction, args *rkcy.StepArgs) *rkcy.ApecsTxn_Step_Result {
			var err error
			rslt := &rkcy.ApecsTxn_Step_Result{}

			if direction == rkcy.Direction_REVERSE && args.ForwardResult == nil {
				rslt.SetResult(fmt.Errorf("Unable to reverse step with nil ForwardResult"))
				return rslt
			}

			if system == rkcy.System_STORAGE {

				switch command {
				// storage handlers
				case rkcy.CmdCreate:
					{
						if direction == rkcy.Direction_FORWARD {
							payloadIn := &Character{}
							err = proto.Unmarshal(args.Payload, payloadIn)
							rslt.SetResult(err)
							if err == nil {
								err = payloadIn.Create(ctx, args.Offset)
								rslt.SetResult(err)
								if err == nil {
									rslt.Offset = args.Offset // for possible delete in rollback
									rslt.Payload, err = proto.Marshal(payloadIn)
									rslt.SetResult(err)
								}
							}
						} else {
							del := &Character{}
							err = del.Delete(ctx, args.Key, args.ForwardResult.Offset)
							rslt.SetResult(err)
						}
					}
				case rkcy.CmdRead:
					{
						if direction == rkcy.Direction_FORWARD {
							inst := &Character{}
							rslt.Offset, err = inst.Read(ctx, args.Key)
							rslt.SetResult(err)
							if err == nil {
								rslt.Payload, err = proto.Marshal(inst)
							}
						}
					}
				case rkcy.CmdUpdate:
					{
						if direction == rkcy.Direction_FORWARD {
							// capture orig so we can roll this back
							orig := &Character{}
							_, err := orig.Read(ctx, args.Key)
							rslt.SetResult(err)
							if err == nil {
								payloadIn := &Character{}
								err = proto.Unmarshal(args.Payload, payloadIn)
								rslt.SetResult(err)
								if err == nil {
									err = payloadIn.Update(ctx, args.Offset)
									rslt.SetResult(err)
									if err == nil {
										rslt.Payload, err = proto.Marshal(payloadIn)
										if err == nil {
											// Set original value into rslt.Instance so we can restore it in the event of a rollback
											rslt.Offset = args.Offset
											rslt.Instance, err = proto.Marshal(orig)
											rslt.SetResult(err)
										}
									}
								}
							}
						} else {
							orig := &Character{}
							err = proto.Unmarshal(args.ForwardResult.Instance, orig)
							rslt.SetResult(err)
							if err == nil {
								err = orig.Update(ctx, args.ForwardResult.Offset)
								rslt.SetResult(err)
							}
						}
					}
				case rkcy.CmdDelete:
					{
						if direction == rkcy.Direction_FORWARD {
							// capture orig so we can roll this back
							orig := &Character{}
							_, err := orig.Read(ctx, args.Key)
							rslt.SetResult(err)
							if err == nil {
								del := &Character{}
								err = del.Delete(ctx, args.Key, args.Offset)
								rslt.SetResult(err)
								if err == nil {
									// Set original value into rslt.Instance so we can restore it in the event of a rollback
									rslt.Offset = args.Offset
									rslt.Instance, err = proto.Marshal(orig)
									rslt.SetResult(err)
								}
							}
						} else {
							orig := &Character{}
							err = proto.Unmarshal(args.ForwardResult.Instance, orig)
							rslt.SetResult(err)
							if err == nil {
								err = orig.Create(ctx, args.ForwardResult.Offset)
								rslt.SetResult(err)
							}
						}
					}
				default:
					rslt.SetResult(fmt.Errorf("Invalid storage command: %s", command))
					return rslt
				}
			} else if system == rkcy.System_PROCESS {
				inst := &Character{}
				if args.Instance != nil {
					err = proto.Unmarshal(args.Instance, inst)
					if err != nil {
						rslt.SetResult(err)
						return rslt
					}
				}

				if inst.Key() == "" && command != rkcy.CmdValidateCreate {
					rslt.SetResult(fmt.Errorf("No key present during HandleCommand"))
					return rslt
				}

				switch command {
				// process handlers
				case rkcy.CmdValidateCreate:
					{
						payloadIn := &Character{}
						err = proto.Unmarshal(args.Payload, payloadIn)
						rslt.SetResult(err)
						if err == nil {
							payloadOut, err := inst.ValidateCreate(ctx, payloadIn)
							rslt.SetResult(err)
							if err == nil {
								rslt.Payload, err = proto.Marshal(payloadOut)
								rslt.SetResult(err)
							}
						}
					}
				case rkcy.CmdValidateUpdate:
					{
						payloadIn := &Character{}
						err = proto.Unmarshal(args.Payload, payloadIn)
						rslt.SetResult(err)
						if err == nil {
							payloadOut, err := inst.ValidateUpdate(ctx, payloadIn)
							rslt.SetResult(err)
							if err == nil {
								rslt.Payload, err = proto.Marshal(payloadOut)
								rslt.SetResult(err)
							}
						}
					}
				case "Fund":
					{
						payloadIn := &FundingRequest{}
						err = proto.Unmarshal(args.Payload, payloadIn)
						rslt.SetResult(err)
						if err == nil {
							payloadOut, err := inst.Fund(ctx, payloadIn)
							rslt.SetResult(err)
							if err == nil {
								rslt.Payload, err = proto.Marshal(payloadOut)
								rslt.SetResult(err)
							}
						}
					}
				default:
					rslt.SetResult(fmt.Errorf("Invalid process command: %s", command))
					return rslt
				}

				// compare inst to see if it has changed
				instSer, err := proto.Marshal(inst)
				rslt.SetResult(err)
				if err == nil {
					if !bytes.Equal(instSer, args.Instance) {
						rslt.Instance = instSer
					}
				}
			} else {
				rslt.SetResult(fmt.Errorf("Invalid system: %d", system))
				return rslt
			}

			return rslt
		},
		// InstanceDecoder
		func(ctx context.Context, buffer []byte) (string, error) {
			return decodeInst(ctx, buffer)
		},
		// ArgDecoder
		func(ctx context.Context, system rkcy.System, command string, buffer []byte) (string, error) {
			switch system {
			case rkcy.System_STORAGE:
				switch command {
				case rkcy.CmdCreate:
					fallthrough
				case rkcy.CmdRead:
					fallthrough
				case rkcy.CmdUpdate:
					return decodeInst(ctx, buffer)
				default:
					return "", fmt.Errorf("ArgDecoder invalid command: %d %s", system, command)
				}
			case rkcy.System_PROCESS:
				switch command {
				case rkcy.CmdRefresh:
					fallthrough
				case rkcy.CmdRead:
					fallthrough
				case rkcy.CmdValidateCreate:
					fallthrough
				case rkcy.CmdValidateUpdate:
					return decodeInst(ctx, buffer)
				case "Fund":
					{
						pb := &FundingRequest{}
						err := proto.Unmarshal(buffer, pb)
						if err != nil {
							return "", err
						}
						decoded, err := protojson.Marshal(pb)
						if err != nil {
							return "", err
						}
						return string(decoded), nil
					}
				default:
					return "", fmt.Errorf("ArgDecoder invalid command: %d %s", system, command)
				}
			default:
				return "", fmt.Errorf("ArgDecoder invalid system: %d", system)
			}
		},
		// ResultDecoder
		func(ctx context.Context, system rkcy.System, command string, buffer []byte) (string, error) {
			switch system {
			case rkcy.System_STORAGE:
				switch command {
				case rkcy.CmdCreate:
					fallthrough
				case rkcy.CmdRead:
					fallthrough
				case rkcy.CmdUpdate:
					return decodeInst(ctx, buffer)
				default:
					return "", fmt.Errorf("ResultDecoder invalid command: %d %s", system, command)
				}
			case rkcy.System_PROCESS:
				switch command {
				case rkcy.CmdRead:
					fallthrough
				case rkcy.CmdRefresh:
					fallthrough
				case rkcy.CmdValidateCreate:
					fallthrough
				case rkcy.CmdValidateUpdate:
					return decodeInst(ctx, buffer)
				case "Fund":
					return decodeInst(ctx, buffer)
				default:
					return "", fmt.Errorf("ResultDecoder invalid command: %d %s", system, command)
				}
			default:
				return "", fmt.Errorf("ResultDecoder invalid system: %d", system)
			}
		},
	)
}
