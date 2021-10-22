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
	ServiceMap map[string]*service
	Configs    []*config
	Concerns   []*concern
}

type message struct {
	Name      string
	FilePb    *descriptorpb.FileDescriptorProto
	MsgPb     *descriptorpb.DescriptorProto
	KeyField  string
	IsConfig  bool
	IsConcern bool
}

type service struct {
	Name   string
	FilePb *descriptorpb.FileDescriptorProto
	SvcPb  *descriptorpb.ServiceDescriptorProto
}

type config struct {
	Name string
	Msg  *message
}

type concern struct {
	Name        string
	Msg         *message
	RelConfigs  *related
	RelConcerns *related
	Svc         *service
	Commands    []*command
}

type related struct {
	Msg    *message
	Fields []*messageField
}

type messageField struct {
	Name    string
	Msg     *message
	FieldPb *descriptorpb.FieldDescriptorProto
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

func buildRelated(cat *catalog, msg *message, pkg string) (*related, error) {
	rel := &related{
		Msg: msg,
	}
	pkgPrefix := fmt.Sprintf(".%s.", pkg)

	if len(msg.MsgPb.Field) == 0 {
		return nil, fmt.Errorf("Related message has no fields: %s", msg.Name)
	}
	for _, pbField := range msg.MsgPb.Field {
		if *pbField.Type != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
			return nil, fmt.Errorf("Related field is not a message type: %s.%s", msg.Name, *pbField.Name)
		}
		fieldType := strings.TrimPrefix(*pbField.TypeName, pkgPrefix)
		fieldMsg, ok := cat.MessageMap[fieldType]
		if !ok {
			return nil, fmt.Errorf("Related field type not found: %s.%s, type: %s", msg.Name, *pbField.Name, fieldType)
		}
		rel.Fields = append(rel.Fields, &messageField{
			Name:    *pbField.Name,
			Msg:     fieldMsg,
			FieldPb: pbField,
		})
	}
	return rel, nil
}

func findKeyField(pbMsg *descriptorpb.DescriptorProto) (string, error) {
	keyField := ""
	for _, pbField := range pbMsg.Field {
		if proto.GetExtension(pbField.Options, rkcy.E_IsKey).(bool) {
			if *pbField.Type != descriptorpb.FieldDescriptorProto_TYPE_STRING {
				return "", fmt.Errorf("Non string field with is_key=true: %s.s", *pbMsg.Name, *pbField.Name)
			}
			if keyField == "" {
				jsonName := *pbField.JsonName
				keyField = strings.ToUpper(string(jsonName[0])) + jsonName[1:]
			} else {
				return "", fmt.Errorf("Multiple fields with is_key=true: %s.s and %s.s", *pbMsg.Name, keyField, *pbField.Name)
			}
		}
	}
	if keyField == "" {
		return "", fmt.Errorf("No field found with is_key=true: %s", *pbMsg.Name)
	}
	return keyField, nil
}

func buildCatalog(pbFiles []*descriptorpb.FileDescriptorProto) (*catalog, error) {
	cat := &catalog{
		MessageMap: make(map[string]*message),
		ServiceMap: make(map[string]*service),
	}

	for _, pbFile := range pbFiles {
		for _, pbMsg := range pbFile.MessageType {
			msg := &message{
				Name:   *pbMsg.Name,
				FilePb: pbFile,
				MsgPb:  pbMsg,
			}
			cat.MessageMap[msg.Name] = msg

			msg.IsConfig = proto.GetExtension(pbMsg.Options, rkcy.E_IsConfig).(bool)
			msg.IsConcern = proto.GetExtension(pbMsg.Options, rkcy.E_IsConcern).(bool)
			if msg.IsConfig && msg.IsConcern {
				return nil, fmt.Errorf("Message cannot be both a config and concern: %s", *pbMsg.Name)
			}
			if msg.IsConfig || msg.IsConcern {
				var err error
				msg.KeyField, err = findKeyField(pbMsg)
				if err != nil {
					return nil, err
				}
			}
			if msg.IsConfig {
				conf := &config{
					Name: *pbMsg.Name,
					Msg:  msg,
				}
				cat.Configs = append(cat.Configs, conf)
			} else if msg.IsConcern {
				cnc := &concern{
					Name: *pbMsg.Name,
					Msg:  msg,
				}
				cat.Concerns = append(cat.Concerns, cnc)
			}
		}

		for _, pbSvc := range pbFile.Service {
			svc := &service{
				Name:   *pbSvc.Name,
				FilePb: pbFile,
				SvcPb:  pbSvc,
			}
			cat.ServiceMap[svc.Name] = svc
		}
	}

	// Second pass through concerns to rind associated messages and services
	for _, cnc := range cat.Concerns {

		// RelatedConfigs
		relConfsName := cnc.Name + "RelatedConfigs"
		relConfs, ok := cat.MessageMap[relConfsName]
		if ok {
			var err error
			cnc.RelConfigs, err = buildRelated(cat, relConfs, *relConfs.FilePb.Package)
			if err != nil {
				return nil, err
			}

			for _, field := range cnc.RelConfigs.Fields {
				if !field.Msg.IsConfig {
					return nil, fmt.Errorf("RelatedConfigs field not a config %s.%s", cnc.RelConfigs.Msg.Name, field.Name)
				}
			}
		}

		// RelatedConcerns
		relCncsName := cnc.Name + "RelatedConcerns"
		relCncs, ok := cat.MessageMap[relCncsName]
		if ok {
			var err error
			cnc.RelConcerns, err = buildRelated(cat, relCncs, *relCncs.FilePb.Package)
			if err != nil {
				return nil, err
			}

			for _, field := range cnc.RelConcerns.Fields {
				if !field.Msg.IsConcern {
					return nil, fmt.Errorf("RelatedConcerns field not a concern %s.%s", cnc.RelConcerns.Msg.Name, field.Name)
				}
			}
		}

		// Commands service methods
		cmdsName := cnc.Name + "Commands"
		cmdsSvc, ok := cat.ServiceMap[cmdsName]
		if ok {
			cnc.Svc = cmdsSvc
			for _, method := range cmdsSvc.SvcPb.Method {
				if rkcy.IsReservedCommandName(*method.Name) {
					return nil, fmt.Errorf("Reserved name used as Command: %s.%s", cmdsSvc.Name, *method.Name)
				}

				cmd := &command{}

				cmd.Name = *method.Name

				typePrefix := *cmdsSvc.FilePb.Package + "."

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
			assoc := make([]string, 0)
			if cnc.Svc != nil {
				assoc = append(assoc, cnc.Svc.Name)
			}
			if cnc.RelConfigs != nil {
				assoc = append(assoc, cnc.RelConfigs.Msg.Name)
			}
			if cnc.RelConcerns != nil {
				assoc = append(assoc, cnc.RelConcerns.Msg.Name)
			}
			withClause := ""
			if len(assoc) > 0 {
				withClause = fmt.Sprintf(" with %s", strings.Join(assoc, ", "))
			}
			printError("  %s%s", cnc.Name, withClause)
		}
	}

	for _, pbFile := range pbFilesToGen {
		tmplData := buildTemplateData(pbFile, cat, rkcyPackage)
		if tmplData.ShouldGenerate {
			mdFile := plugin.CodeGeneratorResponse_File{}
			name := strings.Replace(*pbFile.Name, ".proto", ".rkcy.go", -1)
			mdFile.Name = &name

			printError("Writing %s", name)

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
