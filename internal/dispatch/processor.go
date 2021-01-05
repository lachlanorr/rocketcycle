// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package dispatch

import (
	"errors"
	"fmt"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	pb "github.com/lachlanorr/rocketcycle/build/proto/process"
)

func NextStep(txn *pb.ApecsTxn) *pb.ApecsTxn_Step {
	if txn.Direction == pb.ApecsTxn_FORWARD {
		for i, _ := range txn.ForwardSteps {
			if txn.ForwardSteps[i].Status == pb.ApecsTxn_Step_PENDING {
				return txn.ForwardSteps[i]
			}
		}
	} else if txn.CanRevert { // txn.Direction == Reverse
		for i := len(txn.ForwardSteps) - 1; i >= 0; i-- {
			if txn.ForwardSteps[i].Status == pb.ApecsTxn_Step_COMPLETE {
				return txn.ForwardSteps[i]
			}
		}
	}
	return nil
}

func LastStep(txn *pb.ApecsTxn) (*pb.ApecsTxn_Step, error) {
	if txn.Direction == pb.ApecsTxn_FORWARD {
		return txn.ForwardSteps[len(txn.ForwardSteps)-1], nil
	} else { // if txn.Direction == Reverse
		return txn.ForwardSteps[0], nil
	}
}

func HasErrors(txn *pb.ApecsTxn) bool {
	for i, _ := range txn.ForwardSteps {
		if txn.ForwardSteps[i].Status == pb.ApecsTxn_Step_ERROR {
			return true
		}
	}
	return false
}

type StepHandler struct {
	Command string
	Topic   string
	Do      func(params []string) error
	Undo    func(params []string) error
}

type Processor struct {
	Handlers map[string]StepHandler

	// exactly one of these will be called per ApecsTxn processed
	Committed  func(*pb.ApecsTxn)
	RolledBack func(*pb.ApecsTxn)
	Panicked   func(*pb.ApecsTxn, error)
}

type StepRunner interface {
	HandleStep(step *pb.ApecsTxn_Step, direction pb.ApecsTxn_Dir) error
	ProcessNextStep(txn *pb.ApecsTxn) *pb.ApecsTxn_Step
}

func (proc Processor) AdvanceApecsTxn(step *pb.ApecsTxn_Step, direction pb.ApecsTxn_Dir) error {
	if handler, ok := proc.Handlers[step.Command]; ok {
		if direction == pb.ApecsTxn_FORWARD {
			return handler.Do(step.Params)
		} else {
			return handler.Undo(step.Params)
		}
	} else {
		return errors.New(fmt.Sprintf("AdvanceApecsTxn - invalid command '%s'", step.Command))
	}
}

func (proc Processor) ProcessNextStep(txn *pb.ApecsTxn) (*pb.ApecsTxn_Step, error) {
	step := NextStep(txn)

	if step == nil {
		return nil, nil
	}

	err := proc.AdvanceApecsTxn(step, txn.Direction)
	if err == nil {
		step.Status = pb.ApecsTxn_Step_COMPLETE
	} else {
		step.Errors = append(step.Errors, err.Error())
		step.Status = pb.ApecsTxn_Step_ERROR
		if txn.Direction == pb.ApecsTxn_FORWARD && txn.CanRevert {
			txn.Direction = pb.ApecsTxn_FORWARD
		} else {
			return step, err
		}
	}

	return step, nil
}

func Process() {
	// read from kafka message queue
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost",
		"group.id":          "myGroup",
		"auto.offset.reset": "earliest",
	})

	if err != nil {
		panic(err)
	}

	c.SubscribeTopics([]string{"myTopic", "^aRegex.*[Tt]opic"}, nil)

	for {
		msg, err := c.ReadMessage(-1)
		if err == nil {
			fmt.Printf("Message on %s: %s\n", msg.TopicPartition, string(msg.Value))
		} else {
			// The client will automatically try to recover from all errors.
			fmt.Printf("Consumer error: %v (%v)\n", err, msg)
		}
	}

	c.Close()

	/*
		// Check if last step
		lastStep, err := txn.LastStep()
		if err != nil {
			proc.Panicked(txn, err)
			return
		}
		if step == lastStep {
			// we're complete
			if txn.Direction == Forward {
				proc.Committed(txn)
			} else { // if txn.Direction == Reverse
				proc.RolledBack(txn)
			}
			return
		}
	*/
}
