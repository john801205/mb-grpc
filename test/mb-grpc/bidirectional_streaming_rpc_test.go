package mbgrpc

import (
	"context"
	"fmt"
	"io"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pb "github.com/john801205/mb-grpc/test/mb-grpc/pingpong"
)

type bidirectionalStreamingRPCServer struct {
	pb.UnimplementedServiceServer
}

func (s *bidirectionalStreamingRPCServer) PingPingPongPong(stream pb.Service_PingPingPongPongServer) error {
	header := metadata.Pairs("symbol", "-_.~!#$&'()*+,/:;=?@[]%20")
	stream.SetHeader(header)
	trailer := metadata.Pairs("symbol", "-_.~!#$&'()*+,/:;=?@[]%20")
	stream.SetTrailer(trailer)

	// retrieve header
	header, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return fmt.Errorf("expect flag header inside context")
	}
	if len(header.Get("flag")) != 1 {
		return fmt.Errorf("unexpected flag in header: %v", header.Get("flag"))
	}
	flag := header.Get("flag")[0]

	if strings.Contains(flag, "pingpongpongpingpingping") {
		for i := 0; i < 1; i++ {
			_, err := stream.Recv()
			if err != nil {
				return err
			}
		}
		for i := 0; i < 2; i++ {
			err := stream.Send(&pb.Pong{Pong: "你好，世界"})
			if err != nil {
				return err
			}
		}
		for i := 0; i < 3; i++ {
			_, err := stream.Recv()
			if err != nil {
				return err
			}
		}
	} else if strings.Contains(flag, "pongpingpingpongpongpong") {
		for i := 0; i < 1; i++ {
			err := stream.Send(&pb.Pong{Pong: "你好，世界"})
			if err != nil {
				return err
			}
		}
		for i := 0; i < 2; i++ {
			_, err := stream.Recv()
			if err != nil {
				return err
			}
		}
		for i := 0; i < 3; i++ {
			err := stream.Send(&pb.Pong{Pong: "你好，世界"})
			if err != nil {
				return err
			}
		}
	}

	if strings.Contains(flag, "success") {
		return nil
	} else if strings.Contains(flag, "failure") {
		st := status.New(codes.Aborted, "message")
		st, err := st.WithDetails(&pb.Pong{Pong: "你好，世界"})
		if err != nil {
			return err
		}
		return st.Err()
	}

	return status.Errorf(codes.Unknown, "unexpected flag")
}

func testBidirectionalStreamingRPC(ctx context.Context, t *testing.T) {
	conn, err := grpc.Dial("localhost:5568", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		err := conn.Close()
		if err != nil {
			t.Error(err)
		}
	})

	client := pb.NewServiceClient(conn)

	tests := []struct{
		name         string
		header       metadata.MD
		wantMessages []proto.Message
		wantHeader   metadata.MD
		wantTrailer  metadata.MD
		wantErr      error
	} {
		{
			name: "Success - pingpongpongpingpingping",
			header: metadata.Pairs(
				"flag", "success - pingpongpongpingpingping",
			),
			wantMessages: []proto.Message{
				&pb.Ping{Ping: "success"},
				&pb.Pong{Pong: "你好，世界"},
				&pb.Pong{Pong: "你好，世界"},
				&pb.Ping{Ping: "success"},
				&pb.Ping{Ping: "success"},
				&pb.Ping{Ping: "success"},
			},
			wantHeader: metadata.New(map[string]string{
				"symbol": "-_.~!#$&'()*+,/:;=?@[]%20",
				"content-type": "application/grpc",
			}),
			wantTrailer: metadata.New(map[string]string{
				"symbol": "-_.~!#$&'()*+,/:;=?@[]%20",
			}),
			wantErr: io.EOF,
		},
		{
			name: "Success - pongpingpingpongpongpong",
			header: metadata.Pairs(
				"flag", "success - pongpingpingpongpongpong",
			),
			wantMessages: []proto.Message{
				&pb.Pong{Pong: "你好，世界"},
				&pb.Ping{Ping: "success"},
				&pb.Ping{Ping: "success"},
				&pb.Pong{Pong: "你好，世界"},
				&pb.Pong{Pong: "你好，世界"},
				&pb.Pong{Pong: "你好，世界"},
			},
			wantHeader: metadata.New(map[string]string{
				"symbol": "-_.~!#$&'()*+,/:;=?@[]%20",
				"content-type": "application/grpc",
			}),
			wantTrailer: metadata.New(map[string]string{
				"symbol": "-_.~!#$&'()*+,/:;=?@[]%20",
			}),
			wantErr: io.EOF,
		},
		{
			name: "Failure - pingpongpongpingpingping",
			header: metadata.Pairs(
				"flag", "failure - pingpongpongpingpingping",
			),
			wantMessages: []proto.Message{
				&pb.Ping{Ping: "failure"},
				&pb.Pong{Pong: "你好，世界"},
				&pb.Pong{Pong: "你好，世界"},
				&pb.Ping{Ping: "failure"},
				&pb.Ping{Ping: "failure"},
				&pb.Ping{Ping: "failure"},
			},
			wantHeader: metadata.New(map[string]string{
				"symbol": "-_.~!#$&'()*+,/:;=?@[]%20",
				"content-type": "application/grpc",
			}),
			wantTrailer: func() metadata.MD {
				md := metadata.Pairs(
					"symbol", "-_.~!#$&'()*+,/:;=?@[]%20",
				)
				st := status.New(codes.Aborted, "message")
				st, err := st.WithDetails(&pb.Pong{Pong: "你好，世界"})
				if err != nil {
					t.Fatal(err)
				}
				bytes, err := proto.Marshal(st.Proto())
				if err != nil {
					t.Fatal(err)
				}
				md.Append("grpc-status-details-bin", string(bytes))
				return md
			}(),
			wantErr: func() error {
				st := status.New(codes.Aborted, "message")
				st, err := st.WithDetails(&pb.Pong{Pong: "你好，世界"})
				if err != nil {
					t.Fatal(err)
				}
				return st.Err()
			}(),
		},
		{
			name: "Failure - pongpingpingpongpongpong",
			header: metadata.Pairs(
				"flag", "Failure - pongpingpingpongpongpong",
			),
			wantMessages: []proto.Message{
				&pb.Pong{Pong: "你好，世界"},
				&pb.Ping{Ping: "Failure"},
				&pb.Ping{Ping: "Failure"},
				&pb.Pong{Pong: "你好，世界"},
				&pb.Pong{Pong: "你好，世界"},
				&pb.Pong{Pong: "你好，世界"},
			},
			wantHeader: metadata.New(map[string]string{
				"symbol": "-_.~!#$&'()*+,/:;=?@[]%20",
				"content-type": "application/grpc",
			}),
			wantTrailer: func() metadata.MD {
				md := metadata.Pairs(
					"symbol", "-_.~!#$&'()*+,/:;=?@[]%20",
				)
				st := status.New(codes.Aborted, "message")
				st, err := st.WithDetails(&pb.Pong{Pong: "你好，世界"})
				if err != nil {
					t.Fatal(err)
				}
				bytes, err := proto.Marshal(st.Proto())
				if err != nil {
					t.Fatal(err)
				}
				md.Append("grpc-status-details-bin", string(bytes))
				return md
			}(),
			wantErr: func() error {
				st := status.New(codes.Aborted, "message")
				st, err := st.WithDetails(&pb.Pong{Pong: "你好，世界"})
				if err != nil {
					t.Fatal(err)
				}
				return st.Err()
			}(),
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			var header, trailer metadata.MD
			var got []proto.Message

			ctx := metadata.NewOutgoingContext(ctx, tt.header)
			stream, err := client.PingPingPongPong(
				ctx,
				grpc.Header(&header),
				grpc.Trailer(&trailer),
			)
			if err == nil {
				for _, message := range tt.wantMessages {
					switch msg := message.(type) {
					case *pb.Ping:
						err = stream.Send(msg)
						if err != nil {
							break
						}
						got = append(got, msg)
					case *pb.Pong:
						msg, err = stream.Recv()
						if err != nil {
							break
						}
						got = append(got, msg)
					default:
						t.Fatal("unexpected message type")
					}
				}

				if err == nil {
					// to receive the io.EOF error
					_, err = stream.Recv()
				}
			}


			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("error not equal, want: %s, got: %s", tt.wantErr, err)
			}
			if !reflect.DeepEqual(header, tt.wantHeader) {
				t.Errorf("header not equal, want: %s, got: %s", tt.wantHeader, header)
			}
			if !isSliceEqual(got, tt.wantMessages) {
				t.Errorf(
					"response not equal, want: %s, got: %s",
					formatSlice(tt.wantMessages), formatSlice(got),
				)
			}
			if !reflect.DeepEqual(trailer, tt.wantTrailer) {
				t.Errorf("trailer not equal, want: %s, got: %s", tt.wantTrailer, trailer)
			}
		})
	}
}

func TestBidirectionalStreamingRPC(t *testing.T) {
	ctx := context.Background()
	testBidirectionalStreamingRPC(ctx, t)
}

func TestBidirectionalStreamingRPCProxyOnce(t *testing.T) {
	lis, err := net.Listen("tcp", "localhost:5569")
	if err != nil {
		t.Fatal(err)
	}
	s := &bidirectionalStreamingRPCServer{}
	grpcServer := grpc.NewServer()
	pb.RegisterServiceServer(grpcServer, s)
	go func() {
		err := grpcServer.Serve(lis)
		if err != nil {
			t.Error(err)
		}
	}()

	time.Sleep(2 * time.Second)

	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, "proxy-mode", "proxyOnce")
	testBidirectionalStreamingRPC(ctx, t)
	grpcServer.GracefulStop()
	testBidirectionalStreamingRPC(ctx, t)
}
