// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"fmt"
	"strings"
)

type rtApecsTxn struct {
	txn         *ApecsTxn
	traceParent string
}

func newApecsTxn(traceId string, assocTraceId string, rspTgt *TopicTarget, canRevert bool, steps []*ApecsTxn_Step) (*ApecsTxn, error) {
	if rspTgt != nil && (rspTgt.TopicName == "" || rspTgt.Partition < 0) {
		return nil, fmt.Errorf("NewApecsTxn TraceId=%s TopicTarget=%+v: Invalid TopicTarget", traceId, rspTgt)
	}

	txn := ApecsTxn{
		TraceId:        traceId,
		AssocTraceId:   assocTraceId,
		ResponseTarget: rspTgt,
		CurrentStepIdx: 0,
		Direction:      Direction_FORWARD,
		CanRevert:      canRevert,
		ForwardSteps:   make([]*ApecsTxn_Step, 0, len(steps)),
	}

	for _, step := range steps {
		if step.System == System_PROCESS && step.Command == Command_CREATE {
			// Inject a refresh step so the cache gets updated in storage CREATE succeeds
			refreshStep := *step
			refreshStep.Command = Command_REFRESH
			refreshStep.System = System_PROCESS
			step.System = System_STORAGE // switch to storage CREATE
			txn.ForwardSteps = append(txn.ForwardSteps, step)
			txn.ForwardSteps = append(txn.ForwardSteps, &refreshStep)
		} else if step.System == System_PROCESS && step.Command == Command_DELETE {
			storStep := *step
			storStep.System = System_STORAGE
			txn.ForwardSteps = append(txn.ForwardSteps, step)
			txn.ForwardSteps = append(txn.ForwardSteps, &storStep)
		} else {
			txn.ForwardSteps = append(txn.ForwardSteps, step)
		}
	}

	return &txn, nil
}

func newRtApecsTxn(txn *ApecsTxn, traceParent string) (*rtApecsTxn, error) {
	if !TraceParentIsValid(traceParent) {
		panic("newRtApecsTxn invalid traceParent: " + traceParent)
	}
	rtxn := rtApecsTxn{
		txn:         txn,
		traceParent: traceParent,
	}

	err := rtxn.validate()
	if err != nil {
		return nil, err
	}

	return &rtxn, nil
}

func ApecsTxnResult(txn *ApecsTxn) (bool, *ApecsTxn_Step_Result) {
	step := ApecsTxnCurrentStep(txn)
	success := txn.Direction == Direction_FORWARD &&
		txn.CurrentStepIdx == int32(len(txn.ForwardSteps)-1) &&
		step.Result != nil &&
		step.Result.Code == Code_OK
	return success, step.Result
}

func (rtxn *rtApecsTxn) firstForwardStep() *ApecsTxn_Step {
	return rtxn.txn.ForwardSteps[0]
}

func (rtxn *rtApecsTxn) insertSteps(idx int32, steps ...*ApecsTxn_Step) error {
	currSteps := rtxn.getSteps()
	if idx >= int32(len(steps)) {
		return fmt.Errorf("Index out of range")
	}
	newSteps := make([]*ApecsTxn_Step, len(currSteps)+len(steps))

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

func getSteps(txn *ApecsTxn) []*ApecsTxn_Step {
	if txn.Direction == Direction_FORWARD {
		return txn.ForwardSteps
	} else { // txn.Direction == Reverse
		return txn.ReverseSteps
	}
}

func (rtxn *rtApecsTxn) getSteps() []*ApecsTxn_Step {
	return getSteps(rtxn.txn)
}

func (rtxn *rtApecsTxn) setSteps(steps []*ApecsTxn_Step) {
	if rtxn.txn.Direction == Direction_FORWARD {
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

func (rtxn *rtApecsTxn) previousStep() *ApecsTxn_Step {
	if rtxn.txn.CurrentStepIdx == 0 {
		return nil
	}
	steps := rtxn.getSteps()
	return steps[rtxn.txn.CurrentStepIdx-1]
}

func (rtxn *rtApecsTxn) currentStep() *ApecsTxn_Step {
	return ApecsTxnCurrentStep(rtxn.txn)
}

func ApecsTxnCurrentStep(txn *ApecsTxn) *ApecsTxn_Step {
	steps := getSteps(txn)
	return steps[txn.CurrentStepIdx]
}

func validateSteps(traceId string, currentStepIdx int32, steps []*ApecsTxn_Step, name string) error {
	if currentStepIdx >= int32(len(steps)) {
		return fmt.Errorf(
			"rtApecsTxn.validateSteps TraceId=%s CurrentStepIdx=%d len(%sSteps)=%d: CurrentStepIdx out of bounds",
			traceId,
			currentStepIdx,
			name,
			len(steps),
		)
	}
	if len(steps) == 0 {
		return fmt.Errorf(
			"rtApecsTxn.validateSteps TraceId=%s: No %s steps",
			traceId,
			name,
		)
	}

	for _, step := range steps {
		if step.System != System_PROCESS && step.System != System_STORAGE {
			return fmt.Errorf(
				"rtApecsTxn.validateSteps TraceId=%s System=%d: Invalid System",
				traceId,
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
	if rtxn.txn.TraceId == "" {
		return fmt.Errorf("ApecsTxn with no TraceId")
	}

	if rtxn.txn.CurrentStepIdx < 0 {
		return fmt.Errorf(
			"rtApecsTxn.validate TraceId=%s CurrentStepIdx=%d: Negative CurrentStep",
			rtxn.txn.TraceId,
			rtxn.txn.CurrentStepIdx,
		)
	}
	if rtxn.txn.Direction == Direction_FORWARD {
		if err := validateSteps(rtxn.txn.TraceId, rtxn.txn.CurrentStepIdx, rtxn.txn.ForwardSteps, "Forward"); err != nil {
			return err
		}
	} else if rtxn.txn.Direction == Direction_REVERSE {
		if err := validateSteps(rtxn.txn.TraceId, rtxn.txn.CurrentStepIdx, rtxn.txn.ReverseSteps, "Reverse"); err != nil {
			return err
		}
	} else {
		return fmt.Errorf(
			"rtApecsTxn.validate TraceId=%s Direction=%d: Invalid Direction",
			rtxn.txn.TraceId,
			rtxn.txn.Direction,
		)
	}

	return nil // all looks good
}

func (txn *ApecsTxn) DirectionName() string {
	return strings.Title(strings.ToLower(Direction_name[int32(txn.Direction)]))
}

func (step *ApecsTxn_Step) SystemName() string {
	return strings.Title(strings.ToLower(System_name[int32(step.System)]))
}

func (step *ApecsTxn_Step) CommandName() string {
	return Command_name[int32(step.Command)]
}
