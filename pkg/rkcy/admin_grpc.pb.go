// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package rkcy

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion7

// AdminServiceClient is the client API for AdminService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AdminServiceClient interface {
	Platform(ctx context.Context, in *Void, opts ...grpc.CallOption) (*Platform, error)
	DecodeInstance(ctx context.Context, in *DecodeInstanceArgs, opts ...grpc.CallOption) (*DecodeResponse, error)
	DecodeArgPayload(ctx context.Context, in *DecodePayloadArgs, opts ...grpc.CallOption) (*DecodeResponse, error)
	DecodeResultPayload(ctx context.Context, in *DecodePayloadArgs, opts ...grpc.CallOption) (*DecodeResponse, error)
}

type adminServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewAdminServiceClient(cc grpc.ClientConnInterface) AdminServiceClient {
	return &adminServiceClient{cc}
}

func (c *adminServiceClient) Platform(ctx context.Context, in *Void, opts ...grpc.CallOption) (*Platform, error) {
	out := new(Platform)
	err := c.cc.Invoke(ctx, "/rkcy.AdminService/Platform", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *adminServiceClient) DecodeInstance(ctx context.Context, in *DecodeInstanceArgs, opts ...grpc.CallOption) (*DecodeResponse, error) {
	out := new(DecodeResponse)
	err := c.cc.Invoke(ctx, "/rkcy.AdminService/DecodeInstance", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *adminServiceClient) DecodeArgPayload(ctx context.Context, in *DecodePayloadArgs, opts ...grpc.CallOption) (*DecodeResponse, error) {
	out := new(DecodeResponse)
	err := c.cc.Invoke(ctx, "/rkcy.AdminService/DecodeArgPayload", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *adminServiceClient) DecodeResultPayload(ctx context.Context, in *DecodePayloadArgs, opts ...grpc.CallOption) (*DecodeResponse, error) {
	out := new(DecodeResponse)
	err := c.cc.Invoke(ctx, "/rkcy.AdminService/DecodeResultPayload", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AdminServiceServer is the server API for AdminService service.
// All implementations must embed UnimplementedAdminServiceServer
// for forward compatibility
type AdminServiceServer interface {
	Platform(context.Context, *Void) (*Platform, error)
	DecodeInstance(context.Context, *DecodeInstanceArgs) (*DecodeResponse, error)
	DecodeArgPayload(context.Context, *DecodePayloadArgs) (*DecodeResponse, error)
	DecodeResultPayload(context.Context, *DecodePayloadArgs) (*DecodeResponse, error)
	mustEmbedUnimplementedAdminServiceServer()
}

// UnimplementedAdminServiceServer must be embedded to have forward compatible implementations.
type UnimplementedAdminServiceServer struct {
}

func (UnimplementedAdminServiceServer) Platform(context.Context, *Void) (*Platform, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Platform not implemented")
}
func (UnimplementedAdminServiceServer) DecodeInstance(context.Context, *DecodeInstanceArgs) (*DecodeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DecodeInstance not implemented")
}
func (UnimplementedAdminServiceServer) DecodeArgPayload(context.Context, *DecodePayloadArgs) (*DecodeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DecodeArgPayload not implemented")
}
func (UnimplementedAdminServiceServer) DecodeResultPayload(context.Context, *DecodePayloadArgs) (*DecodeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DecodeResultPayload not implemented")
}
func (UnimplementedAdminServiceServer) mustEmbedUnimplementedAdminServiceServer() {}

// UnsafeAdminServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AdminServiceServer will
// result in compilation errors.
type UnsafeAdminServiceServer interface {
	mustEmbedUnimplementedAdminServiceServer()
}

func RegisterAdminServiceServer(s grpc.ServiceRegistrar, srv AdminServiceServer) {
	s.RegisterService(&_AdminService_serviceDesc, srv)
}

func _AdminService_Platform_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Void)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdminServiceServer).Platform(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rkcy.AdminService/Platform",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdminServiceServer).Platform(ctx, req.(*Void))
	}
	return interceptor(ctx, in, info, handler)
}

func _AdminService_DecodeInstance_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DecodeInstanceArgs)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdminServiceServer).DecodeInstance(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rkcy.AdminService/DecodeInstance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdminServiceServer).DecodeInstance(ctx, req.(*DecodeInstanceArgs))
	}
	return interceptor(ctx, in, info, handler)
}

func _AdminService_DecodeArgPayload_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DecodePayloadArgs)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdminServiceServer).DecodeArgPayload(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rkcy.AdminService/DecodeArgPayload",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdminServiceServer).DecodeArgPayload(ctx, req.(*DecodePayloadArgs))
	}
	return interceptor(ctx, in, info, handler)
}

func _AdminService_DecodeResultPayload_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DecodePayloadArgs)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdminServiceServer).DecodeResultPayload(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rkcy.AdminService/DecodeResultPayload",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdminServiceServer).DecodeResultPayload(ctx, req.(*DecodePayloadArgs))
	}
	return interceptor(ctx, in, info, handler)
}

var _AdminService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "rkcy.AdminService",
	HandlerType: (*AdminServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Platform",
			Handler:    _AdminService_Platform_Handler,
		},
		{
			MethodName: "DecodeInstance",
			Handler:    _AdminService_DecodeInstance_Handler,
		},
		{
			MethodName: "DecodeArgPayload",
			Handler:    _AdminService_DecodeArgPayload_Handler,
		},
		{
			MethodName: "DecodeResultPayload",
			Handler:    _AdminService_DecodeResultPayload_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "admin.proto",
}
