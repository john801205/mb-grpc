package grpc

import (
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type StreamData struct {
	Header  metadata.MD
	Message proto.Message
	Trailer metadata.MD
	Status  *status.Status
}

