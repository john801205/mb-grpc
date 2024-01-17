package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"
	"google.golang.org/protobuf/types/dynamicpb"

	intGrpc "github.com/john801205/mb-grpc/internal/grpc"
	"github.com/john801205/mb-grpc/internal/mountebank"
	intProto "github.com/john801205/mb-grpc/internal/proto"
)

type Service struct {
	registry *intProto.Registry
	mbClient *mountebank.Client
}

func New(registry *intProto.Registry, mbClient *mountebank.Client) *Service {
	return &Service{
		registry: registry,
		mbClient: mbClient,
	}
}

type RpcMessage struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

type RpcData struct {
	Method    string        `json:"method"`
	Header    metadata.MD   `json:"header"`
	Messages  []*RpcMessage `json:"messages"`
}

type RpcStatusDetail struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

type RpcStatus struct {
	Code    codes.Code         `json:"code"`
	Message string             `json:"message"`
	Details []*RpcStatusDetail `json:"details"`
}

type RpcResponse struct {
	Header  metadata.MD     `json:"header,omitempty"`
	Message json.RawMessage `json:"message,omitempty"`
	Trailer metadata.MD     `json:"trailer,omitempty"`
	Status  *RpcStatus      `json:"status,omitempty"`
}

func (s *Service)HandleUnaryCall(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	intCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	method, ok := grpc.Method(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "method doesn't exist in context")
	}

	methodDesc, err := s.registry.FindMethodDescriptorByName(method)
	if err != nil {
		return nil, err
	}

	md, _ := metadata.FromIncomingContext(ctx)

	request := dynamicpb.NewMessage(methodDesc.Input())
	err = dec(request)
	if err != nil {
		return nil, err
	}

	value, err := protojson.Marshal(request)
	if err != nil {
		return nil, err
	}

	rpcData := &RpcData{
		Method: method,
		Header: md,
		Messages: []*RpcMessage{
			{
				Type: "request",
				Value: value,
			},
		},
	}

	log.Println("request", request, rpcData)

	mbResp, err := s.mbClient.GetResponse(intCtx, rpcData)
	if err != nil {
		return nil, err
	}

	if mbResp.Proxy != nil {
		log.Printf("proxy: %s", mbResp.Proxy.To)
		conn, err := grpc.Dial(mbResp.Proxy.To, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
		defer conn.Close()

		var header metadata.MD
		var trailer metadata.MD
		var statusDetails []*RpcStatusDetail

		intCtx = metadata.NewOutgoingContext(intCtx, md)
		resp := dynamicpb.NewMessage(methodDesc.Output())
		err = conn.Invoke(intCtx, method, request, resp, grpc.Header(&header), grpc.Trailer(&trailer))

		rpcStatus := status.Convert(err)
		for _, detail := range rpcStatus.Details() {
			switch detail := detail.(type) {
			case error:
				return nil, detail
			case proto.Message:
				bytes, err := protojson.Marshal(detail)
				if err != nil {
					return nil, err
				}

				statusDetails = append(statusDetails, &RpcStatusDetail{
					Type: string(proto.MessageName(detail)),
					Value: bytes,
				})
			default:
				return nil, fmt.Errorf("unexpected type in the status details")
			}
		}

		message, err := protojson.Marshal(resp)
		if err != nil {
			return nil, err
		}

		rpcResp := &RpcResponse{
			Header: header,
			Message: message,
			Trailer: trailer,
			Status: &RpcStatus{
				Code: rpcStatus.Code(),
				Message: rpcStatus.Message(),
				Details: statusDetails,
			},
		}

		mbResp, err = s.mbClient.SaveProxyResponse(intCtx, mbResp.ProxyCallbackURL, rpcResp)
		if err != nil {
			return nil, err
		}
	}

	var resp proto.Message
	rpcResp := &RpcResponse{}
	err = json.Unmarshal(mbResp.Response, rpcResp)
	if err != nil {
		return nil, err
	}

	err = grpc.SetHeader(ctx, rpcResp.Header)
	if err != nil {
		return nil, err
	}
	err = grpc.SetTrailer(ctx, rpcResp.Trailer)
	if err != nil {
		return nil, err
	}

	if len(rpcResp.Message) != 0 {
		resp = dynamicpb.NewMessage(methodDesc.Output())
		err = protojson.Unmarshal(rpcResp.Message, resp)
		if err != nil {
			return nil, err
		}
	}

	err = nil
	if rpcResp.Status != nil {
		st := status.New(rpcResp.Status.Code, rpcResp.Status.Message)
		if len(rpcResp.Status.Details) > 0 {
			var details []protoadapt.MessageV1
			for _, detail := range rpcResp.Status.Details {
				msgType, err := s.registry.FindMessageByName(detail.Type)
				if err != nil {
					return nil, err
				}

				msg := msgType.New().Interface()
				err = protojson.Unmarshal(detail.Value, msg)
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

		err = st.Err()
	}

	return resp, err
}

func (s *Service)HandleStreamCall(srv any, stream grpc.ServerStream) error {
	intCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	method, ok := grpc.MethodFromServerStream(stream)
	if !ok {
		return status.Error(codes.Internal, "method doesn't exist in context")
	}

	methodDesc, err := s.registry.FindMethodDescriptorByName(method)
	if err != nil {
		return err
	}

	var clientStream *intGrpc.ClientStream
	var proxyCallbackURL string
	serverStream := intGrpc.NewServerStream(intCtx, stream, methodDesc.Input())
	var lastMessage proto.Message

	rpcData := &RpcData{
		Method: method,
	}

	for {
		select {
		case request, ok := <-serverStream.Requests():
			if !ok {
				serverStream = nil
				continue
			}

			if request.Header != nil {
				rpcData.Header = metadata.Join(rpcData.Header, request.Header)
			}

			if request.Message != nil {
				data, err := protojson.Marshal(request.Message)
				if err != nil {
					return err
				}

				rpcData.Messages = append(rpcData.Messages, &RpcMessage{
					Type: "request",
					Value: data,
				})

				lastMessage = request.Message
			}

			if request.Status != nil {
				return request.Status.Err()
			}

		case response, ok := <-clientStream.Responses():
			if !ok {
				clientStream = nil
				continue
			}

			if proxyCallbackURL == "" {
				return fmt.Errorf("unexpected response from proxied server: %+v", response)
			}

			var err error
			var st *RpcStatus
			var message json.RawMessage

			if response.Message != nil {
				message, err = protojson.Marshal(response.Message)
				if err != nil {
					return err
				}
			}

			if response.Status != nil {
				var statusDetails []*RpcStatusDetail

				for _, detail := range response.Status.Details() {
					switch detail := detail.(type) {
					case error:
						return detail
					case proto.Message:
						bytes, err := protojson.Marshal(detail)
						if err != nil {
							return err
						}

						statusDetails = append(statusDetails, &RpcStatusDetail{
							Type: string(proto.MessageName(detail)),
							Value: bytes,
						})
					default:
						return fmt.Errorf("unexpected type in the status details")
					}
				}

				st = &RpcStatus{
					Code: response.Status.Code(),
					Message: response.Status.Message(),
					Details: statusDetails,
				}
			}

			rpcResp := &RpcResponse{
				Header: response.Header,
				Message: message,
				Trailer: response.Trailer,
				Status: st,
			}

			log.Println("client resp", rpcResp)

			_, err = s.mbClient.SaveProxyResponse(intCtx, proxyCallbackURL, rpcResp)
			if err != nil {
				return err
			}
			proxyCallbackURL = ""

			err = serverStream.SendMsg(response.Header, response.Trailer, response.Message)
			if err != nil {
				return err
			}

			if response.Message != nil {
				data, err := protojson.Marshal(response.Message)
				if err != nil {
					return err
				}

				rpcData.Messages = append(rpcData.Messages, &RpcMessage{
					Type: "response",
					Value: data,
				})
				lastMessage = response.Message
			}

			if response.Status != nil {
				return response.Status.Err()
			}
		}

		for {
			mbResp, err := s.mbClient.GetResponse(intCtx, rpcData)
			if err != nil {
				return err
			}

			if mbResp.Proxy != nil {
				if clientStream == nil {
					var err error
					streamDesc := grpc.StreamDesc{
						StreamName: method,
						Handler: s.HandleStreamCall,
						ServerStreams: methodDesc.IsStreamingServer(),
						ClientStreams: methodDesc.IsStreamingClient(),
					}
					clientStream, err = intGrpc.NewClientStream(
						intCtx, mbResp.Proxy.To,
						&streamDesc, method, methodDesc.Output(),
					)
					if err != nil {
						return err
					}
				}

				if len(rpcData.Messages) > 0 && rpcData.Messages[len(rpcData.Messages)-1].Type == "request" {
					err := clientStream.Forward(lastMessage)
					if err != nil {
						return err
					}
				}

				proxyCallbackURL = mbResp.ProxyCallbackURL
			}

			if mbResp.Response != nil {
				var resp proto.Message
				rpcResp := &RpcResponse{}
				err = json.Unmarshal(mbResp.Response, rpcResp)
				if err != nil {
					return err
				}

				if len(rpcResp.Message) != 0 {
					resp = dynamicpb.NewMessage(methodDesc.Output())
					err = protojson.Unmarshal(rpcResp.Message, resp)
					if err != nil {
						return err
					}
				}

				err = serverStream.SendMsg(rpcResp.Header, rpcResp.Trailer, resp)
				if err != nil {
					return err
				}

				if resp != nil {
					rpcData.Messages = append(rpcData.Messages, &RpcMessage{
						Type: "response",
						Value: rpcResp.Message,
					})
				}

				if rpcResp.Status != nil {
					st := status.New(rpcResp.Status.Code, rpcResp.Status.Message)
					if len(rpcResp.Status.Details) > 0 {
						var details []protoadapt.MessageV1
						for _, detail := range rpcResp.Status.Details {
							msgType, err := s.registry.FindMessageByName(detail.Type)
							if err != nil {
								return err
							}

							msg := msgType.New().Interface()
							err = protojson.Unmarshal(detail.Value, msg)
							if err != nil {
								return err
							}

							details = append(details, protoadapt.MessageV1Of(msg))

						}

						st, err = st.WithDetails(details...)
						if err != nil {
							return err
						}
					}

					return st.Err()
				}

				if resp != nil {
					continue
				}
			}

			break
		}
	}

	return fmt.Errorf("shoud not be here")
}
