package mbgrpc

import (
	"context"
	"net"
	"reflect"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	pb "github.com/john801205/mb-grpc/test/mb-grpc/pingpong"
)

type clientStreamingRPCServer struct {
	pb.UnimplementedServiceServer
}

func (s *clientStreamingRPCServer) PingPingPong(stream pb.Service_PingPingPongServer) error {
	header := metadata.Pairs("symbol", "-_.~!#$&'()*+,/:;=?@[]%20")
	stream.SetHeader(header)
	trailer := metadata.Pairs("symbol", "-_.~!#$&'()*+,/:;=?@[]%20")
	stream.SetTrailer(trailer)

	ping, err := stream.Recv()
	if err != nil {
		return err
	}

	if ping.Ping == "success" {
		_, err := stream.Recv()
		if err != nil {
			return err
		}

		return stream.SendAndClose(&pb.Pong{Pong: "你好，世界"})
	} else if ping.Ping == "failure" {
		_, err := stream.Recv()
		if err != nil {
			return err
		}

		_, err = stream.Recv()
		if err != nil {
			return err
		}

		st := status.New(codes.Internal, "message")
		st, err = st.WithDetails(&pb.Pong{Pong: "你好，世界"})
		if err != nil {
			return err
		}
		return st.Err()
	}

	return status.Errorf(codes.Unknown, "unexpected ping")
}

func testClientStreamingRPC(ctx context.Context, t *testing.T) {
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
		name        string
		requests    []*pb.Ping
		want        *pb.Pong
		wantHeader  metadata.MD
		wantTrailer metadata.MD
		wantErr     error
	} {
		{
			name: "Success",
			requests: []*pb.Ping{
				{Ping: "success"},
				{Ping: "success"},
			},
			want: &pb.Pong{Pong: "你好，世界"},
			wantHeader: metadata.New(map[string]string{
				"symbol": "-_.~!#$&'()*+,/:;=?@[]%20",
				"content-type": "application/grpc",
			}),
			wantTrailer: metadata.New(map[string]string{
				"symbol": "-_.~!#$&'()*+,/:;=?@[]%20",
			}),
			wantErr: nil,
		},
		{
			name: "Failure",
			requests: []*pb.Ping{
				{Ping: "failure"},
				{Ping: "failure"},
				{Ping: "failure"},
			},
			want: nil,
			wantHeader: metadata.New(map[string]string{
				"symbol": "-_.~!#$&'()*+,/:;=?@[]%20",
				"content-type": "application/grpc",
			}),
			wantTrailer: func() metadata.MD {
				md := metadata.New(map[string]string{
					"symbol": "-_.~!#$&'()*+,/:;=?@[]%20",
				})
				st := status.New(codes.Internal, "message")
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
				st := status.New(codes.Internal, "message")
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
			var got *pb.Pong

			stream, err := client.PingPingPong(
				ctx,
				grpc.Header(&header),
				grpc.Trailer(&trailer),
			)
			if err == nil {
				for _, request := range tt.requests {
					err = stream.Send(request)
					if err != nil {
						break
					}
				}

				if err == nil {
					got, err = stream.CloseAndRecv()
				}
			}


			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("error not equal, want: %s, got: %s", tt.wantErr, err)
			}
			if !reflect.DeepEqual(header, tt.wantHeader) {
				t.Errorf("header not equal, want: %s, got: %s", tt.wantHeader, header)
			}
			if !proto.Equal(got, tt.want) {
				t.Errorf(
					"response not equal, want: %s, got: %s",
					prototext.Format(tt.want), prototext.Format(got),
				)
			}
			if !reflect.DeepEqual(trailer, tt.wantTrailer) {
				t.Errorf("trailer not equal, want: %s, got: %s", tt.wantTrailer, trailer)
			}
		})
	}
}

func TestClientStreamingRPC(t *testing.T) {
	ctx := context.Background()
	testClientStreamingRPC(ctx, t)
}

func TestClientStreamingRPCProxyOnce(t *testing.T) {
	lis, err := net.Listen("tcp", "localhost:5569")
	if err != nil {
		t.Fatal(err)
	}
	s := &clientStreamingRPCServer{}
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
	testClientStreamingRPC(ctx, t)
	grpcServer.GracefulStop()
	testClientStreamingRPC(ctx, t)
}
