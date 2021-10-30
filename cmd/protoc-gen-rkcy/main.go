// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"embed"
	"encoding/json"
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
	Package              string
	LeadingComments      string
	HasConfigs           bool
	HasConcerns          bool
	HasConfigsOrConcerns bool

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
	Name       string
	FilePb     *descriptorpb.FileDescriptorProto
	MsgPb      *descriptorpb.DescriptorProto
	KeyField   string
	KeyFieldGo string
	IsConfig   bool
	IsConcern  bool
	IsRelated  bool
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
	Name     string
	Msg      *message
	Svc      *service
	Related  *related
	Commands []*command
}

type related struct {
	Msg            *message
	Fields         []*relatedField
	HasFwdCncs     bool
	HasRvsCncs     bool
	HasPureRelCncs bool
	HasConfigs     bool
}

type relatedField struct {
	Name       string
	NameGo     string
	TypeMsg    *message
	TypeMsgRel *message

	IdField   string
	IdFieldGo string

	RelField   string
	RelFieldGo string

	IsPureRelCnc bool

	PairedWith *relatedField

	IsRepeated bool
	FieldPb    *descriptorpb.FieldDescriptorProto
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
	os.Stderr.WriteString(fmt.Sprintf(format, a...))
}

func debugPrintMessage(msg proto.Message) {
	reqJson := protojson.Format(msg)
	printError(reqJson)
}

func printCatalogDestructive(cat *catalog) {
	for _, msg := range cat.MessageMap {
		msg.FilePb = nil
		msg.MsgPb = nil
	}

	for _, svc := range cat.ServiceMap {
		svc.FilePb = nil
		svc.SvcPb = nil
	}

	for _, cnc := range cat.Concerns {
		if cnc.Related != nil {
			for _, fld := range cnc.Related.Fields {
				fld.FieldPb = nil
			}
		}
	}

	b, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		panic(err)
	}
	printError(string(b))

	printError("Exiting after printCatalogDestructive!!!!!")
	os.Exit(1)
}

func buildRelated(cat *catalog, cnc *concern, msg *message, pkg string) (*related, error) {
	var err error
	msg.KeyField, msg.KeyFieldGo, err = findKeyField(msg.MsgPb)
	if err != nil {
		return nil, err
	}

	rel := &related{
		Msg: msg,
	}
	cnc.Related = rel
	pkgPrefix := fmt.Sprintf(".%s.", pkg)

	if len(msg.MsgPb.Field) == 0 {
		return nil, fmt.Errorf("Related message has no fields: %s", msg.Name)
	}
	for _, pbField := range msg.MsgPb.Field {
		if *pbField.Name == msg.KeyField {
			continue // skip id field
		}
		if *pbField.Type != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
			return nil, fmt.Errorf("Related field is not a message type: %s.%s", msg.Name, *pbField.Name)
		}

		field := &relatedField{
			Name:    *pbField.Name,
			NameGo:  GoCamelCase(*pbField.Name),
			FieldPb: pbField,
		}

		if pbField.Label != nil && *pbField.Label == descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
			field.IsRepeated = true
		}

		// find TypeMsg
		var ok bool
		fieldType := strings.TrimPrefix(*pbField.TypeName, pkgPrefix)
		field.TypeMsg, ok = cat.MessageMap[fieldType]
		if !ok {
			return nil, fmt.Errorf("Related field type not found: %s.%s", msg.Name, *pbField.Name)
		}
		if !field.TypeMsg.IsConcern && !field.TypeMsg.IsConfig && !field.TypeMsg.IsRelated {
			return nil, fmt.Errorf("Related field not a concern, config, or related message: %s.%s", msg.Name, *pbField.Name)
		}

		// find TypeMsg's Related message
		if field.TypeMsg.IsConcern {
			fieldTypeRel := fieldType + "Related"
			field.TypeMsgRel, ok = cat.MessageMap[fieldTypeRel]
			if !ok {
				return nil, fmt.Errorf("Related concern has no related data: %s.%s", msg.Name, *pbField.Name)
			}
			if !field.TypeMsgRel.IsRelated {
				return nil, fmt.Errorf("Related concern's related data is not a related message: %s.%s", msg.Name, *pbField.Name)
			}
		}

		if field.TypeMsg.IsConfig {
			rel.HasConfigs = true
		}

		rltn := proto.GetExtension(pbField.Options, rkcy.E_Relation).(*rkcy.Relation)
		if rltn == nil {
			return nil, fmt.Errorf("Related field lacking 'rkcy.relation' option: %s.%s", msg.Name, *pbField.Name)
		}

		if rltn.IdField != "" {
			field.IdField = rltn.IdField
			field.IdFieldGo = GoCamelCase(rltn.IdField)

			if field.TypeMsg.IsConcern {
				rel.HasFwdCncs = true
			}

			idFld := findField(cnc.Msg.MsgPb, field.IdField)
			if idFld == nil {
				return nil, fmt.Errorf("Related id_field field not found: %s.%s", msg.Name, field.IdField)
			}
		}
		if rltn.RelField != "" {
			field.RelField = rltn.RelField
			field.RelFieldGo = GoCamelCase(rltn.RelField)

			if field.TypeMsg.IsConcern {
				rel.HasRvsCncs = true
				if rltn.IdField == "" {
					rel.HasPureRelCncs = true
					field.IsPureRelCnc = true
				}
			}

			idFld := findField(field.TypeMsg.MsgPb, field.RelField)
			if idFld == nil {
				relIdFld := findField(field.TypeMsgRel.MsgPb, field.RelField)
				if relIdFld == nil {
					return nil, fmt.Errorf("Related rel_id_field not found: %s.%s", msg.Name, field.RelField)
				}
			}
		}
		if rltn.PairedWith != "" {
			if rltn.IdField != "" || rltn.RelField != "" {
				return nil, fmt.Errorf("Invalid rkcy.relation options, no id_field or rel_id_field incompatible with paird_with option: %s.%s", msg.Name, *pbField.Name)
			}
			for _, fld := range rel.Fields {
				if fld.Name == rltn.PairedWith {
					field.PairedWith = fld
				}
			}
			if field.PairedWith == nil {
				return nil, fmt.Errorf("Paired field not found: %s.%s", msg.Name, *pbField.Name)
			}
		} else {
			if rltn.IdField == "" && rltn.RelField == "" {
				return nil, fmt.Errorf("Invalid rkcy.relation options, no id_field or rel_id_field: %s.%s", msg.Name, *pbField.Name)
			}
		}

		// couple more checks for valid relations
		if field.IsRepeated && field.IdField != "" && field.RelField == "" {
			return nil, fmt.Errorf("Invalid rkcy.relation options, repeated fields must have rel_id_field and no id_field: %s.%s", msg.Name, *pbField.Name)
		} else if !field.IsRepeated && field.IdField == "" {
			return nil, fmt.Errorf("Invalid rkcy.relation options, no id_field for non repeated field: %s.%s", msg.Name, *pbField.Name)
		}

		if !ok {
			return nil, fmt.Errorf("Related field type not found: %s.%s, type: %s", msg.Name, *pbField.Name, fieldType)
		}
		rel.Fields = append(rel.Fields, field)
	}

	return rel, nil
}

func findField(pbMsg *descriptorpb.DescriptorProto, fieldName string) *descriptorpb.FieldDescriptorProto {
	for _, pbField := range pbMsg.Field {
		if *pbField.Name == fieldName {
			return pbField
		}
	}
	return nil
}

func findKeyField(pbMsg *descriptorpb.DescriptorProto) (string, string, error) {
	keyField := ""
	for _, pbField := range pbMsg.Field {
		if proto.GetExtension(pbField.Options, rkcy.E_IsKey).(bool) {
			if *pbField.Type != descriptorpb.FieldDescriptorProto_TYPE_STRING {
				return "", "", fmt.Errorf("Non string field with is_key=true: %s.%ss", *pbMsg.Name, *pbField.Name)
			}
			if keyField == "" {
				keyField = *pbField.Name
			} else {
				return "", "", fmt.Errorf("Multiple fields with is_key=true: %s.s and %s.%ss", *pbMsg.Name, keyField, *pbField.Name)
			}
		}
	}
	if keyField == "" {
		return "", "", fmt.Errorf("No field found with is_key=true: %s", *pbMsg.Name)
	}
	return keyField, GoCamelCase(keyField), nil
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
				msg.KeyField, msg.KeyFieldGo, err = findKeyField(pbMsg)
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

			if strings.HasSuffix(msg.Name, "Related") && len(msg.Name) > len("Related") {
				msg.IsRelated = true
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
		// Related
		relName := cnc.Name + "Related"
		relMsg, ok := cat.MessageMap[relName]
		if ok {
			var err error
			cnc.Related, err = buildRelated(cat, cnc, relMsg, *relMsg.FilePb.Package)
			if err != nil {
				return nil, err
			}
		}

		// Commands service methods
		logicName := cnc.Name + "Logic"
		logicSvc, ok := cat.ServiceMap[logicName]
		if ok {
			cnc.Svc = logicSvc
			for _, method := range logicSvc.SvcPb.Method {
				if rkcy.IsReservedCommandName(*method.Name) {
					return nil, fmt.Errorf("Reserved name used as Command: %s.%s", logicSvc.Name, *method.Name)
				}

				cmd := &command{}

				cmd.Name = *method.Name

				typePrefix := *logicSvc.FilePb.Package + "."

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

	tmplData.HasConfigs = len(tmplData.Configs) > 0
	tmplData.HasConcerns = len(tmplData.Concerns) > 0
	tmplData.HasConfigsOrConcerns = tmplData.HasConfigs || tmplData.HasConcerns

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

	//printCatalogDestructive(cat)

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
			if cnc.Related != nil {
				assoc = append(assoc, cnc.Related.Msg.Name)
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
		if tmplData.HasConfigsOrConcerns {
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
