package main

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type ServerStream struct {
	messageDesc protoreflect.MessageDescriptor
	stream      grpc.ServerStream
	requestCh   chan *StreamData
}

func NewServerStream(stream grpc.ServerStream, messageDesc protoreflect.MessageDescriptor) *ServerStream {
	serverStream := &ServerStream{
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

func (s *ServerStream) SendMsg(message any) error {
	if s == nil {
		return status.Error(codes.Internal, "server stream hasn't been initialized")
	}

	return s.stream.SendMsg(message)
}

func (s *ServerStream) fetchRequests() {
	defer close(s.requestCh)

	md, _ := metadata.FromIncomingContext(s.stream.Context())
	s.requestCh <- &StreamData{
		Header: md,
	}

	for {
		request := dynamicpb.NewMessage(s.messageDesc)
		err := s.stream.RecvMsg(request)

		if err != nil {
			request = nil
		}

		s.requestCh <- &StreamData{
			Message: request,
			Error:   err,
		}

		if err != nil {
			return
		}
	}
}
