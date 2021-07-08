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
func (inst *Player) Read(ctx context.Context, key string) (*rkcy.Offset, error) {
	// Read Player instance from storage system and set in inst
	// Return Offset as well, as was presented on last Create/Update

	return nil, rkcy.NewError(rkcy.Code_NOT_IMPLEMENTED, "Command Not Implemented: Player.Read")
}

func (inst *Player) Create(ctx context.Context, offset *rkcy.Offset) error {
	// Create new Player instance in the storage system, store offset as well.
	// If storage offset is less than offset argument, do not create,
	// as this is indicative of a message duplicate.

	return rkcy.NewError(rkcy.Code_NOT_IMPLEMENTED, "Command Not Implemented: Player.Create")
}

func (inst *Player) Update(ctx context.Context, offset *rkcy.Offset) error {
	// Update existsing Player instance in the storage system,
	// store offset as well.
	// If storage offset is less than offset argument, do not update,
	// as this is indicative of a message duplicate.

	return rkcy.NewError(rkcy.Code_NOT_IMPLEMENTED, "Command Not Implemented: Player.Update")
}

func (inst *Player) Delete(ctx context.Context, key string, offset *rkcy.Offset) error {
	// Delete existsing Player instance in the storage system.
	// If storage offset is less than offset argument, do not delete,
	// as this is indicative of a message duplicate.

	return rkcy.NewError(rkcy.Code_NOT_IMPLEMENTED, "Command Not Implemented: Player.Delete")
}
// -----------------------------------------------------------------------------
// STORAGE CRUD Handlers (END)
// -----------------------------------------------------------------------------


// -----------------------------------------------------------------------------
// PROCESS Standard Handlers
// -----------------------------------------------------------------------------
func (*Player) ValidateCreate(ctx context.Context, payload *Player) (*Player, error) {
	// Validate contents of Player 'payload', make any changes appropriately, and return it.

	return nil, rkcy.NewError(rkcy.Code_NOT_IMPLEMENTED, "Command Not Implemented: Player.ValidateCreate")
}

func (inst *Player) ValidateUpdate(ctx context.Context, payload *Player) (*Player, error) {
	// Validate contents of Player 'payload', make any changes, and return it.
	// 'inst' contains current instance if that is important for validation.

	return nil, rkcy.NewError(rkcy.Code_NOT_IMPLEMENTED, "Command Not Implemented: Player.ValidateUpdate")
}
// -----------------------------------------------------------------------------
// PROCESS Standard Handlers (END)
// -----------------------------------------------------------------------------


// -----------------------------------------------------------------------------
// PROCESS Command Handlers
// -----------------------------------------------------------------------------
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

func (*Player) Concern() string {
	return "Player"
}

func (inst *Player) Key() string {
	return inst.Id
}

func (inst *Player) SetKey(key string) {
	inst.Id = key
}

func init() {
	decodeInst := func(ctx context.Context, buffer []byte) (string, error) {
		pb := &Player{}
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
		"Player",
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
				case rkcy.CREATE:
					{
						if direction == rkcy.Direction_FORWARD {
							payloadIn := &Player{}
							err = proto.Unmarshal(args.Payload, payloadIn)
							if err != nil {
								rslt.SetResult(err)
								return rslt
							}
							err = payloadIn.Create(ctx, args.Offset)
							if err != nil {
								rslt.SetResult(err)
								return rslt
							}
							rslt.Offset = args.Offset // for possible delete in rollback
							rslt.Payload, err = proto.Marshal(payloadIn)
							if err != nil {
								rslt.SetResult(err)
								return rslt
							}
						} else {
							del := &Player{}
							err = del.Delete(ctx, args.Key, args.ForwardResult.Offset)
							if err != nil {
								rslt.SetResult(err)
								return rslt
							}
						}
					}
				case rkcy.READ:
					{
						if direction == rkcy.Direction_FORWARD {
							inst := &Player{}
							rslt.Offset, err = inst.Read(ctx, args.Key)
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
					}
				case rkcy.UPDATE:
					{
						if direction == rkcy.Direction_FORWARD {
							// capture orig so we can roll this back
							orig := &Player{}
							_, err := orig.Read(ctx, args.Key)
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
							err = payloadIn.Update(ctx, args.Offset)
							if err != nil {
								rslt.SetResult(err)
								return rslt
							}
							rslt.Payload, err = proto.Marshal(payloadIn)
							if err != nil {
								rslt.SetResult(err)
								return rslt
							}
							// Set original value into rslt.Instance so we can restore it in the event of a rollback
							rslt.Offset = args.Offset
							rslt.Instance, err = proto.Marshal(orig)
							if err != nil {
								rslt.SetResult(err)
								return rslt
							}
						} else {
							orig := &Player{}
							err = proto.Unmarshal(args.ForwardResult.Instance, orig)
							if err != nil {
								rslt.SetResult(err)
								return rslt
							}
							err = orig.Update(ctx, args.ForwardResult.Offset)
							if err != nil {
								rslt.SetResult(err)
								return rslt
							}
						}
					}
				case rkcy.DELETE:
					{
						if direction == rkcy.Direction_FORWARD {
							// capture orig so we can roll this back
							orig := &Player{}
							_, err := orig.Read(ctx, args.Key)
							if err != nil {
								rslt.SetResult(err)
								return rslt
							}
							del := &Player{}
							err = del.Delete(ctx, args.Key, args.Offset)
							if err != nil {
								rslt.SetResult(err)
								return rslt
							}
							// Set original value into rslt.Instance so we can restore it in the event of a rollback
							rslt.Offset = args.Offset
							rslt.Instance, err = proto.Marshal(orig)
							if err != nil {
								rslt.SetResult(err)
								return rslt
							}
						} else {
							orig := &Player{}
							err = proto.Unmarshal(args.ForwardResult.Instance, orig)
							if err != nil {
								rslt.SetResult(err)
								return rslt
							}
							err = orig.Create(ctx, args.ForwardResult.Offset)
							if err != nil {
								rslt.SetResult(err)
								return rslt
							}
						}
					}
				default:
					rslt.SetResult(fmt.Errorf("Invalid storage command: %s", command))
					return rslt
				}
			} else if system == rkcy.System_PROCESS {
				inst := &Player{}
				if args.Instance != nil {
					err = proto.Unmarshal(args.Instance, inst)
					if err != nil {
						rslt.SetResult(err)
						return rslt
					}
				}

				if inst.Key() == "" && command != rkcy.VALIDATE_CREATE {
					rslt.SetResult(fmt.Errorf("No key present during HandleCommand"))
					return rslt
				}

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
						payloadOut, err := inst.ValidateCreate(ctx, payloadIn)
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
						payloadIn := &Player{}
						err = proto.Unmarshal(args.Payload, payloadIn)
						if err != nil {
							rslt.SetResult(err)
							return rslt
						}
						payloadOut, err := inst.ValidateUpdate(ctx, payloadIn)
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
				default:
					rslt.SetResult(fmt.Errorf("Invalid process command: %s", command))
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
				case rkcy.CREATE:
					fallthrough
				case rkcy.READ:
					fallthrough
				case rkcy.UPDATE:
					return decodeInst(ctx, buffer)
				default:
					return "", fmt.Errorf("ArgDecoder invalid command: %d %s", system, command)
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
					return decodeInst(ctx, buffer)
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
				case rkcy.CREATE:
					fallthrough
				case rkcy.READ:
					fallthrough
				case rkcy.UPDATE:
					return decodeInst(ctx, buffer)
				default:
					return "", fmt.Errorf("ResultDecoder invalid command: %d %s", system, command)
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
