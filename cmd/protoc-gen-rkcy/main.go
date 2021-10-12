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

type parseResult struct {
	Package         string
	PbPackage       string
	LeadingComments string

	MessageMap     map[string]*messageInfo
	Messages       []*messageInfo
	Configs        []*configInfo
	Concerns       []*concernInfo
	PrimaryConcern *concernInfo
}

type messageInfo struct {
	Name    string
	Message *descriptorpb.DescriptorProto
}

type configInfo struct {
	Name    string
	Message *descriptorpb.DescriptorProto
}

type concernInfo struct {
	Name     string
	Message  *descriptorpb.DescriptorProto
	Service  *descriptorpb.ServiceDescriptorProto
	Commands []*commandInfo
}

type commandInfo struct {
	Name       string
	HasInput   bool
	InputType  string
	HasOutput  bool
	OutputType string
}

func extractMessages(pbMsg *descriptorpb.DescriptorProto, parent string) []*messageInfo {
	var name string
	if parent != "" {
		name = fmt.Sprintf("%s_%s", parent, *pbMsg.Name)
	} else {
		name = *pbMsg.Name
	}

	msg := &messageInfo{
		Name:    name,
		Message: pbMsg,
	}
	msgs := []*messageInfo{msg}

	for _, nested := range pbMsg.NestedType {
		msgs = append(msgs, extractMessages(nested, name)...)
	}

	return msgs
}

func parseDescriptor(fdp *descriptorpb.FileDescriptorProto, rkcyPackage string) (*parseResult, error) {
	parseRes := &parseResult{
		MessageMap: make(map[string]*messageInfo),
	}

	typePrefix := *fdp.Package + "."
	packageParts := strings.Split(*fdp.Package, ".")

	if rkcyPackage != "" {
		parseRes.Package = rkcyPackage
	} else {
		parseRes.Package = packageParts[len(packageParts)-1]
	}
	parseRes.PbPackage = *fdp.Options.GoPackage

	for _, loc := range fdp.SourceCodeInfo.Location {
		if len(loc.LeadingDetachedComments) > 0 {
			for _, cmt := range loc.LeadingDetachedComments {
				parseRes.LeadingComments += "//" + strings.Replace(strings.TrimRight(cmt, "\n"), "\n", "\n//", -1)
			}
		}
	}

	// find and cleanup messages
	for _, pbMsg := range fdp.MessageType {
		parseRes.Messages = append(parseRes.Messages, extractMessages(pbMsg, "")...)

		// TODO: add config if this is a config message
	}

	for _, msg := range parseRes.Messages {
		parseRes.MessageMap[msg.Name] = msg
	}

	// find and cleanup concerns
	for _, pbSvc := range fdp.Service {
		if strings.HasSuffix(*pbSvc.Name, "Commands") {
			cncName := strings.TrimSuffix(*pbSvc.Name, "Commands")

			msg, ok := parseRes.MessageMap[cncName]
			if !ok {
				return nil, fmt.Errorf("No matching message for concern commands: %s", *pbSvc.Name)
			}
			cnc := &concernInfo{
				Name:    cncName,
				Message: msg.Message,
				Service: pbSvc,
			}

			for _, method := range cnc.Service.Method {
				cmd := &commandInfo{}

				cmd.Name = *method.Name

				if *method.InputType != ".rkcy.Void" {
					cmd.HasInput = true
					cmd.InputType = strings.TrimPrefix(*method.InputType, ".")
					cmd.InputType = strings.TrimPrefix(cmd.InputType, typePrefix)
				} else {
					cmd.HasInput = false
				}

				if *method.OutputType != ".rkcy.Void" {
					cmd.HasOutput = true
					cmd.OutputType = strings.TrimPrefix(*method.OutputType, ".")
					cmd.OutputType = strings.TrimPrefix(cmd.OutputType, typePrefix)
				} else {
					cmd.HasOutput = false
				}

				cnc.Commands = append(cnc.Commands, cmd)
			}

			parseRes.Concerns = append(parseRes.Concerns, cnc)
			if parseRes.PrimaryConcern == nil {
				parseRes.PrimaryConcern = cnc
			}
		}
	}

	return parseRes, nil
}

func parseParameters(params *string) map[string]string {
	paramMap := make(map[string]string)
	if params != nil {
		for _, param := range strings.Split(*params, ",") {
			parts := strings.SplitN(param, "=", 2)
			if len(parts) == 1 {
				paramMap[param] = ""
			} else {
				paramMap[parts[0]] = parts[1]
			}
		}
	}
	return paramMap
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

	params := parseParameters(req.Parameter)
	rkcyPackage := ""
	if paramPackage, ok := params["package"]; ok {
		rkcyPackage = paramPackage
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
		parseRes, err := parseDescriptor(fileToGen, rkcyPackage)
		if err != nil {
			panic(err)
		}
		if len(parseRes.Concerns) > 1 {
			panic(fmt.Errorf("More than one concern defined in same file"))
		}
		if len(parseRes.Concerns) > 0 {
			mdFile := plugin.CodeGeneratorResponse_File{}
			name := strings.Replace(*fileToGen.Name, ".proto", ".rkcy.go", -1)
			mdFile.Name = &name

			var contentBld strings.Builder
			err = rkcyTmpl.Execute(&contentBld, parseRes)
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
