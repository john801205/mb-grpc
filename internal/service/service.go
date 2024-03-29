package service

import (
	"context"
	"errors"
	"fmt"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"

	intGrpc "github.com/john801205/mb-grpc/internal/grpc"
	"github.com/john801205/mb-grpc/internal/log"
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

func (s *Service) HandleUnaryCall(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
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
	rpcData := mountebank.NewRpcData(method)
	err = rpcData.AddRequestData(md, request)
	if err != nil {
		return nil, err
	}
	log.Debugf("unary client request: %s", rpcData)

	mbResp, err := s.mbClient.GetResponse(intCtx, rpcData, methodDesc.Output())
	if err != nil {
		return nil, err
	}

	if mbResp.Proxy != nil {
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
		rpcResp := &mountebank.RpcResponse{
			Header:  header,
			Message: resp,
			Trailer: trailer,
			Status:  status.Convert(err),
		}
		log.Debugf("proxied server response: %s", rpcResp)

		mbResp, err = s.mbClient.SaveProxyResponse(intCtx, mbResp.ProxyCallbackURL, rpcResp, methodDesc.Output())
		if err != nil {
			return nil, err
		}
	}

	if mbResp.Response == nil {
		return nil, errors.New("nil response from mountebank")
	}

	err = grpc.SetHeader(ctx, mbResp.Response.Header)
	if err != nil {
		return nil, err
	}
	err = grpc.SetTrailer(ctx, mbResp.Response.Trailer)
	if err != nil {
		return nil, err
	}

	return mbResp.Response.Message, mbResp.Response.Status.Err()
}

func (s *Service) HandleStreamCall(srv any, stream grpc.ServerStream) error {
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
	var lastUnforwardedClientMessage proto.Message

	rpcData := mountebank.NewRpcData(method)

	for {
		select {
		case request := <-serverStream.Requests():
			if request.Error != nil {
				serverStream.WaitFinished()
			}

			log.Debugf("stream client request: %s", request)
			if request.Error == io.EOF {
				err := clientStream.CloseSend()
				if err != nil {
					return err
				}
				continue
			} else if request.Error != nil {
				return request.Error
			}

			if proxyCallbackURL != "" {
				rpcResp := &mountebank.RpcResponse{}
				_, err := s.mbClient.SaveProxyResponse(
					intCtx,
					proxyCallbackURL,
					rpcResp,
					methodDesc.Output(),
				)
				if err != nil {
					return err
				}
				proxyCallbackURL = ""
			}

			err := rpcData.AddRequestData(request.Header, request.Message)
			if err != nil {
				return err
			}

			lastUnforwardedClientMessage = request.Message

		case response := <-clientStream.Responses():
			if proxyCallbackURL == "" {
				return fmt.Errorf("unexpected response from proxied server: %+v", response)
			}

			if response.Error != nil {
				clientStream.WaitFinished()
			}
			log.Debugf("proxied server response: %s", response)

			var st *status.Status
			if response.Error == io.EOF {
				st = status.New(codes.OK, "")
			} else if response.Error != nil {
				st = status.Convert(response.Error)
			}

			rpcResp := &mountebank.RpcResponse{
				Header:  response.Header,
				Message: response.Message,
				Trailer: response.Trailer,
				Status:  st,
			}

			_, err := s.mbClient.SaveProxyResponse(intCtx, proxyCallbackURL, rpcResp, methodDesc.Output())
			if err != nil {
				return err
			}
			proxyCallbackURL = ""

			err = serverStream.SendMsg(response.Header, response.Trailer, response.Message)
			if err != nil {
				return err
			}

			if st != nil {
				return st.Err()
			}

			err = rpcData.AddResponseData(response.Header, response.Message)
			if err != nil {
				return err
			}

			lastUnforwardedClientMessage = nil
		}

		for {
			mbResp, err := s.mbClient.GetResponse(intCtx, rpcData, methodDesc.Output())
			if err != nil {
				return err
			}

			if mbResp.Proxy != nil {
				if clientStream == nil {
					var err error
					streamDesc := grpc.StreamDesc{
						StreamName:    method,
						Handler:       s.HandleStreamCall,
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

				if lastUnforwardedClientMessage != nil {
					err := clientStream.Forward(lastUnforwardedClientMessage)
					if err != nil {
						return err
					}
					lastUnforwardedClientMessage = nil
				}

				proxyCallbackURL = mbResp.ProxyCallbackURL
				break
			} else if !mbResp.Response.IsEmpty() {
				lastUnforwardedClientMessage = nil

				err = serverStream.SendMsg(
					mbResp.Response.Header,
					mbResp.Response.Trailer,
					mbResp.Response.Message,
				)
				if err != nil {
					return err
				}
				if mbResp.Response.Status != nil {
					return mbResp.Response.Status.Err()
				}

				err = rpcData.AddResponseData(mbResp.Response.Header, mbResp.Response.Message)
				if err != nil {
					return err
				}

				continue
			} else {
				lastUnforwardedClientMessage = nil
				break
			}
		}
	}

	return status.Error(codes.Internal, "no response from proxied server or mountebank any more")
}
