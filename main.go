package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"

	intProto "github.com/john801205/mb-grpc/internal/proto"
)

var (
	mountebankClient *MountebankClient
)

type MyServer struct {
	registry *intProto.ProtoRegistry
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

type RpcStatus struct {
	Code    codes.Code `json:"code"`
	Message string     `json:"message"`
}

type RpcResponse struct {
	Header  metadata.MD     `json:"header,omitempty"`
	Message json.RawMessage `json:"message,omitempty"`
	Trailer metadata.MD     `json:"trailer,omitempty"`
	Status  *RpcStatus      `json:"status,omitempty"`
}

func (s *MyServer)HandleUnaryCall(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
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


	mbResp, err := mountebankClient.GetResponse(intCtx, rpcData)
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

		intCtx = metadata.NewOutgoingContext(intCtx, md)
		resp := dynamicpb.NewMessage(methodDesc.Output())
		err = conn.Invoke(intCtx, method, request, resp, grpc.Header(&header), grpc.Trailer(&trailer))

		rpcStatus := status.Convert(err)

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
			},
		}

		mbResp, err = mountebankClient.SaveProxyResponse(intCtx, mbResp.ProxyCallbackURL, rpcResp)
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
		err = status.Error(rpcResp.Status.Code, rpcResp.Status.Message)
	}

	return resp, err
}

type StreamData struct {
	Header  metadata.MD
	Message proto.Message
	Trailer metadata.MD
	Status  *status.Status
}

func (s *MyServer)HandleStreamCall(srv any, stream grpc.ServerStream) error {
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

	var clientStream *ClientStream
	var proxyCallbackURL string
	serverStream := NewServerStream(intCtx, stream, methodDesc.Input())
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
				st = &RpcStatus{
					Code: response.Status.Code(),
					Message: response.Status.Message(),
				}
			}

			rpcResp := &RpcResponse{
				Header: response.Header,
				Message: message,
				Trailer: response.Trailer,
				Status: st,
			}

			log.Println("client resp", rpcResp)

			_, err = mountebankClient.SaveProxyResponse(intCtx, proxyCallbackURL, rpcResp)
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
			mbResp, err := mountebankClient.GetResponse(intCtx, rpcData)
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
					clientStream, err = NewClientStream(
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
					return status.Error(rpcResp.Status.Code, rpcResp.Status.Message)
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

type Config struct {
	Port int     `json:"port"`
	Host *string `json:"host"`

	CallbackURLTemplate string `json:"callbackURLTemplate"`

	Options *ConfigOptions `json:"options"`
}

type ConfigOptions struct {
	Protoc *ProtocOptions `json:"protoc"`
}

type ProtocOptions struct {
	ImportDirs map[string]string `json:"importDirs"`
	ProtoFiles map[string]string `json:"protoFiles"`
}

func main() {
	log.Println(os.Args)

	config := &Config{}
	err := json.Unmarshal([]byte(os.Args[1]), config)
	if err != nil {
		log.Fatal(err)
	}

	var importDirs []string
	var protoFiles []string

	if config.Options != nil && config.Options.Protoc != nil {
		for _, importDir := range config.Options.Protoc.ImportDirs {
			importDirs = append(importDirs, importDir)
		}
		for _, protoFile := range config.Options.Protoc.ProtoFiles {
			protoFiles = append(protoFiles, protoFile)
		}
	}

	registry, err := intProto.Load(importDirs, protoFiles)
	if err != nil {
		log.Fatal(err)
	}

	myServer := &MyServer{registry: registry}
	grpcServer := grpc.NewServer()

	registry.RegisterGenericService(grpcServer, myServer)
	reflection.Register(grpcServer)

	host := ""
	if config.Host != nil {
		host = *config.Host
	}
	port := config.Port

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()

	callbackURL := strings.Replace(config.CallbackURLTemplate, ":port", strconv.Itoa(port), 1)
	mountebankClient = NewMountebankClient(callbackURL)

	fmt.Println("grpc")
	grpcServer.Serve(lis)
}
