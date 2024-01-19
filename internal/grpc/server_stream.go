package grpc

import (
	"context"

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
	done        chan struct{}
}

func NewServerStream(ctx context.Context, stream grpc.ServerStream, messageDesc protoreflect.MessageDescriptor) *ServerStream {
	serverStream := &ServerStream{
		ctx:         ctx,
		stream:      stream,
		messageDesc: messageDesc,
		requestCh:   make(chan *StreamData),
		done:        make(chan struct{}),
	}
	go serverStream.fetchRequests()

	return serverStream
}

func (s *ServerStream) hasRequest() bool {
	if s == nil {
		return false
	}

	running := true
	select {
	case <-s.done:
		running = false
	default:
	}

	return running || len(s.requestCh) != 0
}

func (s *ServerStream) WaitFinished() {
	<-s.done
}

func (s *ServerStream) Requests() <-chan *StreamData {
	if !s.hasRequest() {
		return nil
	}

	return s.requestCh
}

func (s *ServerStream) SendMsg(header, trailer metadata.MD, message any) error {
	if s == nil {
		return status.Error(codes.Internal, "server stream hasn't been initialized")
	}

	if len(header) != 0 {
		err := s.stream.SetHeader(header)
		if err != nil {
			return err
		}
	}

	if message != nil {
		err := s.stream.SendMsg(message)
		if err != nil {
			return err
		}
	}

	if len(trailer) != 0 {
		s.stream.SetTrailer(trailer)
	}

	return nil
}

func (s *ServerStream) fetchRequests() {
	defer func() {
		close(s.done)
		close(s.requestCh)
	}()

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
		var req *StreamData

		if err != nil {
			req = &StreamData{
				Error: err,
			}
		} else {
			req = &StreamData{
				Message: request,
			}
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
