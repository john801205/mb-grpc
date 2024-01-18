// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        v4.25.1
// source: test/mb-grpc/pingpong/pingpong.proto

package pingpong

import (
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

type Ping struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ping string `protobuf:"bytes,1,opt,name=ping,proto3" json:"ping,omitempty"`
}

func (x *Ping) Reset() {
	*x = Ping{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_mb_grpc_pingpong_pingpong_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Ping) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Ping) ProtoMessage() {}

func (x *Ping) ProtoReflect() protoreflect.Message {
	mi := &file_test_mb_grpc_pingpong_pingpong_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Ping.ProtoReflect.Descriptor instead.
func (*Ping) Descriptor() ([]byte, []int) {
	return file_test_mb_grpc_pingpong_pingpong_proto_rawDescGZIP(), []int{0}
}

func (x *Ping) GetPing() string {
	if x != nil {
		return x.Ping
	}
	return ""
}

type Pong struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Pong string `protobuf:"bytes,1,opt,name=pong,proto3" json:"pong,omitempty"`
}

func (x *Pong) Reset() {
	*x = Pong{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_mb_grpc_pingpong_pingpong_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Pong) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Pong) ProtoMessage() {}

func (x *Pong) ProtoReflect() protoreflect.Message {
	mi := &file_test_mb_grpc_pingpong_pingpong_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Pong.ProtoReflect.Descriptor instead.
func (*Pong) Descriptor() ([]byte, []int) {
	return file_test_mb_grpc_pingpong_pingpong_proto_rawDescGZIP(), []int{1}
}

func (x *Pong) GetPong() string {
	if x != nil {
		return x.Pong
	}
	return ""
}

var File_test_mb_grpc_pingpong_pingpong_proto protoreflect.FileDescriptor

var file_test_mb_grpc_pingpong_pingpong_proto_rawDesc = []byte{
	0x0a, 0x24, 0x74, 0x65, 0x73, 0x74, 0x2f, 0x6d, 0x62, 0x2d, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x70,
	0x69, 0x6e, 0x67, 0x70, 0x6f, 0x6e, 0x67, 0x2f, 0x70, 0x69, 0x6e, 0x67, 0x70, 0x6f, 0x6e, 0x67,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x70, 0x69, 0x6e, 0x67, 0x70, 0x6f, 0x6e, 0x67,
	0x22, 0x1a, 0x0a, 0x04, 0x50, 0x69, 0x6e, 0x67, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x69, 0x6e, 0x67,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x70, 0x69, 0x6e, 0x67, 0x22, 0x1a, 0x0a, 0x04,
	0x50, 0x6f, 0x6e, 0x67, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x6f, 0x6e, 0x67, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x04, 0x70, 0x6f, 0x6e, 0x67, 0x32, 0xd9, 0x01, 0x0a, 0x07, 0x53, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x12, 0x2c, 0x0a, 0x08, 0x50, 0x69, 0x6e, 0x67, 0x50, 0x6f, 0x6e, 0x67,
	0x12, 0x0e, 0x2e, 0x70, 0x69, 0x6e, 0x67, 0x70, 0x6f, 0x6e, 0x67, 0x2e, 0x50, 0x69, 0x6e, 0x67,
	0x1a, 0x0e, 0x2e, 0x70, 0x69, 0x6e, 0x67, 0x70, 0x6f, 0x6e, 0x67, 0x2e, 0x50, 0x6f, 0x6e, 0x67,
	0x22, 0x00, 0x12, 0x32, 0x0a, 0x0c, 0x50, 0x69, 0x6e, 0x67, 0x50, 0x6f, 0x6e, 0x67, 0x50, 0x6f,
	0x6e, 0x67, 0x12, 0x0e, 0x2e, 0x70, 0x69, 0x6e, 0x67, 0x70, 0x6f, 0x6e, 0x67, 0x2e, 0x50, 0x69,
	0x6e, 0x67, 0x1a, 0x0e, 0x2e, 0x70, 0x69, 0x6e, 0x67, 0x70, 0x6f, 0x6e, 0x67, 0x2e, 0x50, 0x6f,
	0x6e, 0x67, 0x22, 0x00, 0x30, 0x01, 0x12, 0x32, 0x0a, 0x0c, 0x50, 0x69, 0x6e, 0x67, 0x50, 0x69,
	0x6e, 0x67, 0x50, 0x6f, 0x6e, 0x67, 0x12, 0x0e, 0x2e, 0x70, 0x69, 0x6e, 0x67, 0x70, 0x6f, 0x6e,
	0x67, 0x2e, 0x50, 0x69, 0x6e, 0x67, 0x1a, 0x0e, 0x2e, 0x70, 0x69, 0x6e, 0x67, 0x70, 0x6f, 0x6e,
	0x67, 0x2e, 0x50, 0x6f, 0x6e, 0x67, 0x22, 0x00, 0x28, 0x01, 0x12, 0x38, 0x0a, 0x10, 0x50, 0x69,
	0x6e, 0x67, 0x50, 0x69, 0x6e, 0x67, 0x50, 0x6f, 0x6e, 0x67, 0x50, 0x6f, 0x6e, 0x67, 0x12, 0x0e,
	0x2e, 0x70, 0x69, 0x6e, 0x67, 0x70, 0x6f, 0x6e, 0x67, 0x2e, 0x50, 0x69, 0x6e, 0x67, 0x1a, 0x0e,
	0x2e, 0x70, 0x69, 0x6e, 0x67, 0x70, 0x6f, 0x6e, 0x67, 0x2e, 0x50, 0x6f, 0x6e, 0x67, 0x22, 0x00,
	0x28, 0x01, 0x30, 0x01, 0x42, 0x35, 0x5a, 0x33, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x6a, 0x6f, 0x68, 0x6e, 0x38, 0x30, 0x31, 0x32, 0x30, 0x35, 0x2f, 0x6d, 0x62,
	0x2d, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x2f, 0x6d, 0x62, 0x2d, 0x67, 0x72,
	0x70, 0x63, 0x2f, 0x70, 0x69, 0x6e, 0x67, 0x70, 0x6f, 0x6e, 0x67, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_test_mb_grpc_pingpong_pingpong_proto_rawDescOnce sync.Once
	file_test_mb_grpc_pingpong_pingpong_proto_rawDescData = file_test_mb_grpc_pingpong_pingpong_proto_rawDesc
)

func file_test_mb_grpc_pingpong_pingpong_proto_rawDescGZIP() []byte {
	file_test_mb_grpc_pingpong_pingpong_proto_rawDescOnce.Do(func() {
		file_test_mb_grpc_pingpong_pingpong_proto_rawDescData = protoimpl.X.CompressGZIP(file_test_mb_grpc_pingpong_pingpong_proto_rawDescData)
	})
	return file_test_mb_grpc_pingpong_pingpong_proto_rawDescData
}

var file_test_mb_grpc_pingpong_pingpong_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_test_mb_grpc_pingpong_pingpong_proto_goTypes = []interface{}{
	(*Ping)(nil), // 0: pingpong.Ping
	(*Pong)(nil), // 1: pingpong.Pong
}
var file_test_mb_grpc_pingpong_pingpong_proto_depIdxs = []int32{
	0, // 0: pingpong.Service.PingPong:input_type -> pingpong.Ping
	0, // 1: pingpong.Service.PingPongPong:input_type -> pingpong.Ping
	0, // 2: pingpong.Service.PingPingPong:input_type -> pingpong.Ping
	0, // 3: pingpong.Service.PingPingPongPong:input_type -> pingpong.Ping
	1, // 4: pingpong.Service.PingPong:output_type -> pingpong.Pong
	1, // 5: pingpong.Service.PingPongPong:output_type -> pingpong.Pong
	1, // 6: pingpong.Service.PingPingPong:output_type -> pingpong.Pong
	1, // 7: pingpong.Service.PingPingPongPong:output_type -> pingpong.Pong
	4, // [4:8] is the sub-list for method output_type
	0, // [0:4] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_test_mb_grpc_pingpong_pingpong_proto_init() }
func file_test_mb_grpc_pingpong_pingpong_proto_init() {
	if File_test_mb_grpc_pingpong_pingpong_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_test_mb_grpc_pingpong_pingpong_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Ping); i {
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
		file_test_mb_grpc_pingpong_pingpong_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Pong); i {
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
			RawDescriptor: file_test_mb_grpc_pingpong_pingpong_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_test_mb_grpc_pingpong_pingpong_proto_goTypes,
		DependencyIndexes: file_test_mb_grpc_pingpong_pingpong_proto_depIdxs,
		MessageInfos:      file_test_mb_grpc_pingpong_pingpong_proto_msgTypes,
	}.Build()
	File_test_mb_grpc_pingpong_pingpong_proto = out.File
	file_test_mb_grpc_pingpong_pingpong_proto_rawDesc = nil
	file_test_mb_grpc_pingpong_pingpong_proto_goTypes = nil
	file_test_mb_grpc_pingpong_pingpong_proto_depIdxs = nil
}
