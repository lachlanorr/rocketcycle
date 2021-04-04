// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package storage

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

var messageFactory = map[ResourceType]func() proto.Message{
	ResourceType_PLAYER:           func() proto.Message { return proto.Message(new(Player)) },
	ResourceType_CHARACTER:        func() proto.Message { return proto.Message(new(Character)) },
	ResourceType_FUNDING_REQUEST:  func() proto.Message { return proto.Message(new(FundingRequest)) },
	ResourceType_TRANSFER_REQUEST: func() proto.Message { return proto.Message(new(TransferRequest)) },
}

var MessageFactory = map[string]func() proto.Message{
	"player":    messageFactory[ResourceType_PLAYER],
	"character": messageFactory[ResourceType_CHARACTER],
}

func Unmarshal(buffer *rkcy.Buffer) (proto.Message, error) {
	if buffer == nil {
		return nil, fmt.Errorf("Unmarshal failed, nil buffer")
	}

	msgFac, ok := messageFactory[ResourceType(buffer.Type)]
	if !ok {
		return nil, fmt.Errorf("Unmarshal failed, Unknown Type: %d", buffer.Type)
	}
	msg := msgFac()

	err := proto.Unmarshal(buffer.Data, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func Marshal(resourceType int32, msg proto.Message) (*rkcy.Buffer, error) {
	var err error

	if msg == nil {
		return nil, fmt.Errorf("Marshal failed, nil msg")
	}

	ser, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return &rkcy.Buffer{
		Type: resourceType,
		Data: ser,
	}, nil
}
