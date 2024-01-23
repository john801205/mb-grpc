package mountebank

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type RpcMessage struct {
	From    string          `json:"from"`
	Message json.RawMessage `json:"message"`
}

type RpcData struct {
	Method         string        `json:"method"`
	RequestHeader  metadata.MD   `json:"requestHeader"`
	ResponseHeader metadata.MD   `json:"responseHeader"`
	Messages       []*RpcMessage `json:"messages"`
}

func NewRpcData(method string) *RpcData {
	return &RpcData{
		Method:         method,
		RequestHeader:  metadata.New(nil),
		ResponseHeader: metadata.New(nil),
		Messages:       make([]*RpcMessage, 0),
	}
}

func (d *RpcData) AddRequestData(header metadata.MD, message proto.Message) error {
	for key, vals := range header {
		if strings.HasSuffix(key, "-bin") {
			for _, val := range vals {
				str := base64.StdEncoding.EncodeToString([]byte(val))
				d.RequestHeader.Append(key, str)
			}
		} else {
			d.RequestHeader.Append(key, vals...)
		}
	}

	if message != nil {
		bytes, err := protojson.Marshal(message)
		if err != nil {
			return err
		}

		d.Messages = append(d.Messages, &RpcMessage{
			From:    "client",
			Message: bytes,
		})
	}

	return nil
}

func (d *RpcData) AddResponseData(header metadata.MD, message proto.Message) error {
	for key, vals := range header {
		if strings.HasSuffix(key, "-bin") {
			for _, val := range vals {
				str := base64.StdEncoding.EncodeToString([]byte(val))
				d.ResponseHeader.Append(key, str)
			}
		} else {
			d.ResponseHeader.Append(key, vals...)
		}
	}

	if message != nil {
		bytes, err := protojson.Marshal(message)
		if err != nil {
			return err
		}

		d.Messages = append(d.Messages, &RpcMessage{
			From:    "server",
			Message: bytes,
		})
	}

	return nil
}

func (d *RpcData) String() string {
	bytes, _ := json.Marshal(d)
	return string(bytes)
}
