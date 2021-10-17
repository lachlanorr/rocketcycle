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
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

//go:embed templates
var templates embed.FS

type templateData struct {
	Package         string
	LeadingComments string
	ShouldGenerate  bool

	Configs  []*config
	Concerns []*concern
}

type catalog struct {
	MessageMap map[string]*message
	Configs    []*config
	Concerns   []*concern
}

type message struct {
	Name   string
	FilePb *descriptorpb.FileDescriptorProto
	MsgPb  *descriptorpb.DescriptorProto
}

type config struct {
	Name string
	Msg  *message
}

type concern struct {
	Name      string
	Msg       *message
	Rel       *message
	SvcPb     *descriptorpb.ServiceDescriptorProto
	SvcFilePb *descriptorpb.FileDescriptorProto
	Commands  []*command
}

type command struct {
	Name       string
	HasInput   bool
	InputType  string
	HasOutput  bool
	OutputType string
}

func printError(format string, a ...interface{}) {
	if len(format) == 0 || format[len(format)-1] != '\n' {
		format = format + "\n"
	}
	os.Stderr.Write([]byte(fmt.Sprintf(format, a...)))
}

func debugPrintMessage(msg proto.Message) {
	reqJson := protojson.Format(msg)
	printError(reqJson)
}

func buildCatalog(pbFiles []*descriptorpb.FileDescriptorProto) (*catalog, error) {
	cat := &catalog{MessageMap: make(map[string]*message)}

	for _, pbFile := range pbFiles {
		for _, pbMsg := range pbFile.MessageType {
			msg := &message{
				Name:   *pbMsg.Name,
				FilePb: pbFile,
				MsgPb:  pbMsg,
			}
			cat.MessageMap[msg.Name] = msg

			is_config := proto.GetExtension(pbMsg.Options, rkcy.E_IsConfig).(bool)
			if is_config {
				conf := &config{
					Name: *pbMsg.Name,
					Msg:  msg,
				}
				cat.Configs = append(cat.Configs, conf)
			}
		}
	}

	// Second pass through files to find Concerns messages which can
	// reference other messages by naming convention
	for _, pbFile := range pbFiles {
		typePrefix := *pbFile.Package + "."
		for _, pbSvc := range pbFile.Service {
			if strings.HasSuffix(*pbSvc.Name, "Commands") {
				cncName := strings.TrimSuffix(*pbSvc.Name, "Commands")

				if len(cncName) > 0 {
					msg, ok := cat.MessageMap[cncName]
					if !ok {
						return nil, fmt.Errorf("No matching message for concern commands: %s", *pbSvc.Name)
					}
					cnc := &concern{
						Name:      cncName,
						Msg:       msg,
						SvcPb:     pbSvc,
						SvcFilePb: pbFile,
					}

					relName := cncName + "Related"
					rel, ok := cat.MessageMap[relName]
					if ok {
						cnc.Rel = rel
					}

					for _, method := range cnc.SvcPb.Method {
						cmd := &command{}

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

					cat.Concerns = append(cat.Concerns, cnc)
				}
			}
		}
	}
	return cat, nil
}

func buildTemplateData(pbFile *descriptorpb.FileDescriptorProto, cat *catalog, rkcyPackage string) *templateData {
	tmplData := &templateData{}

	packageParts := strings.Split(*pbFile.Package, ".")
	if rkcyPackage != "" {
		tmplData.Package = rkcyPackage
	} else {
		tmplData.Package = packageParts[len(packageParts)-1]
	}

	for _, loc := range pbFile.SourceCodeInfo.Location {
		if len(loc.LeadingDetachedComments) > 0 {
			for _, cmt := range loc.LeadingDetachedComments {
				tmplData.LeadingComments += "//" + strings.Replace(strings.TrimRight(cmt, "\n"), "\n", "\n//", -1)
			}
		}
	}

	for _, cnf := range cat.Configs {
		if cnf.Msg.FilePb == pbFile {
			tmplData.Configs = append(tmplData.Configs, cnf)
		}
	}

	for _, cnc := range cat.Concerns {
		if cnc.Msg.FilePb == pbFile {
			tmplData.Concerns = append(tmplData.Concerns, cnc)
		}
	}

	tmplData.ShouldGenerate = len(tmplData.Configs) > 0 || len(tmplData.Concerns) > 0

	return tmplData
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

func findFileDescriptor(name string, pbFiles []*descriptorpb.FileDescriptorProto) *descriptorpb.FileDescriptorProto {
	for _, pbFile := range pbFiles {
		if *pbFile.Name == name {
			return pbFile
		}
	}
	return nil
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

	pbFilesToGen := make([]*descriptorpb.FileDescriptorProto, len(req.FileToGenerate))
	for i, pbFileName := range req.FileToGenerate {
		pbFile := findFileDescriptor(pbFileName, req.ProtoFile)
		if pbFile == nil {
			printError("FileDescriptor not found: %s", pbFileName)
		}
		pbFilesToGen[i] = pbFile
	}

	cat, err := buildCatalog(pbFilesToGen)
	if err != nil {
		printError("Unable to build catalog: %s", err.Error())
		panic(err)
	}

	// print some debug info
	if len(cat.Configs) > 0 {
		printError("Compiling Configs:")
		for _, cnf := range cat.Configs {
			printError("  %s", cnf.Name)
		}
	}
	if len(cat.Concerns) > 0 {
		printError("Compiling Concerns:")
		for _, cnc := range cat.Concerns {
			line := fmt.Sprintf("  %s", cnc.Name)
			if cnc.Rel != nil {
				line += fmt.Sprintf(", with %s", cnc.Rel.Name)
			}
			printError(line)
		}
	}

	for _, pbFile := range pbFilesToGen {
		tmplData := buildTemplateData(pbFile, cat, rkcyPackage)
		if tmplData.ShouldGenerate {
			mdFile := plugin.CodeGeneratorResponse_File{}
			name := strings.Replace(*pbFile.Name, ".proto", ".rkcy.go", -1)
			mdFile.Name = &name

			var contentBld strings.Builder
			err = rkcyTmpl.Execute(&contentBld, tmplData)
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
