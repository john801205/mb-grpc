package main

import (
	"context"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type ServerStream struct {
	ctx         context.Context
	messageDesc protoreflect.MessageDescriptor
	stream      grpc.ServerStream
	requestCh   chan *StreamData
}

func NewServerStream(ctx context.Context, stream grpc.ServerStream, messageDesc protoreflect.MessageDescriptor) *ServerStream {
	serverStream := &ServerStream{
		ctx: ctx,
		stream: stream,
		messageDesc: messageDesc,
		requestCh: make(chan *StreamData),
	}
	go serverStream.fetchRequests()

	return serverStream
}

func (s *ServerStream) Requests() <-chan *StreamData {
	if s == nil {
		return nil
	}

	return s.requestCh
}

func (s *ServerStream) SendMsg(header, trailer metadata.MD, message any) error {
	if s == nil {
		return status.Error(codes.Internal, "server stream hasn't been initialized")
	}

	err := s.stream.SetHeader(header)
	if err != nil {
		return err
	}

	s.stream.SetTrailer(trailer)

	if message == nil {
		return nil
	}

	return s.stream.SendMsg(message)
}

func (s *ServerStream) fetchRequests() {
	defer close(s.requestCh)

	md, exist := metadata.FromIncomingContext(s.stream.Context())
	if exist {
		req := &StreamData{
			Header: md,
		}

		select {
		case s.requestCh <- req:
		case <-s.ctx.Done():
			return
		}
	}

	for {
		request := dynamicpb.NewMessage(s.messageDesc)
		err := s.stream.RecvMsg(request)
		var st *status.Status

		if err == io.EOF {
			return
		} else if err != nil {
			st = status.Convert(err)
		}

		req := &StreamData{
			Message: request,
			Status: st,
		}
		select {
		case s.requestCh <- req:
		case <-s.ctx.Done():
			return
		}

		if err != nil {
			return
		}
	}
}
