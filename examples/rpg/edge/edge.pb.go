// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.13.0
// source: edge.proto

package edge

import (
	proto "github.com/golang/protobuf/proto"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2/options"
	concerns "github.com/lachlanorr/rocketcycle/examples/rpg/concerns"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type RpgRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *RpgRequest) Reset() {
	*x = RpgRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_edge_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RpgRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RpgRequest) ProtoMessage() {}

func (x *RpgRequest) ProtoReflect() protoreflect.Message {
	mi := &file_edge_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RpgRequest.ProtoReflect.Descriptor instead.
func (*RpgRequest) Descriptor() ([]byte, []int) {
	return file_edge_proto_rawDescGZIP(), []int{0}
}

func (x *RpgRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type RpgResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *RpgResponse) Reset() {
	*x = RpgResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_edge_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RpgResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RpgResponse) ProtoMessage() {}

func (x *RpgResponse) ProtoReflect() protoreflect.Message {
	mi := &file_edge_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RpgResponse.ProtoReflect.Descriptor instead.
func (*RpgResponse) Descriptor() ([]byte, []int) {
	return file_edge_proto_rawDescGZIP(), []int{1}
}

func (x *RpgResponse) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

var File_edge_proto protoreflect.FileDescriptor

var file_edge_proto_rawDesc = []byte{
	0x0a, 0x0a, 0x65, 0x64, 0x67, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1d, 0x72, 0x6f,
	0x63, 0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c,
	0x65, 0x73, 0x2e, 0x72, 0x70, 0x67, 0x2e, 0x65, 0x64, 0x67, 0x65, 0x1a, 0x1c, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x63, 0x2d, 0x67, 0x65, 0x6e, 0x2d, 0x6f, 0x70, 0x65, 0x6e, 0x61, 0x70, 0x69, 0x76, 0x32, 0x2f,
	0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x22, 0x65, 0x78, 0x61, 0x6d, 0x70,
	0x6c, 0x65, 0x73, 0x2f, 0x72, 0x70, 0x67, 0x2f, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x73,
	0x2f, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x25, 0x65,
	0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2f, 0x72, 0x70, 0x67, 0x2f, 0x63, 0x6f, 0x6e, 0x63,
	0x65, 0x72, 0x6e, 0x73, 0x2f, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0x72, 0x0a, 0x0a, 0x52, 0x70, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x64, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x54,
	0x92, 0x41, 0x51, 0x80, 0x01, 0x01, 0x8a, 0x01, 0x4b, 0x5b, 0x61, 0x2d, 0x66, 0x41, 0x2d, 0x46,
	0x30, 0x2d, 0x39, 0x5d, 0x7b, 0x38, 0x7d, 0x2d, 0x5b, 0x61, 0x2d, 0x66, 0x41, 0x2d, 0x46, 0x30,
	0x2d, 0x39, 0x5d, 0x7b, 0x34, 0x7d, 0x2d, 0x5b, 0x61, 0x2d, 0x66, 0x41, 0x2d, 0x46, 0x30, 0x2d,
	0x39, 0x5d, 0x7b, 0x34, 0x7d, 0x2d, 0x5b, 0x61, 0x2d, 0x66, 0x41, 0x2d, 0x46, 0x30, 0x2d, 0x39,
	0x5d, 0x7b, 0x34, 0x7d, 0x2d, 0x5b, 0x61, 0x2d, 0x66, 0x41, 0x2d, 0x46, 0x30, 0x2d, 0x39, 0x5d,
	0x7b, 0x31, 0x32, 0x7d, 0x52, 0x02, 0x69, 0x64, 0x22, 0x73, 0x0a, 0x0b, 0x52, 0x70, 0x67, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x64, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x42, 0x54, 0x92, 0x41, 0x51, 0x80, 0x01, 0x01, 0x8a, 0x01, 0x4b, 0x5b, 0x61,
	0x2d, 0x66, 0x41, 0x2d, 0x46, 0x30, 0x2d, 0x39, 0x5d, 0x7b, 0x38, 0x7d, 0x2d, 0x5b, 0x61, 0x2d,
	0x66, 0x41, 0x2d, 0x46, 0x30, 0x2d, 0x39, 0x5d, 0x7b, 0x34, 0x7d, 0x2d, 0x5b, 0x61, 0x2d, 0x66,
	0x41, 0x2d, 0x46, 0x30, 0x2d, 0x39, 0x5d, 0x7b, 0x34, 0x7d, 0x2d, 0x5b, 0x61, 0x2d, 0x66, 0x41,
	0x2d, 0x46, 0x30, 0x2d, 0x39, 0x5d, 0x7b, 0x34, 0x7d, 0x2d, 0x5b, 0x61, 0x2d, 0x66, 0x41, 0x2d,
	0x46, 0x30, 0x2d, 0x39, 0x5d, 0x7b, 0x31, 0x32, 0x7d, 0x52, 0x02, 0x69, 0x64, 0x32, 0xef, 0x09,
	0x0a, 0x0a, 0x52, 0x70, 0x67, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x80, 0x01, 0x0a,
	0x0a, 0x52, 0x65, 0x61, 0x64, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x12, 0x29, 0x2e, 0x72, 0x6f,
	0x63, 0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c,
	0x65, 0x73, 0x2e, 0x72, 0x70, 0x67, 0x2e, 0x65, 0x64, 0x67, 0x65, 0x2e, 0x52, 0x70, 0x67, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x29, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63,
	0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70,
	0x67, 0x2e, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x73, 0x2e, 0x50, 0x6c, 0x61, 0x79, 0x65,
	0x72, 0x22, 0x1c, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x16, 0x12, 0x14, 0x2f, 0x76, 0x31, 0x2f, 0x70,
	0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x72, 0x65, 0x61, 0x64, 0x2f, 0x7b, 0x69, 0x64, 0x7d, 0x12,
	0x82, 0x01, 0x0a, 0x0c, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72,
	0x12, 0x29, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65,
	0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70, 0x67, 0x2e, 0x63, 0x6f, 0x6e, 0x63,
	0x65, 0x72, 0x6e, 0x73, 0x2e, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x1a, 0x29, 0x2e, 0x72, 0x6f,
	0x63, 0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c,
	0x65, 0x73, 0x2e, 0x72, 0x70, 0x67, 0x2e, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x73, 0x2e,
	0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x22, 0x1c, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x16, 0x22, 0x11,
	0x2f, 0x76, 0x31, 0x2f, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x63, 0x72, 0x65, 0x61, 0x74,
	0x65, 0x3a, 0x01, 0x2a, 0x12, 0x82, 0x01, 0x0a, 0x0c, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x50,
	0x6c, 0x61, 0x79, 0x65, 0x72, 0x12, 0x29, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79,
	0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70, 0x67,
	0x2e, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x73, 0x2e, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72,
	0x1a, 0x29, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65,
	0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70, 0x67, 0x2e, 0x63, 0x6f, 0x6e, 0x63,
	0x65, 0x72, 0x6e, 0x73, 0x2e, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x22, 0x1c, 0x82, 0xd3, 0xe4,
	0x93, 0x02, 0x16, 0x22, 0x11, 0x2f, 0x76, 0x31, 0x2f, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f,
	0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x3a, 0x01, 0x2a, 0x12, 0x85, 0x01, 0x0a, 0x0c, 0x44, 0x65,
	0x6c, 0x65, 0x74, 0x65, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x12, 0x29, 0x2e, 0x72, 0x6f, 0x63,
	0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65,
	0x73, 0x2e, 0x72, 0x70, 0x67, 0x2e, 0x65, 0x64, 0x67, 0x65, 0x2e, 0x52, 0x70, 0x67, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2a, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79,
	0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70, 0x67,
	0x2e, 0x65, 0x64, 0x67, 0x65, 0x2e, 0x52, 0x70, 0x67, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x22, 0x1e, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x18, 0x22, 0x16, 0x2f, 0x76, 0x31, 0x2f, 0x70,
	0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x64, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x2f, 0x7b, 0x69, 0x64,
	0x7d, 0x12, 0x89, 0x01, 0x0a, 0x0d, 0x52, 0x65, 0x61, 0x64, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63,
	0x74, 0x65, 0x72, 0x12, 0x29, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c,
	0x65, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70, 0x67, 0x2e, 0x65,
	0x64, 0x67, 0x65, 0x2e, 0x52, 0x70, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2c,
	0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78, 0x61,
	0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70, 0x67, 0x2e, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72,
	0x6e, 0x73, 0x2e, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x22, 0x1f, 0x82, 0xd3,
	0xe4, 0x93, 0x02, 0x19, 0x12, 0x17, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63,
	0x74, 0x65, 0x72, 0x2f, 0x72, 0x65, 0x61, 0x64, 0x2f, 0x7b, 0x69, 0x64, 0x7d, 0x12, 0x8e, 0x01,
	0x0a, 0x0f, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65,
	0x72, 0x12, 0x2c, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2e,
	0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70, 0x67, 0x2e, 0x63, 0x6f, 0x6e,
	0x63, 0x65, 0x72, 0x6e, 0x73, 0x2e, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x1a,
	0x2c, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78,
	0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70, 0x67, 0x2e, 0x63, 0x6f, 0x6e, 0x63, 0x65,
	0x72, 0x6e, 0x73, 0x2e, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x22, 0x1f, 0x82,
	0xd3, 0xe4, 0x93, 0x02, 0x19, 0x22, 0x14, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x68, 0x61, 0x72, 0x61,
	0x63, 0x74, 0x65, 0x72, 0x2f, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x3a, 0x01, 0x2a, 0x12, 0x8e,
	0x01, 0x0a, 0x0f, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74,
	0x65, 0x72, 0x12, 0x2c, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65,
	0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70, 0x67, 0x2e, 0x63, 0x6f,
	0x6e, 0x63, 0x65, 0x72, 0x6e, 0x73, 0x2e, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72,
	0x1a, 0x2c, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65,
	0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70, 0x67, 0x2e, 0x63, 0x6f, 0x6e, 0x63,
	0x65, 0x72, 0x6e, 0x73, 0x2e, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x22, 0x1f,
	0x82, 0xd3, 0xe4, 0x93, 0x02, 0x19, 0x22, 0x14, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x68, 0x61, 0x72,
	0x61, 0x63, 0x74, 0x65, 0x72, 0x2f, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x3a, 0x01, 0x2a, 0x12,
	0x8b, 0x01, 0x0a, 0x0f, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63,
	0x74, 0x65, 0x72, 0x12, 0x29, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c,
	0x65, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70, 0x67, 0x2e, 0x65,
	0x64, 0x67, 0x65, 0x2e, 0x52, 0x70, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2a,
	0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78, 0x61,
	0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70, 0x67, 0x2e, 0x65, 0x64, 0x67, 0x65, 0x2e, 0x52,
	0x70, 0x67, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x21, 0x82, 0xd3, 0xe4, 0x93,
	0x02, 0x1b, 0x22, 0x19, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65,
	0x72, 0x2f, 0x64, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x2f, 0x7b, 0x69, 0x64, 0x7d, 0x12, 0x8f, 0x01,
	0x0a, 0x0d, 0x46, 0x75, 0x6e, 0x64, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x12,
	0x31, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78,
	0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70, 0x67, 0x2e, 0x63, 0x6f, 0x6e, 0x63, 0x65,
	0x72, 0x6e, 0x73, 0x2e, 0x46, 0x75, 0x6e, 0x64, 0x69, 0x6e, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x2c, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65,
	0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70, 0x67, 0x2e, 0x63, 0x6f,
	0x6e, 0x63, 0x65, 0x72, 0x6e, 0x73, 0x2e, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72,
	0x22, 0x1d, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x17, 0x22, 0x12, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x68,
	0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x2f, 0x66, 0x75, 0x6e, 0x64, 0x3a, 0x01, 0x2a, 0x42,
	0x35, 0x5a, 0x33, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6c, 0x61,
	0x63, 0x68, 0x6c, 0x61, 0x6e, 0x6f, 0x72, 0x72, 0x2f, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63,
	0x79, 0x63, 0x6c, 0x65, 0x2f, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2f, 0x72, 0x70,
	0x67, 0x2f, 0x65, 0x64, 0x67, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_edge_proto_rawDescOnce sync.Once
	file_edge_proto_rawDescData = file_edge_proto_rawDesc
)

func file_edge_proto_rawDescGZIP() []byte {
	file_edge_proto_rawDescOnce.Do(func() {
		file_edge_proto_rawDescData = protoimpl.X.CompressGZIP(file_edge_proto_rawDescData)
	})
	return file_edge_proto_rawDescData
}

var file_edge_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_edge_proto_goTypes = []interface{}{
	(*RpgRequest)(nil),              // 0: rocketcycle.examples.rpg.edge.RpgRequest
	(*RpgResponse)(nil),             // 1: rocketcycle.examples.rpg.edge.RpgResponse
	(*concerns.Player)(nil),         // 2: rocketcycle.examples.rpg.concerns.Player
	(*concerns.Character)(nil),      // 3: rocketcycle.examples.rpg.concerns.Character
	(*concerns.FundingRequest)(nil), // 4: rocketcycle.examples.rpg.concerns.FundingRequest
}
var file_edge_proto_depIdxs = []int32{
	0, // 0: rocketcycle.examples.rpg.edge.RpgService.ReadPlayer:input_type -> rocketcycle.examples.rpg.edge.RpgRequest
	2, // 1: rocketcycle.examples.rpg.edge.RpgService.CreatePlayer:input_type -> rocketcycle.examples.rpg.concerns.Player
	2, // 2: rocketcycle.examples.rpg.edge.RpgService.UpdatePlayer:input_type -> rocketcycle.examples.rpg.concerns.Player
	0, // 3: rocketcycle.examples.rpg.edge.RpgService.DeletePlayer:input_type -> rocketcycle.examples.rpg.edge.RpgRequest
	0, // 4: rocketcycle.examples.rpg.edge.RpgService.ReadCharacter:input_type -> rocketcycle.examples.rpg.edge.RpgRequest
	3, // 5: rocketcycle.examples.rpg.edge.RpgService.CreateCharacter:input_type -> rocketcycle.examples.rpg.concerns.Character
	3, // 6: rocketcycle.examples.rpg.edge.RpgService.UpdateCharacter:input_type -> rocketcycle.examples.rpg.concerns.Character
	0, // 7: rocketcycle.examples.rpg.edge.RpgService.DeleteCharacter:input_type -> rocketcycle.examples.rpg.edge.RpgRequest
	4, // 8: rocketcycle.examples.rpg.edge.RpgService.FundCharacter:input_type -> rocketcycle.examples.rpg.concerns.FundingRequest
	2, // 9: rocketcycle.examples.rpg.edge.RpgService.ReadPlayer:output_type -> rocketcycle.examples.rpg.concerns.Player
	2, // 10: rocketcycle.examples.rpg.edge.RpgService.CreatePlayer:output_type -> rocketcycle.examples.rpg.concerns.Player
	2, // 11: rocketcycle.examples.rpg.edge.RpgService.UpdatePlayer:output_type -> rocketcycle.examples.rpg.concerns.Player
	1, // 12: rocketcycle.examples.rpg.edge.RpgService.DeletePlayer:output_type -> rocketcycle.examples.rpg.edge.RpgResponse
	3, // 13: rocketcycle.examples.rpg.edge.RpgService.ReadCharacter:output_type -> rocketcycle.examples.rpg.concerns.Character
	3, // 14: rocketcycle.examples.rpg.edge.RpgService.CreateCharacter:output_type -> rocketcycle.examples.rpg.concerns.Character
	3, // 15: rocketcycle.examples.rpg.edge.RpgService.UpdateCharacter:output_type -> rocketcycle.examples.rpg.concerns.Character
	1, // 16: rocketcycle.examples.rpg.edge.RpgService.DeleteCharacter:output_type -> rocketcycle.examples.rpg.edge.RpgResponse
	3, // 17: rocketcycle.examples.rpg.edge.RpgService.FundCharacter:output_type -> rocketcycle.examples.rpg.concerns.Character
	9, // [9:18] is the sub-list for method output_type
	0, // [0:9] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_edge_proto_init() }
func file_edge_proto_init() {
	if File_edge_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_edge_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RpgRequest); i {
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
		file_edge_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RpgResponse); i {
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
			RawDescriptor: file_edge_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_edge_proto_goTypes,
		DependencyIndexes: file_edge_proto_depIdxs,
		MessageInfos:      file_edge_proto_msgTypes,
	}.Build()
	File_edge_proto = out.File
	file_edge_proto_rawDesc = nil
	file_edge_proto_goTypes = nil
	file_edge_proto_depIdxs = nil
}
