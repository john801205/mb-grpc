package mountebank

import (
	"encoding/json"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type RpcMessage struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

type RpcData struct {
	Method         string        `json:"method"`
	RequestHeader  metadata.MD   `json:"requestHeader,omitempty"`
	ResponseHeader metadata.MD   `json:"responseHeader,omitempty"`
	Messages       []*RpcMessage `json:"messages,omitempty"`
}

func NewRpcData(method string) *RpcData {
	return &RpcData{
		Method: method,
	}
}

func (d *RpcData) AddRequestData(header metadata.MD, message proto.Message) error {
	if len(header) != 0 {
		d.RequestHeader = metadata.Join(d.RequestHeader, header)
	}

	if message != nil {
		bytes, err := protojson.Marshal(message)
		if err != nil {
			return err
		}

		d.Messages = append(d.Messages, &RpcMessage{
			Type:  "request",
			Value: bytes,
		})
	}

	return nil
}

func (d *RpcData) AddResponseData(header metadata.MD, message proto.Message) error {
	if len(header) != 0 {
		d.ResponseHeader = metadata.Join(d.ResponseHeader, header)
	}

	if message != nil {
		bytes, err := protojson.Marshal(message)
		if err != nil {
			return err
		}

		d.Messages = append(d.Messages, &RpcMessage{
			Type:  "response",
			Value: bytes,
		})
	}

	return nil
}
