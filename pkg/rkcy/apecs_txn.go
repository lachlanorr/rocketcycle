// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type Txn struct {
	Revert RevertType
	Steps  []Step
}

type Step struct {
	Concern       string
	Command       string
	Key           string
	Payload       proto.Message
	EffectiveTime time.Time
}

type RevertType int

const (
	REVERTABLE     RevertType = 0
	NON_REVERTABLE            = 1
)

func ValidateTxn(txn *Txn) error {
	if txn == nil {
		return fmt.Errorf("Nil txn")
	}
	if txn.Steps == nil || len(txn.Steps) == 0 {
		return fmt.Errorf("No steps in txn")
	}

	for _, s := range txn.Steps {
		if IsTxnProhibitedCommand(s.Command) {
			return fmt.Errorf("Invalid step command: %s", s.Command)
		}
	}

	return nil
}

type RtApecsTxn struct {
	Txn         *rkcypb.ApecsTxn
	TraceParent string
}

func NewApecsTxn(
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

func NewRtApecsTxn(txn *rkcypb.ApecsTxn, traceParent string) (*RtApecsTxn, error) {
	if !TraceParentIsValid(traceParent) {
		panic("NewRtApecsTxn invalid traceParent: " + traceParent)
	}
	rtxn := RtApecsTxn{
		Txn:         txn,
		TraceParent: traceParent,
	}

	err := rtxn.Validate()
	if err != nil {
		return nil, err
	}

	return &rtxn, nil
}

func ApecsTxnResult(
	ctx context.Context,
	cncHdlrs ConcernHandlers,
	txn *rkcypb.ApecsTxn,
) (bool, *ResultProto, *rkcypb.ApecsTxn_Step_Result) {
	step := ApecsTxnCurrentStep(txn)
	success := txn.Direction == rkcypb.Direction_FORWARD &&
		txn.CurrentStepIdx == int32(len(txn.ForwardSteps)-1) &&
		step.Result != nil &&
		step.Result.Code == rkcypb.Code_OK
	var resProto *ResultProto
	if step.Result != nil && step.Result.Payload != nil {
		var err error
		resProto, _, err = cncHdlrs.DecodeResultPayload(ctx, step.Concern, step.System, step.Command, step.Result.Payload)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Failed to DecodeResultPayload")
		}
	}
	return success, resProto, step.Result
}

func (rtxn *RtApecsTxn) FirstForwardStep() *rkcypb.ApecsTxn_Step {
	return rtxn.Txn.ForwardSteps[0]
}

func (rtxn *RtApecsTxn) InsertSteps(idx int32, steps ...*rkcypb.ApecsTxn_Step) error {
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

func (rtxn *RtApecsTxn) ReplaceStep(idx int32, step *rkcypb.ApecsTxn_Step) error {
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

func (rtxn *RtApecsTxn) getSteps() []*rkcypb.ApecsTxn_Step {
	return getSteps(rtxn.Txn)
}

func (rtxn *RtApecsTxn) setSteps(steps []*rkcypb.ApecsTxn_Step) {
	if rtxn.Txn.Direction == rkcypb.Direction_FORWARD {
		rtxn.Txn.ForwardSteps = steps
	} else { // txn.Direction == Reverse
		rtxn.Txn.ReverseSteps = steps
	}
}

func (rtxn *RtApecsTxn) CanAdvance() bool {
	steps := rtxn.getSteps()
	return rtxn.Txn.CurrentStepIdx < int32(len(steps)-1)
}

func (rtxn *RtApecsTxn) AdvanceStepIdx() bool {
	if rtxn.CanAdvance() {
		rtxn.Txn.CurrentStepIdx++
		return true
	}
	return false
}

func (rtxn *RtApecsTxn) PreviousStep() *rkcypb.ApecsTxn_Step {
	if rtxn.Txn.CurrentStepIdx == 0 {
		return nil
	}
	steps := rtxn.getSteps()
	return steps[rtxn.Txn.CurrentStepIdx-1]
}

func (rtxn *RtApecsTxn) CurrentStep() *rkcypb.ApecsTxn_Step {
	return ApecsTxnCurrentStep(rtxn.Txn)
}

func ApecsTxnCurrentStep(txn *rkcypb.ApecsTxn) *rkcypb.ApecsTxn_Step {
	steps := getSteps(txn)
	return steps[txn.CurrentStepIdx]
}

func validateSteps(txnId string, currentStepIdx int32, steps []*rkcypb.ApecsTxn_Step, name string) error {
	if currentStepIdx >= int32(len(steps)) {
		return fmt.Errorf(
			"RtApecsTxn.validateSteps TxnId=%s CurrentStepIdx=%d len(%sSteps)=%d: CurrentStepIdx out of bounds",
			txnId,
			currentStepIdx,
			name,
			len(steps),
		)
	}
	if len(steps) == 0 {
		return fmt.Errorf(
			"RtApecsTxn.validateSteps TxnId=%s: No %s steps",
			txnId,
			name,
		)
	}

	for _, step := range steps {
		if step.System != rkcypb.System_PROCESS && !IsStorageSystem(step.System) {
			return fmt.Errorf(
				"RtApecsTxn.validateSteps TxnId=%s System=%s: Invalid System",
				txnId,
				step.System.String(),
			)
		}
	}
	return nil // all looks good
}

func (rtxn *RtApecsTxn) Validate() error {
	if rtxn.Txn == nil {
		return fmt.Errorf("Nil ApecsTxn")
	}
	if rtxn.Txn.Id == "" {
		return fmt.Errorf("ApecsTxn with no Id")
	}

	if rtxn.Txn.CurrentStepIdx < 0 {
		return fmt.Errorf(
			"RtApecsTxn.validate TxnId=%s CurrentStepIdx=%d: Negative CurrentStep",
			rtxn.Txn.Id,
			rtxn.Txn.CurrentStepIdx,
		)
	}
	if rtxn.Txn.Direction == rkcypb.Direction_FORWARD {
		if err := validateSteps(rtxn.Txn.Id, rtxn.Txn.CurrentStepIdx, rtxn.Txn.ForwardSteps, "Forward"); err != nil {
			return err
		}
	} else if rtxn.Txn.Direction == rkcypb.Direction_REVERSE {
		if err := validateSteps(rtxn.Txn.Id, rtxn.Txn.CurrentStepIdx, rtxn.Txn.ReverseSteps, "Reverse"); err != nil {
			return err
		}
	} else {
		return fmt.Errorf(
			"RtApecsTxn.validate TxnId=%s Direction=%d: Invalid Direction",
			rtxn.Txn.Id,
			rtxn.Txn.Direction,
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

func IsKeylessStep(step *rkcypb.ApecsTxn_Step) bool {
	return (step.System == rkcypb.System_STORAGE && step.Command == CREATE) ||
		(step.System == rkcypb.System_PROCESS && step.Command == VALIDATE_CREATE)
}

func IsInstanceStoreStep(step *rkcypb.ApecsTxn_Step) bool {
	return step.System == rkcypb.System_PROCESS &&
		(step.Command == REFRESH_INSTANCE || step.Command == FLUSH_INSTANCE)
}
