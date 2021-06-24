// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package edge

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/lachlanorr/rocketcycle/examples/rpg/concerns"
)

type ResourceType int

const (
	ResourceType_PLAYER ResourceType = iota
	ResourceType_CHARACTER
	ResourceType_FUNDING_REQUEST
)

var messageFactory = map[ResourceType]func() proto.Message{
	ResourceType_PLAYER:          func() proto.Message { return proto.Message(new(concerns.Player)) },
	ResourceType_CHARACTER:       func() proto.Message { return proto.Message(new(concerns.Character)) },
	ResourceType_FUNDING_REQUEST: func() proto.Message { return proto.Message(new(concerns.FundingRequest)) },
}

var MessageFactory = map[string]func() proto.Message{
	"player":    messageFactory[ResourceType_PLAYER],
	"character": messageFactory[ResourceType_CHARACTER],
}

func readResource(resourceName string, id string) (int, []byte, error) {
	path := fmt.Sprintf("/v1/%s/read/%s?pretty", resourceName, id)

	resp, err := http.Get(settings.EdgeHttpAddr + path)
	if err != nil {
		return 500, nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	return resp.StatusCode, body, nil
}

func cobraReadResource(cmd *cobra.Command, args []string) {
	status, body, err := readResource(args[0], args[1])
	if err != nil {
		log.Fatal().
			Err(err).
			Str("Resource", args[0]).
			Str("Id", args[1]).
			Msg("Failed to READ")
	}

	fmt.Printf("%d %s\n", status, string(body))

	if status != 200 {
		os.Exit(1)
	}
}

func createOrUpdateResource(verb string, resourceName string, msg proto.Message, fieldsToSet []string) (int, []byte, error) {
	path := fmt.Sprintf("/v1/%s/%s?pretty", resourceName, verb)

	err := reflectKeyValArgs(msg, fieldsToSet, true)
	if err != nil {
		return 500, nil, err
	}

	content, err := protojson.Marshal(msg)
	if err != nil {
		return 500, nil, err
	}

	contentRdr := bytes.NewReader(content)
	resp, err := http.Post(settings.EdgeHttpAddr+path, "application/json", contentRdr)
	if err != nil {
		return 500, nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	return resp.StatusCode, body, nil
}

func cobraCreateResource(cmd *cobra.Command, args []string) {
	msgFac, ok := MessageFactory[args[0]]
	if !ok {
		log.Fatal().
			Str("Resource", args[0]).
			Msg("Invalid Resource")
	}
	msg := msgFac()

	status, body, err := createOrUpdateResource("create", args[0], msg, args[1:])
	if err != nil {
		log.Fatal().
			Err(err).
			Str("Resource", args[0]).
			Msg("Failed to CREATE")
	}

	fmt.Printf("%d %s\n", status, body)

	if status != 200 {
		os.Exit(1)
	}
}

func cobraUpdateResource(cmd *cobra.Command, args []string) {
	msgFac, ok := MessageFactory[args[0]]
	if !ok {
		log.Fatal().
			Str("Resource", args[0]).
			Msg("Invalid Resource")
	}
	msg := msgFac()

	// First try to read resource, since we'll be overlaying requested field values
	statusRead, bodyRead, err := readResource(args[0], args[1])
	if err != nil {
		log.Fatal().
			Err(err).
			Str("Resource", args[0]).
			Str("Id", args[1]).
			Msg("Failed to READ")
	}
	if statusRead != 200 {
		log.Fatal().
			Err(err).
			Str("Resource", args[0]).
			Str("Id", args[1]).
			Msg("Error status from READ")
	}

	err = protojson.Unmarshal(bodyRead, msg)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("Resource", args[0]).
			Str("Id", args[1]).
			Msg("Failed to Unmarshal READ response")
	}

	status, body, err := createOrUpdateResource("update", args[0], msg, args[2:])
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to UPDATE")
	}

	fmt.Printf("%d %s\n", status, string(body))

	if status != 200 {
		os.Exit(1)
	}
}

func deleteResource(resourceName string, id string) (int, error) {
	path := fmt.Sprintf("/v1/%s/delete/%s", resourceName, id)

	resp, err := http.Post(settings.EdgeHttpAddr+path, "application/json", nil)
	if err != nil {
		return 500, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func cobraFundCharacter(cmd *cobra.Command, args []string) {
	path := "/v1/character/fund?pretty"

	fr := concerns.FundingRequest{
		CharacterId: args[0],
		Currency:    &concerns.Character_Currency{},
	}

	err := reflectKeyValArgs(fr.Currency, args[1:], false)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to reflectKeyValArgs")
	}

	content, err := protojson.Marshal(&fr)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to Marshal")
	}

	contentRdr := bytes.NewReader(content)
	resp, err := http.Post(settings.EdgeHttpAddr+path, "application/json", contentRdr)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to http.Post")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to ReadAll")
	}

	fmt.Printf("%d %s\n", resp.StatusCode, string(body))

	if resp.StatusCode != 200 {
		os.Exit(1)
	}
}

func cobraDeleteResource(cmd *cobra.Command, args []string) {
	status, err := deleteResource(args[0], args[1])
	if err != nil {
		log.Fatal().
			Err(err).
			Str("Resource", args[0]).
			Str("Id", args[1]).
			Msg("Failed to DELETE")
	}

	fmt.Printf("%d\n", status)

	if status != 200 {
		os.Exit(1)
	}
}

func reflectKeyValArgs(msg proto.Message, keyVals []string, initId bool) error {
	msgR := msg.ProtoReflect()
	desc := msgR.Descriptor()
	fields := desc.Fields()

	for _, arg := range keyVals {
		toks := strings.Split(arg, "=")
		if len(toks) == 2 {
			if initId && toks[0] == "id" {
				return errors.New("'id' set for new resource instance")
			}

			field := fields.ByJSONName(toks[0])
			if field == nil {
				return errors.New(fmt.Sprintf("Invalid field: %s", toks[0]))
			}
			switch kind := field.Kind(); kind {
			case protoreflect.StringKind:
				msgR.Set(field, protoreflect.ValueOf(toks[1]))
			case protoreflect.BoolKind:
				valBool, err := strconv.ParseBool(toks[1])
				if err != nil {
					return errors.New(fmt.Sprintf("Bad value: %s=%s", toks[0], toks[1]))
				}
				msgR.Set(field, protoreflect.ValueOf(valBool))
			case protoreflect.Int32Kind:
				valInt, err := strconv.Atoi(toks[1])
				if err != nil {
					return errors.New(fmt.Sprintf("Bad value: %s=%s", toks[0], toks[1]))
				}
				msgR.Set(field, protoreflect.ValueOf(int32(valInt)))

			case protoreflect.FloatKind,
				protoreflect.DoubleKind:
				valFloat, err := strconv.ParseFloat(toks[1], 64)
				if err != nil {
					return errors.New(fmt.Sprintf("Bad value: %s=%s", toks[0], toks[1]))
				}
				msgR.Set(field, protoreflect.ValueOf(valFloat))
			}
		}
	}

	return nil
}
