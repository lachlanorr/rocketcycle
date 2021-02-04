// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy/pb"
)

func nextStep(txn *pb.ApecsTxn) *pb.ApecsTxn_Step {
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

func lastStep(txn *pb.ApecsTxn) (*pb.ApecsTxn_Step, error) {
	if txn.Direction == pb.ApecsTxn_FORWARD {
		return txn.ForwardSteps[len(txn.ForwardSteps)-1], nil
	} else { // if txn.Direction == Reverse
		return txn.ForwardSteps[0], nil
	}
}

func hasErrors(txn *pb.ApecsTxn) bool {
	for i, _ := range txn.ForwardSteps {
		if txn.ForwardSteps[i].Status == pb.ApecsTxn_Step_ERROR {
			return true
		}
	}
	return false
}

type stepHandler struct {
	AppName string
	Command int
	Do      func(txn *pb.ApecsTxn) error
	Undo    func(txn *pb.ApecsTxn) error
}

type processor struct {
	Handlers map[int32]stepHandler

	// exactly one of these will be called per ApecsTxn processed
	Committed  func(*pb.ApecsTxn)
	RolledBack func(*pb.ApecsTxn)
	Panicked   func(*pb.ApecsTxn, error)
}

type stepRunner interface {
	HandleStep(step *pb.ApecsTxn_Step, direction pb.ApecsTxn_Dir) error
	ProcessNextStep(txn *pb.ApecsTxn) *pb.ApecsTxn_Step
}

func (proc processor) advanceApecsTxn(step *pb.ApecsTxn_Step, direction pb.ApecsTxn_Dir) error {
	if handler, ok := proc.Handlers[step.Command]; ok {
		if direction == pb.ApecsTxn_FORWARD {
			return handler.Do(nil) // LORRTODO: fix
		} else {
			return handler.Undo(nil) // LORRTODO: fix
		}
	} else {
		return errors.New(fmt.Sprintf("AdvanceApecsTxn - invalid command '%s'", step.Command))
	}
}

func (proc processor) processNextStep(txn *pb.ApecsTxn) (*pb.ApecsTxn_Step, error) {
	step := nextStep(txn)

	if step == nil {
		return nil, nil
	}

	err := proc.advanceApecsTxn(step, txn.Direction)
	if err == nil {
		step.Status = pb.ApecsTxn_Step_COMPLETE
	} else {
		step.Errors = nil // LORRTODO: fix this //append(step.Errors, err.Error())
		step.Status = pb.ApecsTxn_Step_ERROR
		if txn.Direction == pb.ApecsTxn_FORWARD && txn.CanRevert {
			txn.Direction = pb.ApecsTxn_FORWARD
		} else {
			return step, err
		}
	}

	return step, nil
}

func process() {
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

func procCommand(cmd *cobra.Command, args []string) {
	process()
}
