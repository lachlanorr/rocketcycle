// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"fmt"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy/pb"
)

type rtApecsTxn struct {
	txn *pb.ApecsTxn
}

func newApecsTxn(reqId string, rspTgt *pb.ResponseTarget, canRevert bool, steps []pb.Step) (*pb.ApecsTxn, error) {
	if rspTgt != nil && (rspTgt.TopicName == "" || rspTgt.Partition < 0) {
		return nil, fmt.Errorf("NewApecsTxn ReqId=%s ResponseTarget=%+v: Invalid ResponseTarget", reqId, rspTgt)
	}

	txn := pb.ApecsTxn{
		ReqId:          reqId,
		ResponseTarget: rspTgt,
		CurrentStepIdx: 0,
		Direction:      pb.Direction_FORWARD,
		CanRevert:      canRevert,
		ForwardSteps:   make([]*pb.Step, 0, len(steps)),
	}

	for _, stepiter := range steps {
		step := stepiter
		if step.System == pb.System_PROCESS && step.Command == pb.Command_CREATE {
			// Inject a refresh step so the cache gets updated in storage CREATE succeeds
			refreshStep := step
			refreshStep.Command = pb.Command_REFRESH
			refreshStep.System = pb.System_PROCESS
			step.System = pb.System_STORAGE // switch to storage CREATE
			txn.ForwardSteps = append(txn.ForwardSteps, &step)
			txn.ForwardSteps = append(txn.ForwardSteps, &refreshStep)
		} else if step.System == pb.System_PROCESS && step.Command == pb.Command_DELETE {
			storStep := step
			storStep.System = pb.System_STORAGE
			txn.ForwardSteps = append(txn.ForwardSteps, &step)
			txn.ForwardSteps = append(txn.ForwardSteps, &storStep)
		} else {
			txn.ForwardSteps = append(txn.ForwardSteps, &step)
		}
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

func ApecsTxnResult(txn *pb.ApecsTxn) (bool, *pb.Step_Result) {
	step := ApecsTxnCurrentStep(txn)
	success := txn.Direction == pb.Direction_FORWARD &&
		txn.CurrentStepIdx == int32(len(txn.ForwardSteps)-1) &&
		step.Result != nil &&
		step.Result.Code == pb.Code_OK
	return success, step.Result
}

func (rtxn *rtApecsTxn) firstForwardStep() *pb.Step {
	return rtxn.txn.ForwardSteps[0]
}

func (rtxn *rtApecsTxn) insertSteps(idx int32, steps ...*pb.Step) error {
	currSteps := rtxn.getSteps()
	if idx >= int32(len(steps)) {
		return fmt.Errorf("Index out of range")
	}
	newSteps := make([]*pb.Step, len(currSteps)+len(steps))

	newIdx := int32(0)
	for currIdx, _ := range currSteps {
		if newIdx == idx {
			for stepIdx, _ := range steps {
				newSteps[newIdx] = steps[stepIdx]
				newIdx++
			}
		}
		newSteps[newIdx] = currSteps[currIdx]
		newIdx++
	}
	rtxn.setSteps(newSteps)
	return nil
}

func getSteps(txn *pb.ApecsTxn) []*pb.Step {
	if txn.Direction == pb.Direction_FORWARD {
		return txn.ForwardSteps
	} else { // txn.Direction == Reverse
		return txn.ReverseSteps
	}
}

func (rtxn *rtApecsTxn) getSteps() []*pb.Step {
	return getSteps(rtxn.txn)
}

func (rtxn *rtApecsTxn) setSteps(steps []*pb.Step) {
	if rtxn.txn.Direction == pb.Direction_FORWARD {
		rtxn.txn.ForwardSteps = steps
	} else { // txn.Direction == Reverse
		rtxn.txn.ReverseSteps = steps
	}
}

func (rtxn *rtApecsTxn) canAdvance() bool {
	steps := rtxn.getSteps()
	return rtxn.txn.CurrentStepIdx < int32(len(steps)-1)
}

func (rtxn *rtApecsTxn) advanceStepIdx() bool {
	if rtxn.canAdvance() {
		rtxn.txn.CurrentStepIdx++
		return true
	}
	return false
}

func (rtxn *rtApecsTxn) previousStep() *pb.Step {
	if rtxn.txn.CurrentStepIdx == 0 {
		return nil
	}
	steps := rtxn.getSteps()
	return steps[rtxn.txn.CurrentStepIdx-1]
}

func (rtxn *rtApecsTxn) currentStep() *pb.Step {
	return ApecsTxnCurrentStep(rtxn.txn)
}

func ApecsTxnCurrentStep(txn *pb.ApecsTxn) *pb.Step {
	steps := getSteps(txn)
	return steps[txn.CurrentStepIdx]
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
