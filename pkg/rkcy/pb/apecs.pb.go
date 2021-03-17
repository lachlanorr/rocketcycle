// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.13.0
// source: apecs.proto

package pb

import (
	proto "github.com/golang/protobuf/proto"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
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

type Status int32

const (
	Status_PENDING  Status = 0
	Status_COMPLETE Status = 1
	Status_REVERTED Status = 2
	Status_ERROR    Status = 3
)

// Enum value maps for Status.
var (
	Status_name = map[int32]string{
		0: "PENDING",
		1: "COMPLETE",
		2: "REVERTED",
		3: "ERROR",
	}
	Status_value = map[string]int32{
		"PENDING":  0,
		"COMPLETE": 1,
		"REVERTED": 2,
		"ERROR":    3,
	}
)

func (x Status) Enum() *Status {
	p := new(Status)
	*p = x
	return p
}

func (x Status) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Status) Descriptor() protoreflect.EnumDescriptor {
	return file_apecs_proto_enumTypes[0].Descriptor()
}

func (Status) Type() protoreflect.EnumType {
	return &file_apecs_proto_enumTypes[0]
}

func (x Status) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Status.Descriptor instead.
func (Status) EnumDescriptor() ([]byte, []int) {
	return file_apecs_proto_rawDescGZIP(), []int{0}
}

type ApecsTxn_Dir int32

const (
	ApecsTxn_FORWARD ApecsTxn_Dir = 0
	ApecsTxn_REVERSE ApecsTxn_Dir = 1
)

// Enum value maps for ApecsTxn_Dir.
var (
	ApecsTxn_Dir_name = map[int32]string{
		0: "FORWARD",
		1: "REVERSE",
	}
	ApecsTxn_Dir_value = map[string]int32{
		"FORWARD": 0,
		"REVERSE": 1,
	}
)

func (x ApecsTxn_Dir) Enum() *ApecsTxn_Dir {
	p := new(ApecsTxn_Dir)
	*p = x
	return p
}

func (x ApecsTxn_Dir) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ApecsTxn_Dir) Descriptor() protoreflect.EnumDescriptor {
	return file_apecs_proto_enumTypes[1].Descriptor()
}

func (ApecsTxn_Dir) Type() protoreflect.EnumType {
	return &file_apecs_proto_enumTypes[1]
}

func (x ApecsTxn_Dir) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ApecsTxn_Dir.Descriptor instead.
func (ApecsTxn_Dir) EnumDescriptor() ([]byte, []int) {
	return file_apecs_proto_rawDescGZIP(), []int{0, 0}
}

type ApecsStorageRequest_Op int32

const (
	ApecsStorageRequest_GET    ApecsStorageRequest_Op = 0
	ApecsStorageRequest_CREATE ApecsStorageRequest_Op = 1
	ApecsStorageRequest_UPDATE ApecsStorageRequest_Op = 2
	ApecsStorageRequest_DELETE ApecsStorageRequest_Op = 3
)

// Enum value maps for ApecsStorageRequest_Op.
var (
	ApecsStorageRequest_Op_name = map[int32]string{
		0: "GET",
		1: "CREATE",
		2: "UPDATE",
		3: "DELETE",
	}
	ApecsStorageRequest_Op_value = map[string]int32{
		"GET":    0,
		"CREATE": 1,
		"UPDATE": 2,
		"DELETE": 3,
	}
)

func (x ApecsStorageRequest_Op) Enum() *ApecsStorageRequest_Op {
	p := new(ApecsStorageRequest_Op)
	*p = x
	return p
}

func (x ApecsStorageRequest_Op) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ApecsStorageRequest_Op) Descriptor() protoreflect.EnumDescriptor {
	return file_apecs_proto_enumTypes[2].Descriptor()
}

func (ApecsStorageRequest_Op) Type() protoreflect.EnumType {
	return &file_apecs_proto_enumTypes[2]
}

func (x ApecsStorageRequest_Op) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ApecsStorageRequest_Op.Descriptor instead.
func (ApecsStorageRequest_Op) EnumDescriptor() ([]byte, []int) {
	return file_apecs_proto_rawDescGZIP(), []int{3, 0}
}

type ApecsStorageResult_Code int32

const (
	ApecsStorageResult_SUCCESS        ApecsStorageResult_Code = 0
	ApecsStorageResult_INTERNAL       ApecsStorageResult_Code = 1
	ApecsStorageResult_MARSHAL_FAILED ApecsStorageResult_Code = 2
	ApecsStorageResult_NOT_FOUND      ApecsStorageResult_Code = 3
	ApecsStorageResult_CONNECTION     ApecsStorageResult_Code = 4
)

// Enum value maps for ApecsStorageResult_Code.
var (
	ApecsStorageResult_Code_name = map[int32]string{
		0: "SUCCESS",
		1: "INTERNAL",
		2: "MARSHAL_FAILED",
		3: "NOT_FOUND",
		4: "CONNECTION",
	}
	ApecsStorageResult_Code_value = map[string]int32{
		"SUCCESS":        0,
		"INTERNAL":       1,
		"MARSHAL_FAILED": 2,
		"NOT_FOUND":      3,
		"CONNECTION":     4,
	}
)

func (x ApecsStorageResult_Code) Enum() *ApecsStorageResult_Code {
	p := new(ApecsStorageResult_Code)
	*p = x
	return p
}

func (x ApecsStorageResult_Code) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ApecsStorageResult_Code) Descriptor() protoreflect.EnumDescriptor {
	return file_apecs_proto_enumTypes[3].Descriptor()
}

func (ApecsStorageResult_Code) Type() protoreflect.EnumType {
	return &file_apecs_proto_enumTypes[3]
}

func (x ApecsStorageResult_Code) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ApecsStorageResult_Code.Descriptor instead.
func (ApecsStorageResult_Code) EnumDescriptor() ([]byte, []int) {
	return file_apecs_proto_rawDescGZIP(), []int{4, 0}
}

type ApecsTxn struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Uid            string           `protobuf:"bytes,1,opt,name=uid,proto3" json:"uid,omitempty"`                                        // uuid of txn used for tracing and reporting
	Direction      ApecsTxn_Dir     `protobuf:"varint,2,opt,name=direction,proto3,enum=rkcy.pb.ApecsTxn_Dir" json:"direction,omitempty"` // starts in forward, can potentially go to reverse if the transaction is reversible
	CanRevert      bool             `protobuf:"varint,3,opt,name=can_revert,json=canRevert,proto3" json:"can_revert,omitempty"`          // whether or not this transaction can be rolled back.
	ForwardSteps   []*ApecsTxn_Step `protobuf:"bytes,4,rep,name=forward_steps,json=forwardSteps,proto3" json:"forward_steps,omitempty"`  // filled upon creation with forward steps
	ReverseSteps   []*ApecsTxn_Step `protobuf:"bytes,5,rep,name=reverse_steps,json=reverseSteps,proto3" json:"reverse_steps,omitempty"`  // upon an error in a "can_revert==true" transaction, this gets filled with the right rollback steps. Separatiing reverse from forward steps preserves the history for review of the nature of the failure.
	ResponseTarget *ResponseTarget  `protobuf:"bytes,6,opt,name=response_target,json=responseTarget,proto3" json:"response_target,omitempty"`
}

func (x *ApecsTxn) Reset() {
	*x = ApecsTxn{}
	if protoimpl.UnsafeEnabled {
		mi := &file_apecs_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ApecsTxn) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ApecsTxn) ProtoMessage() {}

func (x *ApecsTxn) ProtoReflect() protoreflect.Message {
	mi := &file_apecs_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ApecsTxn.ProtoReflect.Descriptor instead.
func (*ApecsTxn) Descriptor() ([]byte, []int) {
	return file_apecs_proto_rawDescGZIP(), []int{0}
}

func (x *ApecsTxn) GetUid() string {
	if x != nil {
		return x.Uid
	}
	return ""
}

func (x *ApecsTxn) GetDirection() ApecsTxn_Dir {
	if x != nil {
		return x.Direction
	}
	return ApecsTxn_FORWARD
}

func (x *ApecsTxn) GetCanRevert() bool {
	if x != nil {
		return x.CanRevert
	}
	return false
}

func (x *ApecsTxn) GetForwardSteps() []*ApecsTxn_Step {
	if x != nil {
		return x.ForwardSteps
	}
	return nil
}

func (x *ApecsTxn) GetReverseSteps() []*ApecsTxn_Step {
	if x != nil {
		return x.ReverseSteps
	}
	return nil
}

func (x *ApecsTxn) GetResponseTarget() *ResponseTarget {
	if x != nil {
		return x.ResponseTarget
	}
	return nil
}

type LogEvent struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Code int32  `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Msg  string `protobuf:"bytes,2,opt,name=msg,proto3" json:"msg,omitempty"`
}

func (x *LogEvent) Reset() {
	*x = LogEvent{}
	if protoimpl.UnsafeEnabled {
		mi := &file_apecs_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LogEvent) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LogEvent) ProtoMessage() {}

func (x *LogEvent) ProtoReflect() protoreflect.Message {
	mi := &file_apecs_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LogEvent.ProtoReflect.Descriptor instead.
func (*LogEvent) Descriptor() ([]byte, []int) {
	return file_apecs_proto_rawDescGZIP(), []int{1}
}

func (x *LogEvent) GetCode() int32 {
	if x != nil {
		return x.Code
	}
	return 0
}

func (x *LogEvent) GetMsg() string {
	if x != nil {
		return x.Msg
	}
	return ""
}

type MostRecentOffset struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Generation int32 `protobuf:"varint,1,opt,name=generation,proto3" json:"generation,omitempty"`
	Offset     int64 `protobuf:"varint,2,opt,name=offset,proto3" json:"offset,omitempty"`
}

func (x *MostRecentOffset) Reset() {
	*x = MostRecentOffset{}
	if protoimpl.UnsafeEnabled {
		mi := &file_apecs_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MostRecentOffset) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MostRecentOffset) ProtoMessage() {}

func (x *MostRecentOffset) ProtoReflect() protoreflect.Message {
	mi := &file_apecs_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MostRecentOffset.ProtoReflect.Descriptor instead.
func (*MostRecentOffset) Descriptor() ([]byte, []int) {
	return file_apecs_proto_rawDescGZIP(), []int{2}
}

func (x *MostRecentOffset) GetGeneration() int32 {
	if x != nil {
		return x.Generation
	}
	return 0
}

func (x *MostRecentOffset) GetOffset() int64 {
	if x != nil {
		return x.Offset
	}
	return 0
}

type ApecsStorageRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Uid            string                 `protobuf:"bytes,1,opt,name=uid,proto3" json:"uid,omitempty"`
	ConcernName    string                 `protobuf:"bytes,2,opt,name=concern_name,json=concernName,proto3" json:"concern_name,omitempty"`
	Op             ApecsStorageRequest_Op `protobuf:"varint,3,opt,name=op,proto3,enum=rkcy.pb.ApecsStorageRequest_Op" json:"op,omitempty"`
	Key            string                 `protobuf:"bytes,4,opt,name=key,proto3" json:"key,omitempty"`
	Payload        []byte                 `protobuf:"bytes,5,opt,name=payload,proto3" json:"payload,omitempty"`
	ResponseTarget *ResponseTarget        `protobuf:"bytes,6,opt,name=response_target,json=responseTarget,proto3" json:"response_target,omitempty"`
}

func (x *ApecsStorageRequest) Reset() {
	*x = ApecsStorageRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_apecs_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ApecsStorageRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ApecsStorageRequest) ProtoMessage() {}

func (x *ApecsStorageRequest) ProtoReflect() protoreflect.Message {
	mi := &file_apecs_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ApecsStorageRequest.ProtoReflect.Descriptor instead.
func (*ApecsStorageRequest) Descriptor() ([]byte, []int) {
	return file_apecs_proto_rawDescGZIP(), []int{3}
}

func (x *ApecsStorageRequest) GetUid() string {
	if x != nil {
		return x.Uid
	}
	return ""
}

func (x *ApecsStorageRequest) GetConcernName() string {
	if x != nil {
		return x.ConcernName
	}
	return ""
}

func (x *ApecsStorageRequest) GetOp() ApecsStorageRequest_Op {
	if x != nil {
		return x.Op
	}
	return ApecsStorageRequest_GET
}

func (x *ApecsStorageRequest) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *ApecsStorageRequest) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *ApecsStorageRequest) GetResponseTarget() *ResponseTarget {
	if x != nil {
		return x.ResponseTarget
	}
	return nil
}

type ApecsStorageResult struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Uid     string                  `protobuf:"bytes,1,opt,name=uid,proto3" json:"uid,omitempty"`
	Code    ApecsStorageResult_Code `protobuf:"varint,2,opt,name=code,proto3,enum=rkcy.pb.ApecsStorageResult_Code" json:"code,omitempty"`
	Payload []byte                  `protobuf:"bytes,3,opt,name=payload,proto3" json:"payload,omitempty"`
	Mro     *MostRecentOffset       `protobuf:"bytes,4,opt,name=mro,proto3" json:"mro,omitempty"`
}

func (x *ApecsStorageResult) Reset() {
	*x = ApecsStorageResult{}
	if protoimpl.UnsafeEnabled {
		mi := &file_apecs_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ApecsStorageResult) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ApecsStorageResult) ProtoMessage() {}

func (x *ApecsStorageResult) ProtoReflect() protoreflect.Message {
	mi := &file_apecs_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ApecsStorageResult.ProtoReflect.Descriptor instead.
func (*ApecsStorageResult) Descriptor() ([]byte, []int) {
	return file_apecs_proto_rawDescGZIP(), []int{4}
}

func (x *ApecsStorageResult) GetUid() string {
	if x != nil {
		return x.Uid
	}
	return ""
}

func (x *ApecsStorageResult) GetCode() ApecsStorageResult_Code {
	if x != nil {
		return x.Code
	}
	return ApecsStorageResult_SUCCESS
}

func (x *ApecsStorageResult) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *ApecsStorageResult) GetMro() *MostRecentOffset {
	if x != nil {
		return x.Mro
	}
	return nil
}

type ResponseTarget struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ConcernName string `protobuf:"bytes,1,opt,name=concern_name,json=concernName,proto3" json:"concern_name,omitempty"`
	TopicName   string `protobuf:"bytes,2,opt,name=topic_name,json=topicName,proto3" json:"topic_name,omitempty"`
	Partition   int32  `protobuf:"varint,3,opt,name=partition,proto3" json:"partition,omitempty"`
}

func (x *ResponseTarget) Reset() {
	*x = ResponseTarget{}
	if protoimpl.UnsafeEnabled {
		mi := &file_apecs_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ResponseTarget) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ResponseTarget) ProtoMessage() {}

func (x *ResponseTarget) ProtoReflect() protoreflect.Message {
	mi := &file_apecs_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ResponseTarget.ProtoReflect.Descriptor instead.
func (*ResponseTarget) Descriptor() ([]byte, []int) {
	return file_apecs_proto_rawDescGZIP(), []int{5}
}

func (x *ResponseTarget) GetConcernName() string {
	if x != nil {
		return x.ConcernName
	}
	return ""
}

func (x *ResponseTarget) GetTopicName() string {
	if x != nil {
		return x.TopicName
	}
	return ""
}

func (x *ResponseTarget) GetPartition() int32 {
	if x != nil {
		return x.Partition
	}
	return 0
}

type ApecsTxnResult struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Uid           string               `protobuf:"bytes,1,opt,name=uid,proto3" json:"uid,omitempty"`
	Status        Status               `protobuf:"varint,2,opt,name=status,proto3,enum=rkcy.pb.Status" json:"status,omitempty"`
	ProcessedTime *timestamp.Timestamp `protobuf:"bytes,3,opt,name=processed_time,json=processedTime,proto3" json:"processed_time,omitempty"`
	EffectiveTime *timestamp.Timestamp `protobuf:"bytes,4,opt,name=effective_time,json=effectiveTime,proto3" json:"effective_time,omitempty"`
	LogEvents     []*LogEvent          `protobuf:"bytes,5,rep,name=logEvents,proto3" json:"logEvents,omitempty"`
}

func (x *ApecsTxnResult) Reset() {
	*x = ApecsTxnResult{}
	if protoimpl.UnsafeEnabled {
		mi := &file_apecs_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ApecsTxnResult) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ApecsTxnResult) ProtoMessage() {}

func (x *ApecsTxnResult) ProtoReflect() protoreflect.Message {
	mi := &file_apecs_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ApecsTxnResult.ProtoReflect.Descriptor instead.
func (*ApecsTxnResult) Descriptor() ([]byte, []int) {
	return file_apecs_proto_rawDescGZIP(), []int{6}
}

func (x *ApecsTxnResult) GetUid() string {
	if x != nil {
		return x.Uid
	}
	return ""
}

func (x *ApecsTxnResult) GetStatus() Status {
	if x != nil {
		return x.Status
	}
	return Status_PENDING
}

func (x *ApecsTxnResult) GetProcessedTime() *timestamp.Timestamp {
	if x != nil {
		return x.ProcessedTime
	}
	return nil
}

func (x *ApecsTxnResult) GetEffectiveTime() *timestamp.Timestamp {
	if x != nil {
		return x.EffectiveTime
	}
	return nil
}

func (x *ApecsTxnResult) GetLogEvents() []*LogEvent {
	if x != nil {
		return x.LogEvents
	}
	return nil
}

type ApecsTxn_Step struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ConcernName   string               `protobuf:"bytes,1,opt,name=concern_name,json=concernName,proto3" json:"concern_name,omitempty"`       // logical persistence model that's used to partition messages
	Command       int32                `protobuf:"varint,2,opt,name=command,proto3" json:"command,omitempty"`                                 // command name, this will map to a piece of code (e.g. function)
	Key           string               `protobuf:"bytes,3,opt,name=key,proto3" json:"key,omitempty"`                                          // partition key
	Payload       []byte               `protobuf:"bytes,4,opt,name=payload,proto3" json:"payload,omitempty"`                                  // opaque payload for command
	Status        Status               `protobuf:"varint,5,opt,name=status,proto3,enum=rkcy.pb.Status" json:"status,omitempty"`               // status of this step, all start at pending, and can transition to complete, error, and rolled back
	ProcessedTime *timestamp.Timestamp `protobuf:"bytes,6,opt,name=processed_time,json=processedTime,proto3" json:"processed_time,omitempty"` // actual time this command result was recorded
	EffectiveTime *timestamp.Timestamp `protobuf:"bytes,7,opt,name=effective_time,json=effectiveTime,proto3" json:"effective_time,omitempty"` // effective time, useful in some applications as it may make sense to deviate from processed_time for reporting purposes
	LogEvents     []*LogEvent          `protobuf:"bytes,8,rep,name=logEvents,proto3" json:"logEvents,omitempty"`                              // general bucket for log events during a processed event
}

func (x *ApecsTxn_Step) Reset() {
	*x = ApecsTxn_Step{}
	if protoimpl.UnsafeEnabled {
		mi := &file_apecs_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ApecsTxn_Step) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ApecsTxn_Step) ProtoMessage() {}

func (x *ApecsTxn_Step) ProtoReflect() protoreflect.Message {
	mi := &file_apecs_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ApecsTxn_Step.ProtoReflect.Descriptor instead.
func (*ApecsTxn_Step) Descriptor() ([]byte, []int) {
	return file_apecs_proto_rawDescGZIP(), []int{0, 0}
}

func (x *ApecsTxn_Step) GetConcernName() string {
	if x != nil {
		return x.ConcernName
	}
	return ""
}

func (x *ApecsTxn_Step) GetCommand() int32 {
	if x != nil {
		return x.Command
	}
	return 0
}

func (x *ApecsTxn_Step) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *ApecsTxn_Step) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *ApecsTxn_Step) GetStatus() Status {
	if x != nil {
		return x.Status
	}
	return Status_PENDING
}

func (x *ApecsTxn_Step) GetProcessedTime() *timestamp.Timestamp {
	if x != nil {
		return x.ProcessedTime
	}
	return nil
}

func (x *ApecsTxn_Step) GetEffectiveTime() *timestamp.Timestamp {
	if x != nil {
		return x.EffectiveTime
	}
	return nil
}

func (x *ApecsTxn_Step) GetLogEvents() []*LogEvent {
	if x != nil {
		return x.LogEvents
	}
	return nil
}

var File_apecs_proto protoreflect.FileDescriptor

var file_apecs_proto_rawDesc = []byte{
	0x0a, 0x0b, 0x61, 0x70, 0x65, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x07, 0x72,
	0x6b, 0x63, 0x79, 0x2e, 0x70, 0x62, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x9f, 0x05, 0x0a, 0x08, 0x41, 0x70, 0x65, 0x63,
	0x73, 0x54, 0x78, 0x6e, 0x12, 0x10, 0x0a, 0x03, 0x75, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x03, 0x75, 0x69, 0x64, 0x12, 0x33, 0x0a, 0x09, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x15, 0x2e, 0x72, 0x6b, 0x63, 0x79,
	0x2e, 0x70, 0x62, 0x2e, 0x41, 0x70, 0x65, 0x63, 0x73, 0x54, 0x78, 0x6e, 0x2e, 0x44, 0x69, 0x72,
	0x52, 0x09, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1d, 0x0a, 0x0a, 0x63,
	0x61, 0x6e, 0x5f, 0x72, 0x65, 0x76, 0x65, 0x72, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52,
	0x09, 0x63, 0x61, 0x6e, 0x52, 0x65, 0x76, 0x65, 0x72, 0x74, 0x12, 0x3b, 0x0a, 0x0d, 0x66, 0x6f,
	0x72, 0x77, 0x61, 0x72, 0x64, 0x5f, 0x73, 0x74, 0x65, 0x70, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x16, 0x2e, 0x72, 0x6b, 0x63, 0x79, 0x2e, 0x70, 0x62, 0x2e, 0x41, 0x70, 0x65, 0x63,
	0x73, 0x54, 0x78, 0x6e, 0x2e, 0x53, 0x74, 0x65, 0x70, 0x52, 0x0c, 0x66, 0x6f, 0x72, 0x77, 0x61,
	0x72, 0x64, 0x53, 0x74, 0x65, 0x70, 0x73, 0x12, 0x3b, 0x0a, 0x0d, 0x72, 0x65, 0x76, 0x65, 0x72,
	0x73, 0x65, 0x5f, 0x73, 0x74, 0x65, 0x70, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x16,
	0x2e, 0x72, 0x6b, 0x63, 0x79, 0x2e, 0x70, 0x62, 0x2e, 0x41, 0x70, 0x65, 0x63, 0x73, 0x54, 0x78,
	0x6e, 0x2e, 0x53, 0x74, 0x65, 0x70, 0x52, 0x0c, 0x72, 0x65, 0x76, 0x65, 0x72, 0x73, 0x65, 0x53,
	0x74, 0x65, 0x70, 0x73, 0x12, 0x40, 0x0a, 0x0f, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x5f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e,
	0x72, 0x6b, 0x63, 0x79, 0x2e, 0x70, 0x62, 0x2e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x52, 0x0e, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x1a, 0xcf, 0x02, 0x0a, 0x04, 0x53, 0x74, 0x65, 0x70, 0x12,
	0x21, 0x0a, 0x0c, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x4e, 0x61,
	0x6d, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x12, 0x10, 0x0a, 0x03,
	0x6b, 0x65, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x18,
	0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x12, 0x27, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0f, 0x2e, 0x72, 0x6b, 0x63, 0x79, 0x2e,
	0x70, 0x62, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x12, 0x41, 0x0a, 0x0e, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x65, 0x64, 0x5f, 0x74,
	0x69, 0x6d, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65,
	0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0d, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x65, 0x64,
	0x54, 0x69, 0x6d, 0x65, 0x12, 0x41, 0x0a, 0x0e, 0x65, 0x66, 0x66, 0x65, 0x63, 0x74, 0x69, 0x76,
	0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0d, 0x65, 0x66, 0x66, 0x65, 0x63, 0x74,
	0x69, 0x76, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x2f, 0x0a, 0x09, 0x6c, 0x6f, 0x67, 0x45, 0x76,
	0x65, 0x6e, 0x74, 0x73, 0x18, 0x08, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x72, 0x6b, 0x63,
	0x79, 0x2e, 0x70, 0x62, 0x2e, 0x4c, 0x6f, 0x67, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x52, 0x09, 0x6c,
	0x6f, 0x67, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x22, 0x1f, 0x0a, 0x03, 0x44, 0x69, 0x72, 0x12,
	0x0b, 0x0a, 0x07, 0x46, 0x4f, 0x52, 0x57, 0x41, 0x52, 0x44, 0x10, 0x00, 0x12, 0x0b, 0x0a, 0x07,
	0x52, 0x45, 0x56, 0x45, 0x52, 0x53, 0x45, 0x10, 0x01, 0x22, 0x30, 0x0a, 0x08, 0x4c, 0x6f, 0x67,
	0x45, 0x76, 0x65, 0x6e, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x6d, 0x73, 0x67,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6d, 0x73, 0x67, 0x22, 0x4a, 0x0a, 0x10, 0x4d,
	0x6f, 0x73, 0x74, 0x52, 0x65, 0x63, 0x65, 0x6e, 0x74, 0x4f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x12,
	0x1e, 0x0a, 0x0a, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x0a, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12,
	0x16, 0x0a, 0x06, 0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x06, 0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x22, 0x9c, 0x02, 0x0a, 0x13, 0x41, 0x70, 0x65, 0x63,
	0x73, 0x53, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x10, 0x0a, 0x03, 0x75, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x75, 0x69,
	0x64, 0x12, 0x21, 0x0a, 0x0c, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x5f, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e,
	0x4e, 0x61, 0x6d, 0x65, 0x12, 0x2f, 0x0a, 0x02, 0x6f, 0x70, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0e,
	0x32, 0x1f, 0x2e, 0x72, 0x6b, 0x63, 0x79, 0x2e, 0x70, 0x62, 0x2e, 0x41, 0x70, 0x65, 0x63, 0x73,
	0x53, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x4f,
	0x70, 0x52, 0x02, 0x6f, 0x70, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f,
	0x61, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61,
	0x64, 0x12, 0x40, 0x0a, 0x0f, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x5f, 0x74, 0x61,
	0x72, 0x67, 0x65, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x72, 0x6b, 0x63,
	0x79, 0x2e, 0x70, 0x62, 0x2e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x54, 0x61, 0x72,
	0x67, 0x65, 0x74, 0x52, 0x0e, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x54, 0x61, 0x72,
	0x67, 0x65, 0x74, 0x22, 0x31, 0x0a, 0x02, 0x4f, 0x70, 0x12, 0x07, 0x0a, 0x03, 0x47, 0x45, 0x54,
	0x10, 0x00, 0x12, 0x0a, 0x0a, 0x06, 0x43, 0x52, 0x45, 0x41, 0x54, 0x45, 0x10, 0x01, 0x12, 0x0a,
	0x0a, 0x06, 0x55, 0x50, 0x44, 0x41, 0x54, 0x45, 0x10, 0x02, 0x12, 0x0a, 0x0a, 0x06, 0x44, 0x45,
	0x4c, 0x45, 0x54, 0x45, 0x10, 0x03, 0x22, 0xf9, 0x01, 0x0a, 0x12, 0x41, 0x70, 0x65, 0x63, 0x73,
	0x53, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x12, 0x10, 0x0a,
	0x03, 0x75, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x75, 0x69, 0x64, 0x12,
	0x34, 0x0a, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x20, 0x2e,
	0x72, 0x6b, 0x63, 0x79, 0x2e, 0x70, 0x62, 0x2e, 0x41, 0x70, 0x65, 0x63, 0x73, 0x53, 0x74, 0x6f,
	0x72, 0x61, 0x67, 0x65, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x2e, 0x43, 0x6f, 0x64, 0x65, 0x52,
	0x04, 0x63, 0x6f, 0x64, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x12,
	0x2b, 0x0a, 0x03, 0x6d, 0x72, 0x6f, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x72,
	0x6b, 0x63, 0x79, 0x2e, 0x70, 0x62, 0x2e, 0x4d, 0x6f, 0x73, 0x74, 0x52, 0x65, 0x63, 0x65, 0x6e,
	0x74, 0x4f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x52, 0x03, 0x6d, 0x72, 0x6f, 0x22, 0x54, 0x0a, 0x04,
	0x43, 0x6f, 0x64, 0x65, 0x12, 0x0b, 0x0a, 0x07, 0x53, 0x55, 0x43, 0x43, 0x45, 0x53, 0x53, 0x10,
	0x00, 0x12, 0x0c, 0x0a, 0x08, 0x49, 0x4e, 0x54, 0x45, 0x52, 0x4e, 0x41, 0x4c, 0x10, 0x01, 0x12,
	0x12, 0x0a, 0x0e, 0x4d, 0x41, 0x52, 0x53, 0x48, 0x41, 0x4c, 0x5f, 0x46, 0x41, 0x49, 0x4c, 0x45,
	0x44, 0x10, 0x02, 0x12, 0x0d, 0x0a, 0x09, 0x4e, 0x4f, 0x54, 0x5f, 0x46, 0x4f, 0x55, 0x4e, 0x44,
	0x10, 0x03, 0x12, 0x0e, 0x0a, 0x0a, 0x43, 0x4f, 0x4e, 0x4e, 0x45, 0x43, 0x54, 0x49, 0x4f, 0x4e,
	0x10, 0x04, 0x22, 0x70, 0x0a, 0x0e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x54, 0x61,
	0x72, 0x67, 0x65, 0x74, 0x12, 0x21, 0x0a, 0x0c, 0x63, 0x6f, 0x6e, 0x63, 0x65, 0x72, 0x6e, 0x5f,
	0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x63, 0x6f, 0x6e, 0x63,
	0x65, 0x72, 0x6e, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x1d, 0x0a, 0x0a, 0x74, 0x6f, 0x70, 0x69, 0x63,
	0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x74, 0x6f, 0x70,
	0x69, 0x63, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x70, 0x61, 0x72, 0x74, 0x69, 0x74,
	0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x09, 0x70, 0x61, 0x72, 0x74, 0x69,
	0x74, 0x69, 0x6f, 0x6e, 0x22, 0x82, 0x02, 0x0a, 0x0e, 0x41, 0x70, 0x65, 0x63, 0x73, 0x54, 0x78,
	0x6e, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x75, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x75, 0x69, 0x64, 0x12, 0x27, 0x0a, 0x06, 0x73, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0f, 0x2e, 0x72, 0x6b, 0x63, 0x79,
	0x2e, 0x70, 0x62, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x12, 0x41, 0x0a, 0x0e, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x65, 0x64, 0x5f,
	0x74, 0x69, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d,
	0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0d, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x65,
	0x64, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x41, 0x0a, 0x0e, 0x65, 0x66, 0x66, 0x65, 0x63, 0x74, 0x69,
	0x76, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0d, 0x65, 0x66, 0x66, 0x65, 0x63,
	0x74, 0x69, 0x76, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x2f, 0x0a, 0x09, 0x6c, 0x6f, 0x67, 0x45,
	0x76, 0x65, 0x6e, 0x74, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x72, 0x6b,
	0x63, 0x79, 0x2e, 0x70, 0x62, 0x2e, 0x4c, 0x6f, 0x67, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x52, 0x09,
	0x6c, 0x6f, 0x67, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x2a, 0x3c, 0x0a, 0x06, 0x53, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x12, 0x0b, 0x0a, 0x07, 0x50, 0x45, 0x4e, 0x44, 0x49, 0x4e, 0x47, 0x10, 0x00,
	0x12, 0x0c, 0x0a, 0x08, 0x43, 0x4f, 0x4d, 0x50, 0x4c, 0x45, 0x54, 0x45, 0x10, 0x01, 0x12, 0x0c,
	0x0a, 0x08, 0x52, 0x45, 0x56, 0x45, 0x52, 0x54, 0x45, 0x44, 0x10, 0x02, 0x12, 0x09, 0x0a, 0x05,
	0x45, 0x52, 0x52, 0x4f, 0x52, 0x10, 0x03, 0x42, 0x2f, 0x5a, 0x2d, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6c, 0x61, 0x63, 0x68, 0x6c, 0x61, 0x6e, 0x6f, 0x72, 0x72,
	0x2f, 0x72, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x2f, 0x70, 0x6b, 0x67,
	0x2f, 0x72, 0x6b, 0x63, 0x79, 0x2f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_apecs_proto_rawDescOnce sync.Once
	file_apecs_proto_rawDescData = file_apecs_proto_rawDesc
)

func file_apecs_proto_rawDescGZIP() []byte {
	file_apecs_proto_rawDescOnce.Do(func() {
		file_apecs_proto_rawDescData = protoimpl.X.CompressGZIP(file_apecs_proto_rawDescData)
	})
	return file_apecs_proto_rawDescData
}

var file_apecs_proto_enumTypes = make([]protoimpl.EnumInfo, 4)
var file_apecs_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_apecs_proto_goTypes = []interface{}{
	(Status)(0),                  // 0: rkcy.pb.Status
	(ApecsTxn_Dir)(0),            // 1: rkcy.pb.ApecsTxn.Dir
	(ApecsStorageRequest_Op)(0),  // 2: rkcy.pb.ApecsStorageRequest.Op
	(ApecsStorageResult_Code)(0), // 3: rkcy.pb.ApecsStorageResult.Code
	(*ApecsTxn)(nil),             // 4: rkcy.pb.ApecsTxn
	(*LogEvent)(nil),             // 5: rkcy.pb.LogEvent
	(*MostRecentOffset)(nil),     // 6: rkcy.pb.MostRecentOffset
	(*ApecsStorageRequest)(nil),  // 7: rkcy.pb.ApecsStorageRequest
	(*ApecsStorageResult)(nil),   // 8: rkcy.pb.ApecsStorageResult
	(*ResponseTarget)(nil),       // 9: rkcy.pb.ResponseTarget
	(*ApecsTxnResult)(nil),       // 10: rkcy.pb.ApecsTxnResult
	(*ApecsTxn_Step)(nil),        // 11: rkcy.pb.ApecsTxn.Step
	(*timestamp.Timestamp)(nil),  // 12: google.protobuf.Timestamp
}
var file_apecs_proto_depIdxs = []int32{
	1,  // 0: rkcy.pb.ApecsTxn.direction:type_name -> rkcy.pb.ApecsTxn.Dir
	11, // 1: rkcy.pb.ApecsTxn.forward_steps:type_name -> rkcy.pb.ApecsTxn.Step
	11, // 2: rkcy.pb.ApecsTxn.reverse_steps:type_name -> rkcy.pb.ApecsTxn.Step
	9,  // 3: rkcy.pb.ApecsTxn.response_target:type_name -> rkcy.pb.ResponseTarget
	2,  // 4: rkcy.pb.ApecsStorageRequest.op:type_name -> rkcy.pb.ApecsStorageRequest.Op
	9,  // 5: rkcy.pb.ApecsStorageRequest.response_target:type_name -> rkcy.pb.ResponseTarget
	3,  // 6: rkcy.pb.ApecsStorageResult.code:type_name -> rkcy.pb.ApecsStorageResult.Code
	6,  // 7: rkcy.pb.ApecsStorageResult.mro:type_name -> rkcy.pb.MostRecentOffset
	0,  // 8: rkcy.pb.ApecsTxnResult.status:type_name -> rkcy.pb.Status
	12, // 9: rkcy.pb.ApecsTxnResult.processed_time:type_name -> google.protobuf.Timestamp
	12, // 10: rkcy.pb.ApecsTxnResult.effective_time:type_name -> google.protobuf.Timestamp
	5,  // 11: rkcy.pb.ApecsTxnResult.logEvents:type_name -> rkcy.pb.LogEvent
	0,  // 12: rkcy.pb.ApecsTxn.Step.status:type_name -> rkcy.pb.Status
	12, // 13: rkcy.pb.ApecsTxn.Step.processed_time:type_name -> google.protobuf.Timestamp
	12, // 14: rkcy.pb.ApecsTxn.Step.effective_time:type_name -> google.protobuf.Timestamp
	5,  // 15: rkcy.pb.ApecsTxn.Step.logEvents:type_name -> rkcy.pb.LogEvent
	16, // [16:16] is the sub-list for method output_type
	16, // [16:16] is the sub-list for method input_type
	16, // [16:16] is the sub-list for extension type_name
	16, // [16:16] is the sub-list for extension extendee
	0,  // [0:16] is the sub-list for field type_name
}

func init() { file_apecs_proto_init() }
func file_apecs_proto_init() {
	if File_apecs_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_apecs_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ApecsTxn); i {
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
		file_apecs_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LogEvent); i {
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
		file_apecs_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MostRecentOffset); i {
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
		file_apecs_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ApecsStorageRequest); i {
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
		file_apecs_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ApecsStorageResult); i {
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
		file_apecs_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ResponseTarget); i {
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
		file_apecs_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ApecsTxnResult); i {
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
		file_apecs_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ApecsTxn_Step); i {
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
			RawDescriptor: file_apecs_proto_rawDesc,
			NumEnums:      4,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_apecs_proto_goTypes,
		DependencyIndexes: file_apecs_proto_depIdxs,
		EnumInfos:         file_apecs_proto_enumTypes,
		MessageInfos:      file_apecs_proto_msgTypes,
	}.Build()
	File_apecs_proto = out.File
	file_apecs_proto_rawDesc = nil
	file_apecs_proto_goTypes = nil
	file_apecs_proto_depIdxs = nil
}
