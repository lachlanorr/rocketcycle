// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	storage_pb "github.com/lachlanorr/rocketcycle/build/proto/storage"
)

func rcedgeGetResource(cmd *cobra.Command, args []string) {
	path := fmt.Sprintf("/v1/%s/get/%s?pretty", args[0], args[1])

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(edgeAddr + path)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to GET")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to ReadAll")
	}

	fmt.Println(string(body))
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
	if initId {
		field := fields.ByJSONName("id")
		if field == nil || field.Kind() != protoreflect.StringKind {
			return errors.New(fmt.Sprintf("No 'id' string field for %s resource", resourceName))
		}
		msgR.Set(field, protoreflect.ValueOf(uuid.NewString()))
	}

	return nil
}

var messageFactory = map[string]func() proto.Message{
	"player": func() proto.Message { return proto.Message(new(storage_pb.Player)) },
}

func rcedgeCreateResource(cmd *cobra.Command, args []string) {
	path := fmt.Sprintf("/v1/%s/create", args[0])

	slog := log.With().
		Str("Path", path).
		Logger()

	resourceName := args[0]
	msgFac, ok := messageFactory[resourceName]
	if !ok {
		slog.Fatal().
			Msgf("Invalid resource: %s", resourceName)
	}
	msg := msgFac()
	err := reflectKeyValArgs(msg, resourceName, args[1:], true)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to reflect key values")
	}

	content, err := protojson.Marshal(msg)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to marshal json payload")
	}

	contentRdr := bytes.NewReader(content)
	resp, err := http.Post(edgeAddr+path, "application/json", contentRdr)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to POST")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to ReadAll")
	}

	if len(body) > 0 {
		fmt.Println(string(body))
	}
}
