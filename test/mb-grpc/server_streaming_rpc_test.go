package mbgrpc

import (
	"context"
	"io"
	"net"
	"reflect"
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

type serverStreamingRPCServer struct {
	pb.UnimplementedServiceServer
}

func (s *serverStreamingRPCServer) PingPongPong(ping *pb.Ping, stream pb.Service_PingPongPongServer) error {
	header := metadata.Pairs("symbol", "-_.~!#$&'()*+,/:;=?@[]%20")
	stream.SetHeader(header)
	trailer := metadata.Pairs("symbol", "-_.~!#$&'()*+,/:;=?@[]%20")
	stream.SetTrailer(trailer)

	if ping.Ping == "success" {
		for i := 0; i < 2; i++ {
			err := stream.Send(&pb.Pong{Pong: "你好，世界"})
			if err != nil {
				return err
			}
		}
		return nil
	} else if ping.Ping == "failure" {
		st := status.New(codes.Internal, "message")
		st, err := st.WithDetails(&pb.Pong{Pong: "你好，世界"})
		if err != nil {
			return err
		}
		return st.Err()
	}

	return status.Errorf(codes.Unknown, "unexpected ping")
}

func testServerStreamingRPC(ctx context.Context, t *testing.T) {
	conn, err := grpc.Dial("localhost:5568", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.Close()
	})

	client := pb.NewServiceClient(conn)

	tests := []struct{
		name        string
		request     *pb.Ping
		want        []*pb.Pong
		wantHeader  metadata.MD
		wantTrailer metadata.MD
		wantErr     error
	} {
		{
			name: "Success",
			request: &pb.Ping{Ping: "success"},
			want: []*pb.Pong{
				{Pong: "你好，世界"},
				{Pong: "你好，世界"},
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
			name: "Failure",
			request: &pb.Ping{Ping: "failure"},
			want: []*pb.Pong{},
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
			var pong *pb.Pong
			var got []*pb.Pong

			stream, err := client.PingPongPong(
				ctx,
				tt.request,
				grpc.Header(&header),
				grpc.Trailer(&trailer),
			)
			if err == nil {
				for {
					pong, err = stream.Recv()
					if err != nil {
						break
					}

					got = append(got, pong)
				}
			}


			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("error not equal, want: %s, got: %s", tt.wantErr, err)
			}
			if !reflect.DeepEqual(header, tt.wantHeader) {
				t.Errorf("header not equal, want: %s, got: %s", tt.wantHeader, header)
			}
			if !isSliceEqual(got, tt.want) {
				t.Errorf(
					"response not equal, want: %s, got: %s",
					formatSlice(tt.want), formatSlice(got),
				)
			}
			if !reflect.DeepEqual(trailer, tt.wantTrailer) {
				t.Errorf("trailer not equal, want: %s, got: %s", tt.wantTrailer, trailer)
			}
		})
	}
}

func TestServerStreamingRPC(t *testing.T) {
	ctx := context.Background()
	testServerStreamingRPC(ctx, t)
}

func TestServerStreamingRPCProxyOnce(t *testing.T) {
	lis, err := net.Listen("tcp", "localhost:5569")
	if err != nil {
		t.Fatal(err)
	}
	s := &serverStreamingRPCServer{}
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
	testServerStreamingRPC(ctx, t)
	grpcServer.GracefulStop()
	time.Sleep(2 * time.Second)
	testServerStreamingRPC(ctx, t)
}
