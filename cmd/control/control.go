// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/gaeneco/pb"
)

// Application pb, with some convenience lookup maps
type runtimeApp struct {
	app      *pb.Application
	hash     string
	models   map[string]*pb.Application_Model
	clusters map[string]*pb.Application_Cluster
}

func checkerr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	appJsonBytes, err := ioutil.ReadFile("./application.json")
	checkerr(err)

	app := pb.Application{}

	err = protojson.Unmarshal(appJsonBytes, proto.Message(&app))
	checkerr(err)

	rtApp, err := buildRuntimeApp(&app)
	checkerr(err)

	fmt.Println(protojson.Format(proto.Message(rtApp.app)))
}

func buildRuntimeApp(app *pb.Application) (*runtimeApp, error) {
	rtApp := runtimeApp{}

	rtApp.app = app

	// calc a checksum of raw bytes for later comparisons to see if ther have been changes
	appBytes, err := proto.Marshal(proto.Message(rtApp.app))
	checkerr(err)
	md5Bytes := md5.Sum(appBytes)
	rtApp.hash = hex.EncodeToString(md5Bytes[:])

	rtApp.clusters = make(map[string]*pb.Application_Cluster)
	for _, cluster := range rtApp.app.Clusters {
		// verify clusters only appear once
		if _, ok := rtApp.clusters[cluster.Name]; ok {
			return nil, errors.New(fmt.Sprintf("Cluster '%s' appears more than once in Application '%s' definition", cluster.Name, rtApp.app.Name))
		}
		rtApp.clusters[cluster.Name] = cluster
	}

	rtApp.models = make(map[string]*pb.Application_Model)
	for _, model := range rtApp.app.Models {
		// verify models only appear once
		if _, ok := rtApp.models[model.Name]; ok {
			return nil, errors.New(fmt.Sprintf("Model '%s' appears more than once in Application '%s' definition", model.Name, rtApp.app.Name))
		}
		rtApp.models[model.Name] = model

		if model.Current == nil {
			return nil, errors.New(fmt.Sprintf("Model '%s' has no 'current' topic definition"))
		}
	}

	return &rtApp, nil
}
