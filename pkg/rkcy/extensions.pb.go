// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.17.3
// source: rkcy/extensions.proto

package rkcy

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type RelConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	IdField string `protobuf:"bytes,1,opt,name=id_field,json=idField,proto3" json:"id_field,omitempty"`
}

func (x *RelConfig) Reset() {
	*x = RelConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rkcy_extensions_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RelConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RelConfig) ProtoMessage() {}

func (x *RelConfig) ProtoReflect() protoreflect.Message {
	mi := &file_rkcy_extensions_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RelConfig.ProtoReflect.Descriptor instead.
func (*RelConfig) Descriptor() ([]byte, []int) {
	return file_rkcy_extensions_proto_rawDescGZIP(), []int{0}
}

func (x *RelConfig) GetIdField() string {
	if x != nil {
		return x.IdField
	}
	return ""
}

type RelConcern struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	IdField  string `protobuf:"bytes,1,opt,name=id_field,json=idField,proto3" json:"id_field,omitempty"`
	IsRemote bool   `protobuf:"varint,2,opt,name=is_remote,json=isRemote,proto3" json:"is_remote,omitempty"`
}

func (x *RelConcern) Reset() {
	*x = RelConcern{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rkcy_extensions_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RelConcern) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RelConcern) ProtoMessage() {}

func (x *RelConcern) ProtoReflect() protoreflect.Message {
	mi := &file_rkcy_extensions_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RelConcern.ProtoReflect.Descriptor instead.
func (*RelConcern) Descriptor() ([]byte, []int) {
	return file_rkcy_extensions_proto_rawDescGZIP(), []int{1}
}

func (x *RelConcern) GetIdField() string {
	if x != nil {
		return x.IdField
	}
	return ""
}

func (x *RelConcern) GetIsRemote() bool {
	if x != nil {
		return x.IsRemote
	}
	return false
}

var file_rkcy_extensions_proto_extTypes = []protoimpl.ExtensionInfo{
	{
		ExtendedType:  (*descriptorpb.MessageOptions)(nil),
		ExtensionType: (*bool)(nil),
		Field:         11371000,
		Name:          "rkcy.is_config",
		Tag:           "varint,11371000,opt,name=is_config",
		Filename:      "rkcy/extensions.proto",
	},
	{
		ExtendedType:  (*descriptorpb.MessageOptions)(nil),
		ExtensionType: (*bool)(nil),
		Field:         11371001,
		Name:          "rkcy.is_concern",
		Tag:           "varint,11371001,opt,name=is_concern",
		Filename:      "rkcy/extensions.proto",
	},
	{
		ExtendedType:  (*descriptorpb.FieldOptions)(nil),
		ExtensionType: (*bool)(nil),
		Field:         11371100,
		Name:          "rkcy.is_key",
		Tag:           "varint,11371100,opt,name=is_key",
		Filename:      "rkcy/extensions.proto",
	},
	{
		ExtendedType:  (*descriptorpb.FieldOptions)(nil),
		ExtensionType: (*RelConfig)(nil),
		Field:         11371101,
		Name:          "rkcy.rel_config",
		Tag:           "bytes,11371101,opt,name=rel_config",
		Filename:      "rkcy/extensions.proto",
	},
	{
		ExtendedType:  (*descriptorpb.FieldOptions)(nil),
		ExtensionType: (*RelConcern)(nil),
		Field:         11371102,
		Name:          "rkcy.rel_concern",
		Tag:           "bytes,11371102,opt,name=rel_concern",
		Filename:      "rkcy/extensions.proto",
	},
}

// Extension fields to descriptorpb.MessageOptions.
var (
	// optional bool is_config = 11371000;
	E_IsConfig = &file_rkcy_extensions_proto_extTypes[0]
	// optional bool is_concern = 11371001;
	E_IsConcern = &file_rkcy_extensions_proto_extTypes[1]
)

// Extension fields to descriptorpb.FieldOptions.
var (
	// optional bool is_key = 11371100;
	E_IsKey = &file_rkcy_extensions_proto_extTypes[2]
	// optional rkcy.RelConfig rel_config = 11371101;
	E_RelConfig = &file_rkcy_extensions_proto_extTypes[3]
	// optional rkcy.RelConcern rel_concern = 11371102;
	E_RelConcern = &file_rkcy_extensions_proto_extTypes[4]
)

var File_rkcy_extensions_proto protoreflect.FileDescriptor

var file_rkcy_extensions_proto_rawDesc = []byte{
	0x0a, 0x15, 0x72, 0x6b, 0x63, 0x79, 0x2f, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x04, 0x72, 0x6b, 0x63, 0x79, 0x1a, 0x20, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64,
	0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0x26, 0x0a, 0x09, 0x52, 0x65, 0x6c, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x19, 0x0a, 0x08,
	0x69, 0x64, 0x5f, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07,
	0x69, 0x64, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x22, 0x44, 0x0a, 0x0a, 0x52, 0x65, 0x6c, 0x43, 0x6f,
	0x6e, 0x63, 0x65, 0x72, 0x6e, 0x12, 0x19, 0x0a, 0x08, 0x69, 0x64, 0x5f, 0x66, 0x69, 0x65, 0x6c,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x69, 0x64, 0x46, 0x69, 0x65, 0x6c, 0x64,
	0x12, 0x1b, 0x0a, 0x09, 0x69, 0x73, 0x5f, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x08, 0x52, 0x08, 0x69, 0x73, 0x52, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x3a, 0x42, 0x0a,
	0x09, 0x69, 0x73, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x1f, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf8, 0x83, 0xb6, 0x05,
	0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x69, 0x73, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x88, 0x01,
	0x01, 0x3a, 0x44, 0x0a, 0x0a, 0x69, 0x73, 0x5f, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x12,
	0x1f, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x18, 0xf9, 0x83, 0xb6, 0x05, 0x20, 0x01, 0x28, 0x08, 0x52, 0x09, 0x69, 0x73, 0x43, 0x6f, 0x6e,
	0x63, 0x65, 0x72, 0x6e, 0x88, 0x01, 0x01, 0x3a, 0x3a, 0x0a, 0x06, 0x69, 0x73, 0x5f, 0x6b, 0x65,
	0x79, 0x12, 0x1d, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x18, 0xdc, 0x84, 0xb6, 0x05, 0x20, 0x01, 0x28, 0x08, 0x52, 0x05, 0x69, 0x73, 0x4b, 0x65, 0x79,
	0x88, 0x01, 0x01, 0x3a, 0x53, 0x0a, 0x0a, 0x72, 0x65, 0x6c, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x12, 0x1d, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x18, 0xdd, 0x84, 0xb6, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x72, 0x6b, 0x63, 0x79,
	0x2e, 0x52, 0x65, 0x6c, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x09, 0x72, 0x65, 0x6c, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x88, 0x01, 0x01, 0x3a, 0x56, 0x0a, 0x0b, 0x72, 0x65, 0x6c, 0x5f,
	0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x12, 0x1d, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x4f,
	0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xde, 0x84, 0xb6, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x10, 0x2e, 0x72, 0x6b, 0x63, 0x79, 0x2e, 0x52, 0x65, 0x6c, 0x43, 0x6f, 0x6e, 0x63, 0x65, 0x72,
	0x6e, 0x52, 0x0a, 0x72, 0x65, 0x6c, 0x43, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x88, 0x01, 0x01,
	0x42, 0x2c, 0x5a, 0x2a, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6c,
	0x61, 0x63, 0x68, 0x6c, 0x61, 0x6e, 0x6f, 0x72, 0x72, 0x2f, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74,
	0x63, 0x79, 0x63, 0x6c, 0x65, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x72, 0x6b, 0x63, 0x79, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_rkcy_extensions_proto_rawDescOnce sync.Once
	file_rkcy_extensions_proto_rawDescData = file_rkcy_extensions_proto_rawDesc
)

func file_rkcy_extensions_proto_rawDescGZIP() []byte {
	file_rkcy_extensions_proto_rawDescOnce.Do(func() {
		file_rkcy_extensions_proto_rawDescData = protoimpl.X.CompressGZIP(file_rkcy_extensions_proto_rawDescData)
	})
	return file_rkcy_extensions_proto_rawDescData
}

var file_rkcy_extensions_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_rkcy_extensions_proto_goTypes = []interface{}{
	(*RelConfig)(nil),                   // 0: rkcy.RelConfig
	(*RelConcern)(nil),                  // 1: rkcy.RelConcern
	(*descriptorpb.MessageOptions)(nil), // 2: google.protobuf.MessageOptions
	(*descriptorpb.FieldOptions)(nil),   // 3: google.protobuf.FieldOptions
}
var file_rkcy_extensions_proto_depIdxs = []int32{
	2, // 0: rkcy.is_config:extendee -> google.protobuf.MessageOptions
	2, // 1: rkcy.is_concern:extendee -> google.protobuf.MessageOptions
	3, // 2: rkcy.is_key:extendee -> google.protobuf.FieldOptions
	3, // 3: rkcy.rel_config:extendee -> google.protobuf.FieldOptions
	3, // 4: rkcy.rel_concern:extendee -> google.protobuf.FieldOptions
	0, // 5: rkcy.rel_config:type_name -> rkcy.RelConfig
	1, // 6: rkcy.rel_concern:type_name -> rkcy.RelConcern
	7, // [7:7] is the sub-list for method output_type
	7, // [7:7] is the sub-list for method input_type
	5, // [5:7] is the sub-list for extension type_name
	0, // [0:5] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_rkcy_extensions_proto_init() }
func file_rkcy_extensions_proto_init() {
	if File_rkcy_extensions_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_rkcy_extensions_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RelConfig); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_rkcy_extensions_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RelConcern); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_rkcy_extensions_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 5,
			NumServices:   0,
		},
		GoTypes:           file_rkcy_extensions_proto_goTypes,
		DependencyIndexes: file_rkcy_extensions_proto_depIdxs,
		MessageInfos:      file_rkcy_extensions_proto_msgTypes,
		ExtensionInfos:    file_rkcy_extensions_proto_extTypes,
	}.Build()
	File_rkcy_extensions_proto = out.File
	file_rkcy_extensions_proto_rawDesc = nil
	file_rkcy_extensions_proto_goTypes = nil
	file_rkcy_extensions_proto_depIdxs = nil
}
