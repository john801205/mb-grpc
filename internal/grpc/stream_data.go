package grpc

import (
	"encoding/json"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type StreamData struct {
	Header  metadata.MD
	Message proto.Message
	Trailer metadata.MD
	Error   error
}

func (s *StreamData) MarshalJSON() ([]byte, error) {
	type streamData struct {
		Header  metadata.MD     `json:"header,omitempty"`
		Message json.RawMessage `json:"message,omitempty"`
		Trailer metadata.MD     `json:"trailer,omitempty"`
		Error   string          `json:"error,omitempty"`
	}

	data := &streamData{
		Header: s.Header,
		Trailer: s.Trailer,
	}

	if s.Message != nil {
		bytes, err := protojson.Marshal(s.Message)
		if err != nil {
			return nil, err
		}

		data.Message = bytes
	}
	if s.Error != nil {
		data.Error = s.Error.Error()
	}

	return json.Marshal(data)
}

func (s *StreamData) String() string {
	bytes, _ := json.Marshal(s)
	return string(bytes)
}
