// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.17.3
// source: edge.proto

package edge

import (
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2/options"
	pb "github.com/lachlanorr/rocketcycle/examples/rpg/pb"
	rkcy "github.com/lachlanorr/rocketcycle/pkg/rkcy"
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

type PlayerResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Player  *pb.Player                `protobuf:"bytes,1,opt,name=player,proto3" json:"player,omitempty"`
	Related *pb.PlayerRelatedConcerns `protobuf:"bytes,2,opt,name=related,proto3" json:"related,omitempty"`
}

func (x *PlayerResponse) Reset() {
	*x = PlayerResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_edge_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PlayerResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PlayerResponse) ProtoMessage() {}

func (x *PlayerResponse) ProtoReflect() protoreflect.Message {
	mi := &file_edge_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PlayerResponse.ProtoReflect.Descriptor instead.
func (*PlayerResponse) Descriptor() ([]byte, []int) {
	return file_edge_proto_rawDescGZIP(), []int{2}
}

func (x *PlayerResponse) GetPlayer() *pb.Player {
	if x != nil {
		return x.Player
	}
	return nil
}

func (x *PlayerResponse) GetRelated() *pb.PlayerRelatedConcerns {
	if x != nil {
		return x.Related
	}
	return nil
}

type CharacterResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Character *pb.Character                `protobuf:"bytes,1,opt,name=character,proto3" json:"character,omitempty"`
	Related   *pb.CharacterRelatedConcerns `protobuf:"bytes,2,opt,name=related,proto3" json:"related,omitempty"`
}

func (x *CharacterResponse) Reset() {
	*x = CharacterResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_edge_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CharacterResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CharacterResponse) ProtoMessage() {}

func (x *CharacterResponse) ProtoReflect() protoreflect.Message {
	mi := &file_edge_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CharacterResponse.ProtoReflect.Descriptor instead.
func (*CharacterResponse) Descriptor() ([]byte, []int) {
	return file_edge_proto_rawDescGZIP(), []int{3}
}

func (x *CharacterResponse) GetCharacter() *pb.Character {
	if x != nil {
		return x.Character
	}
	return nil
}

func (x *CharacterResponse) GetRelated() *pb.CharacterRelatedConcerns {
	if x != nil {
		return x.Related
	}
	return nil
}

var File_edge_proto protoreflect.FileDescriptor

var file_edge_proto_rawDesc = []byte{
	0x0a, 0x0a, 0x65, 0x64, 0x67, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x04, 0x65, 0x64,
	0x67, 0x65, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61,
	0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x2d, 0x67, 0x65, 0x6e, 0x2d, 0x6f, 0x70, 0x65,
	0x6e, 0x61, 0x70, 0x69, 0x76, 0x32, 0x2f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x61,
	0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x10, 0x72, 0x6b, 0x63, 0x79, 0x2f, 0x61, 0x70, 0x65, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x0c, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x0f, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x13, 0x72, 0x65, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x68, 0x69, 0x70, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x72, 0x0a, 0x0a, 0x52, 0x70, 0x67, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x12, 0x64, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x42, 0x54, 0x92, 0x41, 0x51, 0x80, 0x01, 0x01, 0x8a, 0x01, 0x4b, 0x5b, 0x61, 0x2d, 0x66, 0x41,
	0x2d, 0x46, 0x30, 0x2d, 0x39, 0x5d, 0x7b, 0x38, 0x7d, 0x2d, 0x5b, 0x61, 0x2d, 0x66, 0x41, 0x2d,
	0x46, 0x30, 0x2d, 0x39, 0x5d, 0x7b, 0x34, 0x7d, 0x2d, 0x5b, 0x61, 0x2d, 0x66, 0x41, 0x2d, 0x46,
	0x30, 0x2d, 0x39, 0x5d, 0x7b, 0x34, 0x7d, 0x2d, 0x5b, 0x61, 0x2d, 0x66, 0x41, 0x2d, 0x46, 0x30,
	0x2d, 0x39, 0x5d, 0x7b, 0x34, 0x7d, 0x2d, 0x5b, 0x61, 0x2d, 0x66, 0x41, 0x2d, 0x46, 0x30, 0x2d,
	0x39, 0x5d, 0x7b, 0x31, 0x32, 0x7d, 0x52, 0x02, 0x69, 0x64, 0x22, 0x73, 0x0a, 0x0b, 0x52, 0x70,
	0x67, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x64, 0x0a, 0x02, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x54, 0x92, 0x41, 0x51, 0x80, 0x01, 0x01, 0x8a, 0x01, 0x4b,
	0x5b, 0x61, 0x2d, 0x66, 0x41, 0x2d, 0x46, 0x30, 0x2d, 0x39, 0x5d, 0x7b, 0x38, 0x7d, 0x2d, 0x5b,
	0x61, 0x2d, 0x66, 0x41, 0x2d, 0x46, 0x30, 0x2d, 0x39, 0x5d, 0x7b, 0x34, 0x7d, 0x2d, 0x5b, 0x61,
	0x2d, 0x66, 0x41, 0x2d, 0x46, 0x30, 0x2d, 0x39, 0x5d, 0x7b, 0x34, 0x7d, 0x2d, 0x5b, 0x61, 0x2d,
	0x66, 0x41, 0x2d, 0x46, 0x30, 0x2d, 0x39, 0x5d, 0x7b, 0x34, 0x7d, 0x2d, 0x5b, 0x61, 0x2d, 0x66,
	0x41, 0x2d, 0x46, 0x30, 0x2d, 0x39, 0x5d, 0x7b, 0x31, 0x32, 0x7d, 0x52, 0x02, 0x69, 0x64, 0x22,
	0x69, 0x0a, 0x0e, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x22, 0x0a, 0x06, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x0a, 0x2e, 0x70, 0x62, 0x2e, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x52, 0x06, 0x70,
	0x6c, 0x61, 0x79, 0x65, 0x72, 0x12, 0x33, 0x0a, 0x07, 0x72, 0x65, 0x6c, 0x61, 0x74, 0x65, 0x64,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x70, 0x62, 0x2e, 0x50, 0x6c, 0x61, 0x79,
	0x65, 0x72, 0x52, 0x65, 0x6c, 0x61, 0x74, 0x65, 0x64, 0x43, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e,
	0x73, 0x52, 0x07, 0x72, 0x65, 0x6c, 0x61, 0x74, 0x65, 0x64, 0x22, 0x78, 0x0a, 0x11, 0x43, 0x68,
	0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x2b, 0x0a, 0x09, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x70, 0x62, 0x2e, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65,
	0x72, 0x52, 0x09, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x12, 0x36, 0x0a, 0x07,
	0x72, 0x65, 0x6c, 0x61, 0x74, 0x65, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e,
	0x70, 0x62, 0x2e, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x52, 0x65, 0x6c, 0x61,
	0x74, 0x65, 0x64, 0x43, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x73, 0x52, 0x07, 0x72, 0x65, 0x6c,
	0x61, 0x74, 0x65, 0x64, 0x32, 0xba, 0x06, 0x0a, 0x0a, 0x52, 0x70, 0x67, 0x53, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x12, 0x52, 0x0a, 0x0a, 0x52, 0x65, 0x61, 0x64, 0x50, 0x6c, 0x61, 0x79, 0x65,
	0x72, 0x12, 0x10, 0x2e, 0x65, 0x64, 0x67, 0x65, 0x2e, 0x52, 0x70, 0x67, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x1a, 0x14, 0x2e, 0x65, 0x64, 0x67, 0x65, 0x2e, 0x50, 0x6c, 0x61, 0x79, 0x65,
	0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x1c, 0x82, 0xd3, 0xe4, 0x93, 0x02,
	0x16, 0x12, 0x14, 0x2f, 0x76, 0x31, 0x2f, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x72, 0x65,
	0x61, 0x64, 0x2f, 0x7b, 0x69, 0x64, 0x7d, 0x12, 0x44, 0x0a, 0x0c, 0x43, 0x72, 0x65, 0x61, 0x74,
	0x65, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x12, 0x0a, 0x2e, 0x70, 0x62, 0x2e, 0x50, 0x6c, 0x61,
	0x79, 0x65, 0x72, 0x1a, 0x0a, 0x2e, 0x70, 0x62, 0x2e, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x22,
	0x1c, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x16, 0x22, 0x11, 0x2f, 0x76, 0x31, 0x2f, 0x70, 0x6c, 0x61,
	0x79, 0x65, 0x72, 0x2f, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x3a, 0x01, 0x2a, 0x12, 0x44, 0x0a,
	0x0c, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x12, 0x0a, 0x2e,
	0x70, 0x62, 0x2e, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x1a, 0x0a, 0x2e, 0x70, 0x62, 0x2e, 0x50,
	0x6c, 0x61, 0x79, 0x65, 0x72, 0x22, 0x1c, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x16, 0x22, 0x11, 0x2f,
	0x76, 0x31, 0x2f, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65,
	0x3a, 0x01, 0x2a, 0x12, 0x4c, 0x0a, 0x0c, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x50, 0x6c, 0x61,
	0x79, 0x65, 0x72, 0x12, 0x10, 0x2e, 0x65, 0x64, 0x67, 0x65, 0x2e, 0x52, 0x70, 0x67, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x0a, 0x2e, 0x70, 0x62, 0x2e, 0x50, 0x6c, 0x61, 0x79, 0x65,
	0x72, 0x22, 0x1e, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x18, 0x22, 0x16, 0x2f, 0x76, 0x31, 0x2f, 0x70,
	0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x64, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x2f, 0x7b, 0x69, 0x64,
	0x7d, 0x12, 0x5b, 0x0a, 0x0d, 0x52, 0x65, 0x61, 0x64, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74,
	0x65, 0x72, 0x12, 0x10, 0x2e, 0x65, 0x64, 0x67, 0x65, 0x2e, 0x52, 0x70, 0x67, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x17, 0x2e, 0x65, 0x64, 0x67, 0x65, 0x2e, 0x43, 0x68, 0x61, 0x72,
	0x61, 0x63, 0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x1f, 0x82,
	0xd3, 0xe4, 0x93, 0x02, 0x19, 0x12, 0x17, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x68, 0x61, 0x72, 0x61,
	0x63, 0x74, 0x65, 0x72, 0x2f, 0x72, 0x65, 0x61, 0x64, 0x2f, 0x7b, 0x69, 0x64, 0x7d, 0x12, 0x50,
	0x0a, 0x0f, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65,
	0x72, 0x12, 0x0d, 0x2e, 0x70, 0x62, 0x2e, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72,
	0x1a, 0x0d, 0x2e, 0x70, 0x62, 0x2e, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x22,
	0x1f, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x19, 0x22, 0x14, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x68, 0x61,
	0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x2f, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x3a, 0x01, 0x2a,
	0x12, 0x50, 0x0a, 0x0f, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63,
	0x74, 0x65, 0x72, 0x12, 0x0d, 0x2e, 0x70, 0x62, 0x2e, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74,
	0x65, 0x72, 0x1a, 0x0d, 0x2e, 0x70, 0x62, 0x2e, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65,
	0x72, 0x22, 0x1f, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x19, 0x22, 0x14, 0x2f, 0x76, 0x31, 0x2f, 0x63,
	0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x2f, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x3a,
	0x01, 0x2a, 0x12, 0x55, 0x0a, 0x0f, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x43, 0x68, 0x61, 0x72,
	0x61, 0x63, 0x74, 0x65, 0x72, 0x12, 0x10, 0x2e, 0x65, 0x64, 0x67, 0x65, 0x2e, 0x52, 0x70, 0x67,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x0d, 0x2e, 0x70, 0x62, 0x2e, 0x43, 0x68, 0x61,
	0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x22, 0x21, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x1b, 0x22, 0x19,
	0x2f, 0x76, 0x31, 0x2f, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x2f, 0x64, 0x65,
	0x6c, 0x65, 0x74, 0x65, 0x2f, 0x7b, 0x69, 0x64, 0x7d, 0x12, 0x51, 0x0a, 0x0d, 0x46, 0x75, 0x6e,
	0x64, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x12, 0x12, 0x2e, 0x70, 0x62, 0x2e,
	0x46, 0x75, 0x6e, 0x64, 0x69, 0x6e, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x0d,
	0x2e, 0x70, 0x62, 0x2e, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x22, 0x1d, 0x82,
	0xd3, 0xe4, 0x93, 0x02, 0x17, 0x22, 0x12, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x68, 0x61, 0x72, 0x61,
	0x63, 0x74, 0x65, 0x72, 0x2f, 0x66, 0x75, 0x6e, 0x64, 0x3a, 0x01, 0x2a, 0x12, 0x53, 0x0a, 0x0c,
	0x43, 0x6f, 0x6e, 0x64, 0x75, 0x63, 0x74, 0x54, 0x72, 0x61, 0x64, 0x65, 0x12, 0x10, 0x2e, 0x70,
	0x62, 0x2e, 0x54, 0x72, 0x61, 0x64, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x0a,
	0x2e, 0x72, 0x6b, 0x63, 0x79, 0x2e, 0x56, 0x6f, 0x69, 0x64, 0x22, 0x25, 0x82, 0xd3, 0xe4, 0x93,
	0x02, 0x1f, 0x22, 0x1a, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65,
	0x72, 0x2f, 0x63, 0x6f, 0x6e, 0x64, 0x75, 0x63, 0x74, 0x54, 0x72, 0x61, 0x64, 0x65, 0x3a, 0x01,
	0x2a, 0x42, 0x35, 0x5a, 0x33, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x6c, 0x61, 0x63, 0x68, 0x6c, 0x61, 0x6e, 0x6f, 0x72, 0x72, 0x2f, 0x72, 0x6f, 0x63, 0x6b, 0x65,
	0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2f, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2f,
	0x72, 0x70, 0x67, 0x2f, 0x65, 0x64, 0x67, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
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

var file_edge_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_edge_proto_goTypes = []interface{}{
	(*RpgRequest)(nil),                  // 0: edge.RpgRequest
	(*RpgResponse)(nil),                 // 1: edge.RpgResponse
	(*PlayerResponse)(nil),              // 2: edge.PlayerResponse
	(*CharacterResponse)(nil),           // 3: edge.CharacterResponse
	(*pb.Player)(nil),                   // 4: pb.Player
	(*pb.PlayerRelatedConcerns)(nil),    // 5: pb.PlayerRelatedConcerns
	(*pb.Character)(nil),                // 6: pb.Character
	(*pb.CharacterRelatedConcerns)(nil), // 7: pb.CharacterRelatedConcerns
	(*pb.FundingRequest)(nil),           // 8: pb.FundingRequest
	(*pb.TradeRequest)(nil),             // 9: pb.TradeRequest
	(*rkcy.Void)(nil),                   // 10: rkcy.Void
}
var file_edge_proto_depIdxs = []int32{
	4,  // 0: edge.PlayerResponse.player:type_name -> pb.Player
	5,  // 1: edge.PlayerResponse.related:type_name -> pb.PlayerRelatedConcerns
	6,  // 2: edge.CharacterResponse.character:type_name -> pb.Character
	7,  // 3: edge.CharacterResponse.related:type_name -> pb.CharacterRelatedConcerns
	0,  // 4: edge.RpgService.ReadPlayer:input_type -> edge.RpgRequest
	4,  // 5: edge.RpgService.CreatePlayer:input_type -> pb.Player
	4,  // 6: edge.RpgService.UpdatePlayer:input_type -> pb.Player
	0,  // 7: edge.RpgService.DeletePlayer:input_type -> edge.RpgRequest
	0,  // 8: edge.RpgService.ReadCharacter:input_type -> edge.RpgRequest
	6,  // 9: edge.RpgService.CreateCharacter:input_type -> pb.Character
	6,  // 10: edge.RpgService.UpdateCharacter:input_type -> pb.Character
	0,  // 11: edge.RpgService.DeleteCharacter:input_type -> edge.RpgRequest
	8,  // 12: edge.RpgService.FundCharacter:input_type -> pb.FundingRequest
	9,  // 13: edge.RpgService.ConductTrade:input_type -> pb.TradeRequest
	2,  // 14: edge.RpgService.ReadPlayer:output_type -> edge.PlayerResponse
	4,  // 15: edge.RpgService.CreatePlayer:output_type -> pb.Player
	4,  // 16: edge.RpgService.UpdatePlayer:output_type -> pb.Player
	4,  // 17: edge.RpgService.DeletePlayer:output_type -> pb.Player
	3,  // 18: edge.RpgService.ReadCharacter:output_type -> edge.CharacterResponse
	6,  // 19: edge.RpgService.CreateCharacter:output_type -> pb.Character
	6,  // 20: edge.RpgService.UpdateCharacter:output_type -> pb.Character
	6,  // 21: edge.RpgService.DeleteCharacter:output_type -> pb.Character
	6,  // 22: edge.RpgService.FundCharacter:output_type -> pb.Character
	10, // 23: edge.RpgService.ConductTrade:output_type -> rkcy.Void
	14, // [14:24] is the sub-list for method output_type
	4,  // [4:14] is the sub-list for method input_type
	4,  // [4:4] is the sub-list for extension type_name
	4,  // [4:4] is the sub-list for extension extendee
	0,  // [0:4] is the sub-list for field type_name
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
		file_edge_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PlayerResponse); i {
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
		file_edge_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CharacterResponse); i {
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
			NumMessages:   4,
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
