package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

var (
	methodMap = map[string]protoreflect.MethodDescriptor{}
	mountebankClient = NewMountebankClient("http://localhost:2525/imposters/5567/_requests")
)

type ProtoMessage struct {
	proto.Message
}

func (r *ProtoMessage) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(r.Message)
}

func (r *ProtoMessage) UnmarshalJSON(data []byte) error {
	return protojson.Unmarshal(data, r.Message)
}

type ProtoData struct {
	Type  string        `json:"type"`
	Value *ProtoMessage `json:"value,omitempty"`
	Error error         `json:"error,omitempty"`
}

type MyBody struct {
	Request MyRequest `json:"request"`
}

type MyRequest struct {
	Method   string          `json:"method"`
	Metadata metadata.MD     `json:"metadata,omitempty"`
	Data     []*ProtoData    `json:"data"`
}

type ProxyTo struct {
	To string `json:"to"`
}

type ProxyToResponse struct {
	Response *MyResponse `json:"proxyResponse"`
}

type MyResponse struct {
	Metadata metadata.MD   `json:"metadata,omitempty"`
	Data     *ProtoMessage `json:"data,omitempty"`
	Error    error         `json:"error,omitempty"`
}

type MyResponseBody struct {
	// standard response
	Response *MyResponse `json:"response"`

	// proxy response
	Proxy    *ProxyTo   `json:"proxy"`
	CallbackURL string  `json:"callbackURL"`
}

type MyServer struct {
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
	Header  metadata.MD     `json:"header"`
	Message json.RawMessage `json:"message"`
	Trailer metadata.MD     `json:"trailer"`
	Status  *RpcStatus      `json:"status"`
}

func (s *MyServer)HandleUnaryCall(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	intCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	method, ok := grpc.Method(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "method doesn't exist in context")
	}

	methodDesc, ok := methodMap[method]
	if !ok {
		return nil, status.Error(codes.Internal, "unknown method descriptor")
	}

	md, _ := metadata.FromIncomingContext(ctx)

	request := dynamicpb.NewMessage(methodDesc.Input())
	err := dec(request)
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
	Message any
	Trailer metadata.MD
	Error   error
}



/*
	var clientStream *ClientStream
	var proxyCallbackURL string
	serverStream := NewServerStream(stream, methodDesc.Input())

	for {
		select {
		case request, ok := <-serverStream.Requests():
			if !ok || request.Error == io.EOF {
				serverStream = nil
				continue
			}
		case response, ok := <-clientStream.Responses():
			if !ok {
				clientStream = nil
				continue
			}

			if callbackURL == "" {
				return fmt.Errorf("unexpected response from proxied server: %+v", response)
			}
		}
	}

*/

func (s *MyServer)HandleStreamCall(srv any, stream grpc.ServerStream) error {
	intCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	method, ok := grpc.MethodFromServerStream(stream)
	if !ok {
		return status.Error(codes.Internal, "method doesn't exist in context")
	}

	methodDesc, ok := methodMap[method]
	if !ok {
		return status.Error(codes.Internal, "unknown method descriptor")
	}

	md, _ := metadata.FromIncomingContext(stream.Context())
	log.Printf("headers: %+v", md)

	requestCh := make(chan *ProtoData)
	var requests []*ProtoData
	var clientStream grpc.ClientStream
	var lastCallback string

	intCtx = metadata.NewOutgoingContext(intCtx, md)

	go func() {
		// requestCh <- &ProtoData{}

		for intCtx.Err() == nil {
			request := dynamicpb.NewMessage(methodDesc.Input())
			err := stream.RecvMsg(request)
			if err == io.EOF {
				log.Println("client EOF")
				return
			} else if err != nil {
				log.Println("client error", err)
				requestCh <- &ProtoData{
					Error: err,
				}
				return
			} else {
				log.Println("client request", request)
				requestCh <- &ProtoData{
					Type: "request",
					Value: &ProtoMessage{request},
				}
			}
		}

		if err := intCtx.Err(); err != context.Canceled {
			requestCh <- &ProtoData{
				Error: err,
			}
		}
	} ()

	for request := range requestCh {
		for request != nil {
			log.Println("request", request)

			if request.Error != nil {
				return request.Error
			}

			if request.Type == "response"  && lastCallback != "" {
				if lastCallback == "" {
					return fmt.Errorf("unexpected response from proxied server: %+v", request)
				}

				proxyResponse := ProxyToResponse{
					Response: &MyResponse{Data: &ProtoMessage{request.Value}},
				}

				log.Println("proxy response")

				data, err := json.Marshal(proxyResponse)
				if err != nil {
					return err
				}

				log.Println("proxy response", lastCallback, string(data))

				req, err := http.NewRequestWithContext(
					intCtx,
					http.MethodPost,
					lastCallback,
					bytes.NewReader(data),
				)
				if err != nil {
					return err
				}

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return err
				}

				respBody, err := io.ReadAll(resp.Body)
				if err != nil {
					return err
				}

				log.Printf("proxy response: %s", string(respBody))
				err = stream.SendMsg(request.Value.Message)
				if err != nil {
					return err
				}

				lastCallback = ""
			}

			if request.Type != "" {
				requests = append(requests, request)
			}

			log.Printf("sequence %d: %+v", len(requests), request)

			myrequest := MyBody{
				Request: MyRequest{
					Method: method,
					Metadata: md,
					Data: requests,
				},
			}

			data, err := json.Marshal(myrequest)
			if err != nil {
				return err
			}

			log.Println(string(data))

			req, err := http.NewRequestWithContext(
				intCtx,
				http.MethodPost,
				"http://localhost:2525/imposters/5567/_requests",
				bytes.NewReader(data),
			)
			if err != nil {
				return err
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			myResp := dynamicpb.NewMessage(methodDesc.Output())
			parsedResp := &MyResponseBody{
				Response: &MyResponse{Data: &ProtoMessage{myResp}},
			}

			err = json.Unmarshal(respBody, parsedResp)
			if err != nil {
				return err
			}

			myDict := map[string]json.RawMessage{}
			myData := map[string]json.RawMessage{}

			err = json.Unmarshal(respBody, &myDict)
			if err != nil {
				return err
			}

			if len(myDict["response"]) != 0 {
				err = json.Unmarshal(myDict["response"], &myData)
				if err != nil {
					return err
				}
			}

			if parsedResp.Proxy != nil {
				lastCallback = parsedResp.CallbackURL

				if clientStream == nil {
					conn, err := grpc.Dial(parsedResp.Proxy.To, grpc.WithTransportCredentials(insecure.NewCredentials()))
					if err != nil {
						return err
					}
					defer conn.Close()

					streamDesc := grpc.StreamDesc{
						StreamName: method,
						Handler: s.HandleStreamCall,
						ServerStreams: methodDesc.IsStreamingServer(),
						ClientStreams: methodDesc.IsStreamingClient(),
					}
					clientStream, err = conn.NewStream(intCtx, &streamDesc, method)
					if err != nil {
						return err
					}

					go func() {
						for intCtx.Err() == nil {
							resp := dynamicpb.NewMessage(methodDesc.Output())
							err := clientStream.RecvMsg(resp)
							if err == io.EOF {
								log.Println("proxy EOF")
								return
							} else if err != nil {
								log.Println("proxy error", err)
								requestCh <- &ProtoData{
									Type: "Proxy",
									Error: err,
								}
								return
							} else {
								log.Println("proxy response", resp)
								requestCh <- &ProtoData{
									Type: "response",
									Value: &ProtoMessage{resp},
								}
							}
						}

						if err := intCtx.Err(); err != context.Canceled {
							requestCh <- &ProtoData{
								Type: "Proxy",
								Error: err,
							}
						}
					} ()
				}

				if request.Type == "request" {
					err = clientStream.SendMsg(request.Value.Message)
					if err != nil {
						return err
					}
				}

				request = nil
			} else if len(myData["data"]) != 0 && parsedResp.Response != nil {
				err := stream.SendMsg(myResp)
				if err != nil {
					return err
				}

				request = &ProtoData{
					Type: "response",
					Value: &ProtoMessage{myResp},
				}
			} else {
				request = nil
			}
		}
	}

	return fmt.Errorf("shoud not be here")
}

func main() {
	log.Println(os.Args)

	myServer := new(MyServer)

	grpcServer := grpc.NewServer()

	data, err := os.ReadFile("/tmp/grpc-go/examples/route_guide/routeguide/route_guide.desc")
	if err != nil {
		log.Fatal(err)
	}

	desc := new(descriptorpb.FileDescriptorSet)
	err = proto.Unmarshal(data, desc)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range desc.GetFile() {
		fd, err := protodesc.NewFile(file, protoregistry.GlobalFiles)
		if err != nil {
			log.Fatal(err)
		}

		err = protoregistry.GlobalFiles.RegisterFile(fd);
		if err != nil {
			log.Fatal(err)
		}

		for i := 0; i < fd.Messages().Len(); i++ {
			msg := fd.Messages().Get(i)
			err := protoregistry.GlobalTypes.RegisterMessage(dynamicpb.NewMessageType(msg))
			if err != nil {
				log.Fatal(err)
			}
		}

		for i := 0; i < fd.Services().Len(); i++ {
			service := fd.Services().Get(i)

			serviceDesc := grpc.ServiceDesc{
				ServiceName: string(service.FullName()),
				HandlerType: (*any)(nil),
				Metadata: fd.Path(),
			}

			for j := 0; j < service.Methods().Len(); j++ {
				method := service.Methods().Get(j)

				if method.IsStreamingClient() || method.IsStreamingServer() {
					streamDesc := grpc.StreamDesc{
						StreamName: string(method.Name()),
						Handler: myServer.HandleStreamCall,
						ServerStreams: method.IsStreamingServer(),
						ClientStreams: method.IsStreamingClient(),
					}
					serviceDesc.Streams = append(serviceDesc.Streams, streamDesc)
				} else {
					methodDesc := grpc.MethodDesc{
						MethodName: string(method.Name()),
						Handler: myServer.HandleUnaryCall,
					}
					serviceDesc.Methods = append(serviceDesc.Methods, methodDesc)
				}

				methodMap["/" + string(service.FullName()) + "/" + string(method.Name())] = method
			}

			grpcServer.RegisterService(&serviceDesc, myServer)
		}
	}

	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":5567")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()
	fmt.Println("grpc")
	grpcServer.Serve(lis)
}
