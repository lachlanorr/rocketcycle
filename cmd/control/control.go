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
	fmt.Println("--------------------------------")
	fmt.Println(rtApp.app.String())
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
	for idx, cluster := range rtApp.app.Clusters {
		if cluster.Name == "" {
			return nil, fmt.Errorf("Cluster %d missing name field", idx)
		}
		if cluster.BootstrapServers == "" {
			return nil, fmt.Errorf("Cluster '%s' missing bootstrap_servers field", cluster.Name)
		}
		// verify clusters only appear once
		if _, ok := rtApp.clusters[cluster.Name]; ok {
			return nil, fmt.Errorf("Cluster '%s' appears more than once in Application '%s' definition", cluster.Name, rtApp.app.Name)
		}
		rtApp.clusters[cluster.Name] = cluster
	}

	rtApp.models = make(map[string]*pb.Application_Model)
	for idx, model := range rtApp.app.Models {
		if model.Name == "" {
			return nil, fmt.Errorf("Model %d missing name field", idx)
		}

		if model.Control == nil {
			return nil, fmt.Errorf("Model '%s' missing required Control Topics definition", model.Name)
		} else {
			if err := validateTopics(model.Control, rtApp.clusters); err != nil {
				return nil, fmt.Errorf("Model '%s' has invalid Control Topics: %s", model.Name, err.Error())
			}
		}
		if model.Process == nil {
			return nil, fmt.Errorf("Model '%s' missing required Process Topics definition", model.Name)
		} else {
			if err := validateTopics(model.Process, rtApp.clusters); err != nil {
				return nil, fmt.Errorf("Model '%s' has invalid Process Topics: %s", model.Name, err.Error())
			}
		}
		// Persist is optional, so ok if not there
		if model.Persist != nil {
			if err := validateTopics(model.Persist, rtApp.clusters); err != nil {
				return nil, fmt.Errorf("Model '%s' has invalid Persist Topics: %s", model.Name, err.Error())
			}
		}

		// verify models only appear once
		if _, ok := rtApp.models[model.Name]; ok {
			return nil, fmt.Errorf("Model '%s' appears more than once in Application '%s' definition", model.Name, rtApp.app.Name)
		}
		rtApp.models[model.Name] = model
	}

	return &rtApp, nil
}

func validateTopics(topics *pb.Application_Model_Topics, clusters map[string]*pb.Application_Cluster) error {
	if topics.Current == nil {
		return errors.New("Topics missing Current Topic")
	} else {
		if err := validateTopic(topics.Current, clusters); err != nil {
			return err
		}
	}
	if topics.Future != nil {
		if err := validateTopic(topics.Future, clusters); err != nil {
			return err
		}
	}
	return nil
}

func validateTopic(topic *pb.Application_Model_Topic, clusters map[string]*pb.Application_Cluster) error {
	if topic.Name == "" {
		return errors.New("Topic missing Name field")
	}
	if topic.ClusterName == "" {
		return errors.New("Topic missing ClusterName field")
	}
	if _, ok := clusters[topic.ClusterName]; !ok {
		return fmt.Errorf("Topic refers to non-existent cluster: '%s'", topic.ClusterName)
	}
	if topic.PartitionCount < 4 || topic.PartitionCount > 1024 {
		return fmt.Errorf("Topic with out of bounds PartitionCount %d", topic.PartitionCount)
	}
	return nil
}
