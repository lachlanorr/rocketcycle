// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package portal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

func ClientReadPlatform(httpAddr string) {
	path := "/v1/platform/read?pretty"

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(httpAddr + path)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to READ")
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

func ClientReadConfig(httpAddr string) {
	path := "/v1/config/read"

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(httpAddr + path)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to READ")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to ReadAll")
		return
	}

	confRsp := &rkcypb.ConfigReadResponse{}
	err = protojson.Unmarshal(body, confRsp)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to Unmarshal ConfigResponse")
		return
	}

	var prettyJson bytes.Buffer
	err = json.Indent(&prettyJson, []byte(confRsp.ConfigJson), "", "  ")
	if err != nil {
		slog.Fatal().
			Err(err).
			Str("Json", confRsp.ConfigJson).
			Msg("Failed to prettify json")
		return
	}

	fmt.Printf("%s\n", string(prettyJson.Bytes()))
}

func ClientReadProducers(httpAddr string) {
	path := "/v1/producers/read?pretty"

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(httpAddr + path)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to READ")
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

func ClientDecodeInstance(httpAddr string, concern string, payload64 string) {
	path := "/v1/instance/decode"
	slog := log.With().
		Str("Path", path).
		Logger()

	var err error

	rpcArgs := rkcypb.DecodeInstanceArgs{
		Concern:   concern,
		Payload64: payload64,
	}

	rpcArgsSer, err := protojson.Marshal(&rpcArgs)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to marshal rpcArgs")
	}

	contentRdr := bytes.NewReader(rpcArgsSer)
	resp, err := http.Post(httpAddr+path, "application/json", contentRdr)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to DECODE")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to ReadAll")
	}

	decodeRsp := rkcypb.DecodeResponse{}
	err = protojson.Unmarshal(body, &decodeRsp)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to Unmarshal DecodeResponse")
	}

	fmt.Printf("%s\n", decodeRsp.Instance)
	if decodeRsp.Related != "" {
		fmt.Printf("\nRelated:\n%s\n", decodeRsp.Related)
	}
}

func ClientCancelTxn(grpcAddr string, txnId string) {
	conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatal().
			Err(err).
			Str("GrpcAddr", grpcAddr).
			Msg("Failed to grpc.Dial")
	}
	defer conn.Close()
	client := rkcypb.NewPortalServiceClient(conn)

	cancelTxnReq := &rkcypb.CancelTxnRequest{
		TxnId: txnId,
	}

	_, err = client.CancelTxn(context.Background(), cancelTxnReq)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("CancelTxn error")
	}
}
