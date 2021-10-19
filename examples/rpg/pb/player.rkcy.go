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

// TODO: Config stuff for Limits

// -----------------------------------------------------------------------------
// Concern Player
// -----------------------------------------------------------------------------
func init() {
	rkcy.RegisterConcernHandler(PlayerConcernHandler{})
}

type PlayerConcernHandler struct{}

func (PlayerConcernHandler) ConcernName() string {
	return "Player"
}

func (PlayerConcernHandler) HandleCommand(
	ctx context.Context,
	system rkcy.System,
	command string,
	direction rkcy.Direction,
	args *rkcy.StepArgs,
) *rkcy.ApecsTxn_Step_Result {
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
					err = payloadIn.Create(ctx, args.CmpdOffset)
					if err != nil {
						rslt.SetResult(err)
						return rslt
					}
					rslt.CmpdOffset = args.CmpdOffset // for possible delete in rollback
					rslt.Payload, err = proto.Marshal(payloadIn)
					if err != nil {
						rslt.SetResult(err)
						return rslt
					}
				} else {
					del := &Player{}
					err = del.Delete(ctx, args.Key, args.ForwardResult.CmpdOffset)
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
					rslt.CmpdOffset, err = inst.Read(ctx, args.Key)
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
					err = payloadIn.Update(ctx, args.CmpdOffset)
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
					rslt.CmpdOffset = args.CmpdOffset
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
					err = orig.Update(ctx, args.ForwardResult.CmpdOffset)
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
					err = del.Delete(ctx, args.Key, args.CmpdOffset)
					if err != nil {
						rslt.SetResult(err)
						return rslt
					}
					// Set original value into rslt.Instance so we can restore it in the event of a rollback
					rslt.CmpdOffset = args.CmpdOffset
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
					err = orig.Create(ctx, args.ForwardResult.CmpdOffset)
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

		if inst.Id == "" && command != rkcy.VALIDATE_CREATE {
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
}

func (PlayerConcernHandler) DecodeInstance(
	ctx context.Context,
	buffer []byte,
) (string, error) {
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

func (cncHandler PlayerConcernHandler) DecodeArg(
	ctx context.Context,
	system rkcy.System,
	command string,
	buffer []byte,
) (string, error) {
	switch system {
	case rkcy.System_STORAGE:
		switch command {
		case rkcy.CREATE:
			fallthrough
		case rkcy.READ:
			fallthrough
		case rkcy.UPDATE:
			return cncHandler.DecodeInstance(ctx, buffer)
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
			return cncHandler.DecodeInstance(ctx, buffer)
		default:
			return "", fmt.Errorf("ArgDecoder invalid command: %d %s", system, command)
		}
	default:
		return "", fmt.Errorf("ArgDecoder invalid system: %d", system)
	}
}

func (cncHandler PlayerConcernHandler) DecodeResult(
	ctx context.Context,
	system rkcy.System,
	command string,
	buffer []byte,
) (string, error) {
	switch system {
	case rkcy.System_STORAGE:
		switch command {
		case rkcy.CREATE:
			fallthrough
		case rkcy.READ:
			fallthrough
		case rkcy.UPDATE:
			return cncHandler.DecodeInstance(ctx, buffer)
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
			return cncHandler.DecodeInstance(ctx, buffer)
		default:
			return "", fmt.Errorf("ResultDecoder invalid command: %d %s", system, command)
		}
	default:
		return "", fmt.Errorf("ResultDecoder invalid system: %d", system)
	}
}
// -----------------------------------------------------------------------------
// Concern Player END
// -----------------------------------------------------------------------------
