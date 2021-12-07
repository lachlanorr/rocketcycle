// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type rtApecsTxn struct {
	txn         *rkcypb.ApecsTxn
	traceParent string
}

func newApecsTxn(
	txnId string,
	assocTxn *rkcypb.AssocTxn,
	respTarget *rkcypb.TopicTarget,
	uponError rkcypb.UponError,
	steps []*rkcypb.ApecsTxn_Step,
) (*rkcypb.ApecsTxn, error) {
	if respTarget != nil && (respTarget.Topic == "" || respTarget.Partition < 0) {
		return nil, fmt.Errorf("NewApecsTxn TxnId=%s TopicTarget=%+v: Invalid TopicTarget", txnId, respTarget)
	}

	var assocTxns []*rkcypb.AssocTxn
	if assocTxn != nil {
		assocTxns = append(assocTxns, assocTxn)
	}

	txn := rkcypb.ApecsTxn{
		Id:             txnId,
		AssocTxns:      assocTxns,
		ResponseTarget: respTarget,
		CurrentStepIdx: 0,
		Direction:      rkcypb.Direction_FORWARD,
		UponError:      uponError,
		ForwardSteps:   make([]*rkcypb.ApecsTxn_Step, 0, len(steps)),
	}

	for _, step := range steps {
		if step.System == rkcypb.System_PROCESS && step.Command == CREATE {
			// Inject a refresh step so the cache gets updated if storage Create succeeds
			validateStep := &rkcypb.ApecsTxn_Step{
				System:        rkcypb.System_PROCESS,
				Concern:       step.Concern,
				Command:       VALIDATE_CREATE,
				Payload:       step.Payload,
				EffectiveTime: step.EffectiveTime,
			}
			step.System = rkcypb.System_STORAGE // switch to storage CREATE
			txn.ForwardSteps = append(txn.ForwardSteps, validateStep)
			txn.ForwardSteps = append(txn.ForwardSteps, step)
		} else if step.System == rkcypb.System_PROCESS && step.Command == UPDATE {
			validateStep := &rkcypb.ApecsTxn_Step{
				System:        rkcypb.System_PROCESS,
				Concern:       step.Concern,
				Command:       VALIDATE_UPDATE,
				Payload:       step.Payload,
				EffectiveTime: step.EffectiveTime,
			}
			txn.ForwardSteps = append(txn.ForwardSteps, validateStep)
			txn.ForwardSteps = append(txn.ForwardSteps, step)
		} else {
			txn.ForwardSteps = append(txn.ForwardSteps, step)
		}
	}

	return &txn, nil
}

func newRtApecsTxn(txn *rkcypb.ApecsTxn, traceParent string) (*rtApecsTxn, error) {
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

func ApecsTxnResult(ctx context.Context, plat Platform, txn *rkcypb.ApecsTxn) (bool, *ResultProto, *rkcypb.ApecsTxn_Step_Result) {
	step := ApecsTxnCurrentStep(txn)
	success := txn.Direction == rkcypb.Direction_FORWARD &&
		txn.CurrentStepIdx == int32(len(txn.ForwardSteps)-1) &&
		step.Result != nil &&
		step.Result.Code == rkcypb.Code_OK
	var resProto *ResultProto
	if step.Result != nil && step.Result.Payload != nil {
		var err error
		resProto, _, err = plat.ConcernHandlers().decodeResultPayload(ctx, step.Concern, step.System, step.Command, step.Result.Payload)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Failed to decodeResultPayload")
		}
	}
	return success, resProto, step.Result
}

func (rtxn *rtApecsTxn) firstForwardStep() *rkcypb.ApecsTxn_Step {
	return rtxn.txn.ForwardSteps[0]
}

func (rtxn *rtApecsTxn) insertSteps(idx int32, steps ...*rkcypb.ApecsTxn_Step) error {
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
	newSteps := make([]*rkcypb.ApecsTxn_Step, len(currSteps)+len(steps))

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

func (rtxn *rtApecsTxn) replaceStep(idx int32, step *rkcypb.ApecsTxn_Step) error {
	currSteps := rtxn.getSteps()

	if idx < 0 || idx >= int32(len(currSteps)) {
		return fmt.Errorf("Index out of range")
	}

	currSteps[idx] = step
	return nil
}

func getSteps(txn *rkcypb.ApecsTxn) []*rkcypb.ApecsTxn_Step {
	if txn.Direction == rkcypb.Direction_FORWARD {
		return txn.ForwardSteps
	} else { // txn.Direction == Reverse
		return txn.ReverseSteps
	}
}

func (rtxn *rtApecsTxn) getSteps() []*rkcypb.ApecsTxn_Step {
	return getSteps(rtxn.txn)
}

func (rtxn *rtApecsTxn) setSteps(steps []*rkcypb.ApecsTxn_Step) {
	if rtxn.txn.Direction == rkcypb.Direction_FORWARD {
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

func (rtxn *rtApecsTxn) previousStep() *rkcypb.ApecsTxn_Step {
	if rtxn.txn.CurrentStepIdx == 0 {
		return nil
	}
	steps := rtxn.getSteps()
	return steps[rtxn.txn.CurrentStepIdx-1]
}

func (rtxn *rtApecsTxn) currentStep() *rkcypb.ApecsTxn_Step {
	return ApecsTxnCurrentStep(rtxn.txn)
}

func ApecsTxnCurrentStep(txn *rkcypb.ApecsTxn) *rkcypb.ApecsTxn_Step {
	steps := getSteps(txn)
	return steps[txn.CurrentStepIdx]
}

func validateSteps(txnId string, currentStepIdx int32, steps []*rkcypb.ApecsTxn_Step, name string) error {
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
		if step.System != rkcypb.System_PROCESS && !IsStorageSystem(step.System) {
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
	if rtxn.txn.Direction == rkcypb.Direction_FORWARD {
		if err := validateSteps(rtxn.txn.Id, rtxn.txn.CurrentStepIdx, rtxn.txn.ForwardSteps, "Forward"); err != nil {
			return err
		}
	} else if rtxn.txn.Direction == rkcypb.Direction_REVERSE {
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

func TxnDirectionName(txn *rkcypb.ApecsTxn) string {
	return strings.Title(strings.ToLower(rkcypb.Direction_name[int32(txn.Direction)]))
}

func StepSystemName(step *rkcypb.ApecsTxn_Step) string {
	return strings.Title(strings.ToLower(rkcypb.System_name[int32(step.System)]))
}
