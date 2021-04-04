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

	"github.com/lachlanorr/rocketcycle/examples/rpg/storage"
)

var messageFactory = map[string]func() proto.Message{
	"player": func() proto.Message { return proto.Message(new(storage.Player)) },
}

func getResource(resourceName string, id string) (int, []byte, error) {
	path := fmt.Sprintf("/v1/%s/get/%s?pretty", resourceName, id)

	resp, err := http.Get(settings.EdgeAddr + path)
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

func cobraGetResource(cmd *cobra.Command, args []string) {
	status, body, err := getResource(args[0], args[1])
	if err != nil {
		log.Fatal().
			Err(err).
			Str("Resource", args[0]).
			Str("Id", args[1]).
			Msg("Failed to GET")
	}

	fmt.Printf("%d %s\n", status, string(body))

	if status != 200 {
		os.Exit(1)
	}
}

func createOrUpdateResource(verb string, resourceName string, msg proto.Message, fieldsToSet []string) (int, []byte, error) {
	path := fmt.Sprintf("/v1/%s/%s?pretty", resourceName, verb)

	err := reflectKeyValArgs(msg, resourceName, fieldsToSet, true)
	if err != nil {
		return 500, nil, err
	}

	content, err := protojson.Marshal(msg)
	if err != nil {
		return 500, nil, err
	}

	contentRdr := bytes.NewReader(content)
	resp, err := http.Post(settings.EdgeAddr+path, "application/json", contentRdr)
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
	msgFac, ok := messageFactory[args[0]]
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
	msgFac, ok := messageFactory[args[0]]
	if !ok {
		log.Fatal().
			Str("Resource", args[0]).
			Msg("Invalid Resource")
	}
	msg := msgFac()

	// First try to get resource, since we'll be overlaying requested field values
	statusGet, bodyGet, err := getResource(args[0], args[1])
	if err != nil {
		log.Fatal().
			Err(err).
			Str("Resource", args[0]).
			Str("Id", args[1]).
			Msg("Failed to GET")
	}
	if statusGet != 200 {
		log.Fatal().
			Err(err).
			Str("Resource", args[0]).
			Str("Id", args[1]).
			Msg("Error status from GET")
	}

	err = protojson.Unmarshal(bodyGet, msg)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("Resource", args[0]).
			Str("Id", args[1]).
			Msg("Failed to Unmarshal GET response")
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

	resp, err := http.Post(settings.EdgeAddr+path, "application/json", nil)
	if err != nil {
		return 500, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
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

func reflectKeyValArgs(msg proto.Message, resourceName string, keyVals []string, initId bool) error {
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
				return errors.New(fmt.Sprintf("Invalid field for %s resource: %s", resourceName, toks[0]))
			}
			switch kind := field.Kind(); kind {
			case protoreflect.StringKind:
				msgR.Set(field, protoreflect.ValueOf(toks[1]))
			case protoreflect.BoolKind:
				valBool, err := strconv.ParseBool(toks[1])
				if err != nil {
					return errors.New(fmt.Sprintf("Bad value for %s resource: %s=%s", resourceName, toks[0], toks[1]))
				}
				msgR.Set(field, protoreflect.ValueOf(valBool))
			case protoreflect.Int32Kind,
				protoreflect.Sint32Kind,
				protoreflect.Uint32Kind,
				protoreflect.Int64Kind,
				protoreflect.Sint64Kind,
				protoreflect.Uint64Kind:
				valInt, err := strconv.Atoi(toks[1])
				if err != nil {
					return errors.New(fmt.Sprintf("Bad value for %s resource: %s=%s", resourceName, toks[0], toks[1]))
				}
				msgR.Set(field, protoreflect.ValueOf(valInt))

			case protoreflect.Sfixed32Kind,
				protoreflect.Fixed32Kind,
				protoreflect.FloatKind,
				protoreflect.Sfixed64Kind,
				protoreflect.Fixed64Kind,
				protoreflect.DoubleKind:
				valFloat, err := strconv.ParseFloat(toks[1], 64)
				if err != nil {
					return errors.New(fmt.Sprintf("Bad value for %s resource: %s=%s", resourceName, toks[0], toks[1]))
				}
				msgR.Set(field, protoreflect.ValueOf(valFloat))
			}
		}
	}

	return nil
}
