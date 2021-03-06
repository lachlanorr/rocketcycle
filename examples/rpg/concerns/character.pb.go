// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.13.0
// source: character.proto

package concerns

import (
	proto "github.com/golang/protobuf/proto"
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

type Character struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id       string              `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	PlayerId string              `protobuf:"bytes,2,opt,name=player_id,json=playerId,proto3" json:"player_id,omitempty"`
	Fullname string              `protobuf:"bytes,3,opt,name=fullname,proto3" json:"fullname,omitempty"`
	Active   bool                `protobuf:"varint,4,opt,name=active,proto3" json:"active,omitempty"`
	Currency *Character_Currency `protobuf:"bytes,5,opt,name=currency,proto3" json:"currency,omitempty"`
	Items    []*Character_Item   `protobuf:"bytes,6,rep,name=items,proto3" json:"items,omitempty"`
}

func (x *Character) Reset() {
	*x = Character{}
	if protoimpl.UnsafeEnabled {
		mi := &file_character_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Character) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Character) ProtoMessage() {}

func (x *Character) ProtoReflect() protoreflect.Message {
	mi := &file_character_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Character.ProtoReflect.Descriptor instead.
func (*Character) Descriptor() ([]byte, []int) {
	return file_character_proto_rawDescGZIP(), []int{0}
}

func (x *Character) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Character) GetPlayerId() string {
	if x != nil {
		return x.PlayerId
	}
	return ""
}

func (x *Character) GetFullname() string {
	if x != nil {
		return x.Fullname
	}
	return ""
}

func (x *Character) GetActive() bool {
	if x != nil {
		return x.Active
	}
	return false
}

func (x *Character) GetCurrency() *Character_Currency {
	if x != nil {
		return x.Currency
	}
	return nil
}

func (x *Character) GetItems() []*Character_Item {
	if x != nil {
		return x.Items
	}
	return nil
}

type FundingRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CharacterId string              `protobuf:"bytes,1,opt,name=character_id,json=characterId,proto3" json:"character_id,omitempty"`
	Currency    *Character_Currency `protobuf:"bytes,2,opt,name=currency,proto3" json:"currency,omitempty"`
}

func (x *FundingRequest) Reset() {
	*x = FundingRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_character_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FundingRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FundingRequest) ProtoMessage() {}

func (x *FundingRequest) ProtoReflect() protoreflect.Message {
	mi := &file_character_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FundingRequest.ProtoReflect.Descriptor instead.
func (*FundingRequest) Descriptor() ([]byte, []int) {
	return file_character_proto_rawDescGZIP(), []int{1}
}

func (x *FundingRequest) GetCharacterId() string {
	if x != nil {
		return x.CharacterId
	}
	return ""
}

func (x *FundingRequest) GetCurrency() *Character_Currency {
	if x != nil {
		return x.Currency
	}
	return nil
}

type Character_Currency struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Gold      int32 `protobuf:"varint,1,opt,name=gold,proto3" json:"gold,omitempty"`
	Faction_0 int32 `protobuf:"varint,2,opt,name=faction_0,json=faction0,proto3" json:"faction_0,omitempty"`
	Faction_1 int32 `protobuf:"varint,3,opt,name=faction_1,json=faction1,proto3" json:"faction_1,omitempty"`
	Faction_2 int32 `protobuf:"varint,4,opt,name=faction_2,json=faction2,proto3" json:"faction_2,omitempty"`
}

func (x *Character_Currency) Reset() {
	*x = Character_Currency{}
	if protoimpl.UnsafeEnabled {
		mi := &file_character_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Character_Currency) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Character_Currency) ProtoMessage() {}

func (x *Character_Currency) ProtoReflect() protoreflect.Message {
	mi := &file_character_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Character_Currency.ProtoReflect.Descriptor instead.
func (*Character_Currency) Descriptor() ([]byte, []int) {
	return file_character_proto_rawDescGZIP(), []int{0, 0}
}

func (x *Character_Currency) GetGold() int32 {
	if x != nil {
		return x.Gold
	}
	return 0
}

func (x *Character_Currency) GetFaction_0() int32 {
	if x != nil {
		return x.Faction_0
	}
	return 0
}

func (x *Character_Currency) GetFaction_1() int32 {
	if x != nil {
		return x.Faction_1
	}
	return 0
}

func (x *Character_Currency) GetFaction_2() int32 {
	if x != nil {
		return x.Faction_2
	}
	return 0
}

type Character_Item struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id          string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Description string `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
}

func (x *Character_Item) Reset() {
	*x = Character_Item{}
	if protoimpl.UnsafeEnabled {
		mi := &file_character_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Character_Item) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Character_Item) ProtoMessage() {}

func (x *Character_Item) ProtoReflect() protoreflect.Message {
	mi := &file_character_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Character_Item.ProtoReflect.Descriptor instead.
func (*Character_Item) Descriptor() ([]byte, []int) {
	return file_character_proto_rawDescGZIP(), []int{0, 1}
}

func (x *Character_Item) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Character_Item) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

var File_character_proto protoreflect.FileDescriptor

var file_character_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x21, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65,
	0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70, 0x67, 0x2e, 0x63, 0x6f, 0x6e, 0x63,
	0x65, 0x72, 0x6e, 0x73, 0x22, 0xb9, 0x03, 0x0a, 0x09, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74,
	0x65, 0x72, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02,
	0x69, 0x64, 0x12, 0x1b, 0x0a, 0x09, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x49, 0x64, 0x12,
	0x1a, 0x0a, 0x08, 0x66, 0x75, 0x6c, 0x6c, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x08, 0x66, 0x75, 0x6c, 0x6c, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x61,
	0x63, 0x74, 0x69, 0x76, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x06, 0x61, 0x63, 0x74,
	0x69, 0x76, 0x65, 0x12, 0x51, 0x0a, 0x08, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x63, 0x79, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x35, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79,
	0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70, 0x67,
	0x2e, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x73, 0x2e, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63,
	0x74, 0x65, 0x72, 0x2e, 0x43, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x63, 0x79, 0x52, 0x08, 0x63, 0x75,
	0x72, 0x72, 0x65, 0x6e, 0x63, 0x79, 0x12, 0x47, 0x0a, 0x05, 0x69, 0x74, 0x65, 0x6d, 0x73, 0x18,
	0x06, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x31, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79,
	0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70, 0x67,
	0x2e, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x73, 0x2e, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63,
	0x74, 0x65, 0x72, 0x2e, 0x49, 0x74, 0x65, 0x6d, 0x52, 0x05, 0x69, 0x74, 0x65, 0x6d, 0x73, 0x1a,
	0x75, 0x0a, 0x08, 0x43, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x63, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x67,
	0x6f, 0x6c, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x04, 0x67, 0x6f, 0x6c, 0x64, 0x12,
	0x1b, 0x0a, 0x09, 0x66, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x30, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x05, 0x52, 0x08, 0x66, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x30, 0x12, 0x1b, 0x0a, 0x09,
	0x66, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x31, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x08, 0x66, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x31, 0x12, 0x1b, 0x0a, 0x09, 0x66, 0x61, 0x63,
	0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x32, 0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x66, 0x61,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x32, 0x1a, 0x38, 0x0a, 0x04, 0x49, 0x74, 0x65, 0x6d, 0x12, 0x0e,
	0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x20,
	0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e,
	0x22, 0x86, 0x01, 0x0a, 0x0e, 0x46, 0x75, 0x6e, 0x64, 0x69, 0x6e, 0x67, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x21, 0x0a, 0x0c, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72,
	0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x63, 0x68, 0x61, 0x72, 0x61,
	0x63, 0x74, 0x65, 0x72, 0x49, 0x64, 0x12, 0x51, 0x0a, 0x08, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e,
	0x63, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x35, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65,
	0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e,
	0x72, 0x70, 0x67, 0x2e, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x73, 0x2e, 0x43, 0x68, 0x61,
	0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x2e, 0x43, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x63, 0x79, 0x52,
	0x08, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x63, 0x79, 0x32, 0xdb, 0x02, 0x0a, 0x11, 0x43, 0x68,
	0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x73, 0x12,
	0x67, 0x0a, 0x04, 0x46, 0x75, 0x6e, 0x64, 0x12, 0x31, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74,
	0x63, 0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72,
	0x70, 0x67, 0x2e, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x73, 0x2e, 0x46, 0x75, 0x6e, 0x64,
	0x69, 0x6e, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2c, 0x2e, 0x72, 0x6f, 0x63,
	0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65,
	0x73, 0x2e, 0x72, 0x70, 0x67, 0x2e, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x73, 0x2e, 0x43,
	0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x12, 0x6d, 0x0a, 0x0a, 0x44, 0x65, 0x62, 0x69,
	0x74, 0x46, 0x75, 0x6e, 0x64, 0x73, 0x12, 0x31, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63,
	0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70,
	0x67, 0x2e, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x73, 0x2e, 0x46, 0x75, 0x6e, 0x64, 0x69,
	0x6e, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2c, 0x2e, 0x72, 0x6f, 0x63, 0x6b,
	0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73,
	0x2e, 0x72, 0x70, 0x67, 0x2e, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x73, 0x2e, 0x43, 0x68,
	0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x12, 0x6e, 0x0a, 0x0b, 0x43, 0x72, 0x65, 0x64, 0x69,
	0x74, 0x46, 0x75, 0x6e, 0x64, 0x73, 0x12, 0x31, 0x2e, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63,
	0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2e, 0x72, 0x70,
	0x67, 0x2e, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x73, 0x2e, 0x46, 0x75, 0x6e, 0x64, 0x69,
	0x6e, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2c, 0x2e, 0x72, 0x6f, 0x63, 0x6b,
	0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73,
	0x2e, 0x72, 0x70, 0x67, 0x2e, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x73, 0x2e, 0x43, 0x68,
	0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x42, 0x39, 0x5a, 0x37, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6c, 0x61, 0x63, 0x68, 0x6c, 0x61, 0x6e, 0x6f, 0x72, 0x72,
	0x2f, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2f, 0x65, 0x78, 0x61,
	0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2f, 0x72, 0x70, 0x67, 0x2f, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72,
	0x6e, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_character_proto_rawDescOnce sync.Once
	file_character_proto_rawDescData = file_character_proto_rawDesc
)

func file_character_proto_rawDescGZIP() []byte {
	file_character_proto_rawDescOnce.Do(func() {
		file_character_proto_rawDescData = protoimpl.X.CompressGZIP(file_character_proto_rawDescData)
	})
	return file_character_proto_rawDescData
}

var file_character_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_character_proto_goTypes = []interface{}{
	(*Character)(nil),          // 0: rocketcycle.examples.rpg.concerns.Character
	(*FundingRequest)(nil),     // 1: rocketcycle.examples.rpg.concerns.FundingRequest
	(*Character_Currency)(nil), // 2: rocketcycle.examples.rpg.concerns.Character.Currency
	(*Character_Item)(nil),     // 3: rocketcycle.examples.rpg.concerns.Character.Item
}
var file_character_proto_depIdxs = []int32{
	2, // 0: rocketcycle.examples.rpg.concerns.Character.currency:type_name -> rocketcycle.examples.rpg.concerns.Character.Currency
	3, // 1: rocketcycle.examples.rpg.concerns.Character.items:type_name -> rocketcycle.examples.rpg.concerns.Character.Item
	2, // 2: rocketcycle.examples.rpg.concerns.FundingRequest.currency:type_name -> rocketcycle.examples.rpg.concerns.Character.Currency
	1, // 3: rocketcycle.examples.rpg.concerns.CharacterCommands.Fund:input_type -> rocketcycle.examples.rpg.concerns.FundingRequest
	1, // 4: rocketcycle.examples.rpg.concerns.CharacterCommands.DebitFunds:input_type -> rocketcycle.examples.rpg.concerns.FundingRequest
	1, // 5: rocketcycle.examples.rpg.concerns.CharacterCommands.CreditFunds:input_type -> rocketcycle.examples.rpg.concerns.FundingRequest
	0, // 6: rocketcycle.examples.rpg.concerns.CharacterCommands.Fund:output_type -> rocketcycle.examples.rpg.concerns.Character
	0, // 7: rocketcycle.examples.rpg.concerns.CharacterCommands.DebitFunds:output_type -> rocketcycle.examples.rpg.concerns.Character
	0, // 8: rocketcycle.examples.rpg.concerns.CharacterCommands.CreditFunds:output_type -> rocketcycle.examples.rpg.concerns.Character
	6, // [6:9] is the sub-list for method output_type
	3, // [3:6] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_character_proto_init() }
func file_character_proto_init() {
	if File_character_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_character_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Character); i {
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
		file_character_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FundingRequest); i {
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
		file_character_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Character_Currency); i {
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
		file_character_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Character_Item); i {
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
			RawDescriptor: file_character_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_character_proto_goTypes,
		DependencyIndexes: file_character_proto_depIdxs,
		MessageInfos:      file_character_proto_msgTypes,
	}.Build()
	File_character_proto = out.File
	file_character_proto_rawDesc = nil
	file_character_proto_goTypes = nil
	file_character_proto_depIdxs = nil
}
