// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy/pb"
)

type rtApecsTxn struct {
	txn *pb.ApecsTxn
}

func newApecsTxn(responseTarget *pb.ResponseTarget, canRevert bool, steps []pb.Step) (*pb.ApecsTxn, error) {
	reqId := uuid.NewString()

	if responseTarget != nil && (responseTarget.TopicName == "" || responseTarget.Partition < 0) {
		return nil, fmt.Errorf("NewApecsTxn ReqId=%s ResponseTarget=%+v: Invalid ResponseTarget", reqId, responseTarget)
	}

	txn := pb.ApecsTxn{
		ReqId:          reqId,
		ResponseTarget: responseTarget,
		CurrentStepIdx: 0,
		Direction:      pb.Direction_FORWARD,
		CanRevert:      canRevert,
		ForwardSteps:   make([]*pb.Step, 0, len(steps)),
	}
	fwdstps := make([]*pb.Step, 0, len(steps))

	for _, stepiter := range steps {
		step := stepiter
		if step.System == pb.System_PROCESS && step.Command == pb.Command_CREATE {
			storStep := step
			storStep.System = pb.System_STORAGE
			txn.ForwardSteps = append(txn.ForwardSteps, &storStep)
		}
		fwdstps = append(fwdstps, &step)
		txn.ForwardSteps = append(txn.ForwardSteps, &step)
	}

	return &txn, nil
}

func newRtApecsTxn(txn *pb.ApecsTxn) (*rtApecsTxn, error) {
	rtxn := rtApecsTxn{
		txn: txn,
	}

	err := rtxn.validate()
	if err != nil {
		return nil, err
	}

	return &rtxn, nil
}

func (rtxn *rtApecsTxn) firstForwardStep() *pb.Step {
	return rtxn.txn.ForwardSteps[0]
}

func (rtxn *rtApecsTxn) advanceStepIdx() bool {
	canAdvance := false
	if rtxn.txn.Direction == pb.Direction_FORWARD {
		if rtxn.txn.CurrentStepIdx < int32(len(rtxn.txn.ForwardSteps)-1) {
			canAdvance = true
		}
	} else { // txn.Direction == Reverse
		if rtxn.txn.CurrentStepIdx < int32(len(rtxn.txn.ReverseSteps)-1) {
			canAdvance = true
		}
	}
	if canAdvance {
		rtxn.txn.CurrentStepIdx++
	}
	return canAdvance
}

func (rtxn *rtApecsTxn) previousStep() *pb.Step {
	if rtxn.txn.CurrentStepIdx == 0 {
		return nil
	}

	if rtxn.txn.Direction == pb.Direction_FORWARD {
		return rtxn.txn.ForwardSteps[rtxn.txn.CurrentStepIdx-1]
	} else { // txn.Direction == Reverse
		return rtxn.txn.ReverseSteps[rtxn.txn.CurrentStepIdx-1]
	}
}

func (rtxn *rtApecsTxn) currentStep() *pb.Step {
	if rtxn.txn.Direction == pb.Direction_FORWARD {
		return rtxn.txn.ForwardSteps[rtxn.txn.CurrentStepIdx]
	} else { // txn.Direction == Reverse
		return rtxn.txn.ReverseSteps[rtxn.txn.CurrentStepIdx]
	}
}

func validateSteps(reqId string, currentStepIdx int32, steps []*pb.Step, name string) error {
	if currentStepIdx >= int32(len(steps)) {
		return fmt.Errorf(
			"rtApecsTxn.validateSteps ReqId=%s CurrentStepIdx=%d len(%sSteps)=%d: CurrentStepIdx out of bounds",
			reqId,
			currentStepIdx,
			name,
			len(steps),
		)
	}
	if len(steps) == 0 {
		return fmt.Errorf(
			"rtApecsTxn.validateSteps ReqId=%s: No %s steps",
			reqId,
			name,
		)
	}

	for _, step := range steps {
		if step.System != pb.System_PROCESS && step.System != pb.System_STORAGE {
			return fmt.Errorf(
				"rtApecsTxn.validateSteps ReqId=%s System=%d: Invalid System",
				reqId,
				step.System,
			)
		}
	}
	return nil // all looks good
}

func (rtxn *rtApecsTxn) validate() error {
	if rtxn.txn == nil {
		return fmt.Errorf("Nil ApecsTxn")
	}
	if rtxn.txn.ReqId == "" {
		return fmt.Errorf("ApecsTxn with no ReqId")
	}

	if rtxn.txn.CurrentStepIdx < 0 {
		return fmt.Errorf(
			"rtApecsTxn.validate ReqId=%s CurrentStepIdx=%d: Negative CurrentStep",
			rtxn.txn.ReqId,
			rtxn.txn.CurrentStepIdx,
		)
	}
	if rtxn.txn.Direction == pb.Direction_FORWARD {
		if err := validateSteps(rtxn.txn.ReqId, rtxn.txn.CurrentStepIdx, rtxn.txn.ForwardSteps, "Forward"); err != nil {
			return err
		}
	} else if rtxn.txn.Direction == pb.Direction_REVERSE {
		if err := validateSteps(rtxn.txn.ReqId, rtxn.txn.CurrentStepIdx, rtxn.txn.ReverseSteps, "Reverse"); err != nil {
			return err
		}
	} else {
		return fmt.Errorf(
			"rtApecsTxn.validate ReqId=%s Direction=%d: Invalid Direction",
			rtxn.txn.ReqId,
			rtxn.txn.Direction,
		)
	}

	return nil // all looks good
}
