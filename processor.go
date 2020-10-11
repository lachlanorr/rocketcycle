//go:generate protoc --go_out=. --go_opt=paths=source_relative pb/txn.proto

package main

import (
	"errors"
	"fmt"

	"github.com/lachlanorr/gaeneco/pb"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

func NextStep(txn *pb.Txn) *pb.Txn_Step {
	if txn.Direction == pb.Txn_FORWARD {
		for i, _ := range txn.Steps {
			if txn.Steps[i].Status == pb.Txn_Step_PENDING {
				return txn.Steps[i]
			}
		}
	} else if txn.CanRollback { // txn.Direction == Reverse
		for i := len(txn.Steps) - 1; i >= 0; i-- {
			if txn.Steps[i].Status == pb.Txn_Step_COMPLETE {
				return txn.Steps[i]
			}
		}
	}
	return nil
}

func LastStep(txn *pb.Txn) (*pb.Txn_Step, error) {
	if txn.Direction == pb.Txn_FORWARD {
		return txn.Steps[len(txn.Steps)-1], nil
	} else { // if txn.Direction == Reverse
		return txn.Steps[0], nil
	}
}

func HasErrors(txn *pb.Txn) bool {
	for i, _ := range txn.Steps {
		if txn.Steps[i].Status == pb.Txn_Step_ERROR {
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

	// exactly one of these will be called per Txn processed
	Committed  func(*pb.Txn)
	RolledBack func(*pb.Txn)
	Panicked   func(*pb.Txn, error)
}

type StepRunner interface {
	HandleStep(step *pb.Txn_Step, direction pb.Txn_Dir) error
	ProcessNextStep(txn *pb.Txn) *pb.Txn_Step
}

func (proc Processor) AdvanceTxn(step *pb.Txn_Step, direction pb.Txn_Dir) error {
	if handler, ok := proc.Handlers[step.Command]; ok {
		if direction == pb.Txn_FORWARD {
			return handler.Do(step.Params)
		} else {
			return handler.Undo(step.Params)
		}
	} else {
		return errors.New(fmt.Sprintf("AdvanceTxn - invalid command '%s'", step.Command))
	}
}

func (proc Processor) ProcessNextStep(txn *pb.Txn) (*pb.Txn_Step, error) {
	step := NextStep(txn)

	if step == nil {
		return nil, nil
	}

	err := proc.AdvanceTxn(step, txn.Direction)
	if err == nil {
		step.Status = pb.Txn_Step_COMPLETE
	} else {
		step.Errors = append(step.Errors, err.Error())
		step.Status = pb.Txn_Step_ERROR
		if txn.Direction == pb.Txn_FORWARD && txn.CanRollback {
			txn.Direction = pb.Txn_FORWARD
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
