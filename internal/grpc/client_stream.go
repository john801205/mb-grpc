package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type ClientStream struct {
	ctx         context.Context
	stream      grpc.ClientStream
	messageDesc protoreflect.MessageDescriptor
	responseCh  chan *StreamData
	done        chan struct{}
}

func NewClientStream(
	ctx context.Context,
	target string,
	streamDesc *grpc.StreamDesc,
	method string,
	messageDesc protoreflect.MessageDescriptor,
) (*ClientStream, error) {
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	stream, err := conn.NewStream(ctx, streamDesc, method)
	if err != nil {
		return nil, err
	}

	clientStream := &ClientStream{
		ctx:         ctx,
		stream:      stream,
		messageDesc: messageDesc,
		responseCh:  make(chan *StreamData),
		done:        make(chan struct{}),
	}
	go clientStream.fetchResponses()

	return clientStream, nil
}

func (s *ClientStream) CloseSend() error {
	if s == nil {
		return nil
	}

	return s.stream.CloseSend()
}

func (s *ClientStream) hasResponse() bool {
	if s == nil {
		return false
	}

	running := true
	select {
	case <-s.done:
		running = false
	default:
	}

	return running || len(s.responseCh) != 0
}

func (s *ClientStream) WaitFinished() {
	<-s.done
}

func (s *ClientStream) Responses() <-chan *StreamData {
	if !s.hasResponse() {
		return nil
	}

	return s.responseCh
}

func (s *ClientStream) Forward(message any) error {
	if s == nil {
		return status.Error(codes.Internal, "client stream hasn't been initialized")
	}

	return s.stream.SendMsg(message)
}

func (s *ClientStream) fetchResponses() {
	defer func() {
		close(s.done)
		close(s.responseCh)
	}()

	header, err := s.stream.Header()
	if err != nil {
		resp := &StreamData{
			Header: header,
			Error:  err,
		}

		select {
		case s.responseCh <- resp:
		case <-s.ctx.Done():
		}

		return
	}

	for {
		message := dynamicpb.NewMessage(s.messageDesc)
		err := s.stream.RecvMsg(message)
		var resp *StreamData

		if err != nil {
			resp = &StreamData{
				Header: header,
				Trailer: s.stream.Trailer(),
				Error:  err,
			}
		} else {
			resp = &StreamData{
				Header: header,
				Message: message,
			}
		}
		header = nil

		select {
		case s.responseCh <- resp:
		case <-s.ctx.Done():
			return
		}

		if err != nil {
			return
		}
	}
}
