// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

type rtApecsTxn struct {
	txn         *ApecsTxn
	traceParent string
}

func newApecsTxn(
	txnId string,
	assocTxn *AssocTxn,
	rspTgt *TopicTarget,
	uponError UponError,
	steps []*ApecsTxn_Step,
) (*ApecsTxn, error) {
	if rspTgt != nil && (rspTgt.Topic == "" || rspTgt.Partition < 0) {
		return nil, fmt.Errorf("NewApecsTxn TxnId=%s TopicTarget=%+v: Invalid TopicTarget", txnId, rspTgt)
	}

	var assocTxns []*AssocTxn
	if assocTxn != nil {
		assocTxns = append(assocTxns, assocTxn)
	}

	txn := ApecsTxn{
		Id:             txnId,
		AssocTxns:      assocTxns,
		ResponseTarget: rspTgt,
		CurrentStepIdx: 0,
		Direction:      Direction_FORWARD,
		UponError:      uponError,
		ForwardSteps:   make([]*ApecsTxn_Step, 0, len(steps)),
	}

	for _, step := range steps {
		if step.System == System_PROCESS && step.Command == CREATE {
			// Inject a refresh step so the cache gets updated if storage Create succeeds
			validateStep := &ApecsTxn_Step{
				System:        System_PROCESS,
				Concern:       step.Concern,
				Command:       VALIDATE_CREATE,
				Payload:       step.Payload,
				EffectiveTime: step.EffectiveTime,
			}
			step.System = System_STORAGE // switch to storage CREATE
			txn.ForwardSteps = append(txn.ForwardSteps, validateStep)
			txn.ForwardSteps = append(txn.ForwardSteps, step)
		} else if step.System == System_PROCESS && step.Command == UPDATE {
			validateStep := &ApecsTxn_Step{
				System:        System_PROCESS,
				Concern:       step.Concern,
				Command:       VALIDATE_UPDATE,
				Payload:       step.Payload,
				EffectiveTime: step.EffectiveTime,
			}
			step.System = System_STORAGE // switch to storage UPDATE
			txn.ForwardSteps = append(txn.ForwardSteps, validateStep)
			txn.ForwardSteps = append(txn.ForwardSteps, step)
		} else if step.System == System_PROCESS && step.Command == DELETE {
			step.System = System_STORAGE // switch to storage DELETE
			txn.ForwardSteps = append(txn.ForwardSteps, step)
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

func ApecsTxnResult(ctx context.Context, txn *ApecsTxn) (bool, *ResultProto, *ApecsTxn_Step_Result) {
	step := ApecsTxnCurrentStep(txn)
	success := txn.Direction == Direction_FORWARD &&
		txn.CurrentStepIdx == int32(len(txn.ForwardSteps)-1) &&
		step.Result != nil &&
		step.Result.Code == Code_OK
	var resProto *ResultProto
	if step.Result != nil && step.Result.Payload != nil {
		var err error
		resProto, _, err = decodeResultPayload(ctx, step.Concern, step.System, step.Command, step.Result.Payload)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Failed to decodeResultPayload")
		}
	}
	return success, resProto, step.Result
}

func (rtxn *rtApecsTxn) firstForwardStep() *ApecsTxn_Step {
	return rtxn.txn.ForwardSteps[0]
}

func (rtxn *rtApecsTxn) insertSteps(idx int32, steps ...*ApecsTxn_Step) error {
	currSteps := rtxn.getSteps()

	if idx == int32(len(currSteps)) {
		// put at the end
		for _, step := range steps {
			currSteps = append(currSteps, step)
		}
		rtxn.setSteps(currSteps)
		return nil
	} else if idx > int32(len(currSteps)) {
		return fmt.Errorf("Index out of range")
	}

	// put in the middle, so make a new slice
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

func (rtxn *rtApecsTxn) replaceStep(idx int32, step *ApecsTxn_Step) error {
	currSteps := rtxn.getSteps()

	if idx < 0 || idx >= int32(len(currSteps)) {
		return fmt.Errorf("Index out of range")
	}

	currSteps[idx] = step
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

func validateSteps(txnId string, currentStepIdx int32, steps []*ApecsTxn_Step, name string) error {
	if currentStepIdx >= int32(len(steps)) {
		return fmt.Errorf(
			"rtApecsTxn.validateSteps TxnId=%s CurrentStepIdx=%d len(%sSteps)=%d: CurrentStepIdx out of bounds",
			txnId,
			currentStepIdx,
			name,
			len(steps),
		)
	}
	if len(steps) == 0 {
		return fmt.Errorf(
			"rtApecsTxn.validateSteps TxnId=%s: No %s steps",
			txnId,
			name,
		)
	}

	for _, step := range steps {
		if step.System != System_PROCESS && !IsStorageSystem(step.System) {
			return fmt.Errorf(
				"rtApecsTxn.validateSteps TxnId=%s System=%s: Invalid System",
				txnId,
				step.System.String(),
			)
		}
	}
	return nil // all looks good
}

func (rtxn *rtApecsTxn) validate() error {
	if rtxn.txn == nil {
		return fmt.Errorf("Nil ApecsTxn")
	}
	if rtxn.txn.Id == "" {
		return fmt.Errorf("ApecsTxn with no Id")
	}

	if rtxn.txn.CurrentStepIdx < 0 {
		return fmt.Errorf(
			"rtApecsTxn.validate TxnId=%s CurrentStepIdx=%d: Negative CurrentStep",
			rtxn.txn.Id,
			rtxn.txn.CurrentStepIdx,
		)
	}
	if rtxn.txn.Direction == Direction_FORWARD {
		if err := validateSteps(rtxn.txn.Id, rtxn.txn.CurrentStepIdx, rtxn.txn.ForwardSteps, "Forward"); err != nil {
			return err
		}
	} else if rtxn.txn.Direction == Direction_REVERSE {
		if err := validateSteps(rtxn.txn.Id, rtxn.txn.CurrentStepIdx, rtxn.txn.ReverseSteps, "Reverse"); err != nil {
			return err
		}
	} else {
		return fmt.Errorf(
			"rtApecsTxn.validate TxnId=%s Direction=%d: Invalid Direction",
			rtxn.txn.Id,
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
