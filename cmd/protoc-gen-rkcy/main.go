// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"embed"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	//"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

//go:embed templates
var templates embed.FS

type concernInfo struct {
	Name            string
	Message         *descriptorpb.DescriptorProto
	Service         *descriptorpb.ServiceDescriptorProto
	Commands        []*commandInfo
	LeadingComments string
}

type commandInfo struct {
	Name       string
	HasInput   bool
	InputType  string
	HasOutput  bool
	OutputType string
}

func findConcern(fdp *descriptorpb.FileDescriptorProto) (*concernInfo, error) {
	typePrefix := "." + *fdp.Package + "."

	ci := concernInfo{Commands: make([]*commandInfo, 0)}

	for _, svc := range fdp.Service {
		if strings.HasSuffix(*svc.Name, "Commands") {
			ci.Service = svc
			break
		}
	}

	if ci.Service == nil {
		return nil, nil
	}

	ci.Name = strings.TrimSuffix(*ci.Service.Name, "Commands")

	for _, msg := range fdp.MessageType {
		if *msg.Name == ci.Name {
			ci.Message = msg
			break
		}
	}

	if ci.Message == nil {
		return nil, fmt.Errorf("No Concern Message type matches %sCommands service", *ci.Message.Name)
	}

	for _, loc := range fdp.SourceCodeInfo.Location {
		if len(loc.LeadingDetachedComments) > 0 {
			for _, cmt := range loc.LeadingDetachedComments {
				ci.LeadingComments += "//" + strings.Replace(strings.TrimRight(cmt, "\n"), "\n", "\n//", -1)
			}
		}
	}

	for _, method := range ci.Service.Method {
		cmd := commandInfo{}

		cmd.Name = *method.Name

		if *method.InputType != ".rkcy.Void" {
			cmd.HasInput = true
			cmd.InputType = strings.TrimPrefix(*method.InputType, typePrefix)
		} else {
			cmd.HasInput = false
		}

		if *method.OutputType != ".rkcy.Void" {
			cmd.HasOutput = true
			cmd.OutputType = strings.TrimPrefix(*method.OutputType, typePrefix)
		} else {
			cmd.HasOutput = false
		}

		ci.Commands = append(ci.Commands, &cmd)
	}

	return &ci, nil
}

func main() {
	req := &plugin.CodeGeneratorRequest{}
	reqBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	err = proto.Unmarshal(reqBytes, req)
	if err != nil {
		panic(err)
	}

	rkcyTmplContent, err := templates.ReadFile("templates/rkcy.go.tmpl")
	if err != nil {
		panic(err)
	}
	rkcyTmpl, err := template.New("rkcy").Parse(string(rkcyTmplContent))
	if err != nil {
		panic(err)
	}

	rsp := &plugin.CodeGeneratorResponse{}

	//reqJson := protojson.Format(req)
	//os.Stderr.Write([]byte(reqJson))

	for _, fileToGen := range req.ProtoFile {
		ci, err := findConcern(fileToGen)
		if err != nil {
			panic(err)
		}
		if ci != nil {
			mdFile := plugin.CodeGeneratorResponse_File{}
			name := strings.Replace(*fileToGen.Name, ".proto", ".rkcy.go", -1)
			mdFile.Name = &name

			var contentBld strings.Builder
			err = rkcyTmpl.Execute(&contentBld, ci)
			if err != nil {
				panic(err)
			}
			content := contentBld.String()
			mdFile.Content = &content

			rsp.File = append(rsp.File, &mdFile)
		}
	}

	rspBytes, err := proto.Marshal(rsp)
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(rspBytes)

}
