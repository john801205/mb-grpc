package mountebank

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

type RpcResponse struct {
	Header  metadata.MD
	Message proto.Message
	Trailer metadata.MD
	Status  *status.Status
}

type rpcStatusDetail struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message"`
}

type rpcStatus struct {
	Code    codes.Code         `json:"code"`
	Message string             `json:"message"`
	Details []*rpcStatusDetail `json:"details,omitempty"`
}

type rpcResponse struct {
	Header  metadata.MD     `json:"header,omitempty"`
	Message json.RawMessage `json:"message,omitempty"`
	Trailer metadata.MD     `json:"trailer,omitempty"`
	Status  *rpcStatus      `json:"status,omitempty"`
}

func convert(r *rpcResponse, desc protoreflect.MessageDescriptor) (*RpcResponse, error) {
	if r == nil {
		return nil, nil
	}

	var message proto.Message
	if len(r.Message) != 0 {
		message = dynamicpb.NewMessage(desc)
		err := protojson.Unmarshal(r.Message, message)
		if err != nil {
			return nil, err
		}
	}

	var st *status.Status
	if r.Status != nil {
		st = status.New(r.Status.Code, r.Status.Message)
		if len(r.Status.Details) != 0 {
			var details []protoadapt.MessageV1
			var err error

			for _, detail := range r.Status.Details {
				name := protoreflect.FullName(detail.Type)
				msgType, err := protoregistry.GlobalTypes.FindMessageByName(name)
				if err != nil {
					return nil, err
				}

				msg := msgType.New().Interface()
				err = protojson.Unmarshal(detail.Message, msg)
				if err != nil {
					return nil, err
				}

				details = append(details, protoadapt.MessageV1Of(msg))

			}

			st, err = st.WithDetails(details...)
			if err != nil {
				return nil, err
			}
		}
	}

	var header, trailer metadata.MD
	if len(r.Header) != 0 {
		header = metadata.New(nil)
	}
	for key, vals := range r.Header {
		if strings.HasSuffix(key, "-bin") {
			for _, val := range vals {
				decoded, err := base64.StdEncoding.DecodeString(val)
				if err != nil {
					return nil, err
				}

				header.Append(key, string(decoded))
			}
		} else {
			header.Append(key, vals...)
		}
	}
	if len(r.Trailer) != 0 {
		trailer = metadata.New(nil)
	}
	for key, vals := range r.Trailer {
		if strings.HasSuffix(key, "-bin") {
			for _, val := range vals {
				decoded, err := base64.StdEncoding.DecodeString(val)
				if err != nil {
					return nil, err
				}

				trailer.Append(key, string(decoded))
			}
		} else {
			trailer.Append(key, vals...)
		}
	}

	return &RpcResponse{
		Header:  header,
		Message: message,
		Trailer: trailer,
		Status:  st,
	}, nil
}

func (r *RpcResponse) MarshalJSON() ([]byte, error) {
	var message json.RawMessage
	var err error
	if r.Message != nil {
		message, err = protojson.Marshal(r.Message)
		if err != nil {
			return nil, err
		}
	}

	var st *rpcStatus
	if r.Status != nil {
		var details []*rpcStatusDetail
		for _, detail := range r.Status.Details() {
			switch dd := detail.(type) {
			case error:
				return nil, dd
			case proto.Message:
				bytes, err := protojson.Marshal(dd)
				if err != nil {
					return nil, err
				}

				details = append(details, &rpcStatusDetail{
					Type:    string(proto.MessageName(dd)),
					Message: bytes,
				})
			default:
				return nil, errors.New("unexpected type inside the status details")
			}
		}

		st = &rpcStatus{
			Code:    r.Status.Code(),
			Message: r.Status.Message(),
			Details: details,
		}
	}

	header := r.Header.Copy()
	for key, vals := range r.Header {
		if strings.HasSuffix(key, "-bin") {
			header.Delete(key)
			for _, val := range vals {
				str := base64.StdEncoding.EncodeToString([]byte(val))
				header.Append(key, str)
			}
		}
	}

	trailer := r.Trailer.Copy()
	for key, vals := range r.Trailer {
		if strings.HasSuffix(key, "-bin") {
			trailer.Delete(key)
			for _, val := range vals {
				str := base64.StdEncoding.EncodeToString([]byte(val))
				trailer.Append(key, str)
			}
		}
	}
	trailer.Delete("grpc-status-details-bin")
	resp := &rpcResponse{
		Header:  header,
		Message: message,
		Trailer: trailer,
		Status:  st,
	}

	return json.Marshal(resp)
}

func (r *RpcResponse) IsEmpty() bool {
	if r == nil {
		return true
	}
	return len(r.Header) == 0 && len(r.Trailer) == 0 && r.Message == nil && r.Status == nil
}

func (r *RpcResponse) String() string {
	bytes, _ := json.Marshal(r)
	return string(bytes)
}
