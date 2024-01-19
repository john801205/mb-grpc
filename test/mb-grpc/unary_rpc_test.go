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

type unaryRPCServer struct {
	pb.UnimplementedServiceServer
}

func (s *unaryRPCServer) PingPong(ctx context.Context, req *pb.Ping) (*pb.Pong, error) {
	header := metadata.Pairs("aaaa", "bbb", "mb-grpc-data-bin", "いちばん")
	grpc.SetHeader(ctx, header)
	trailer := metadata.Pairs("bbbb", "aaa", "mb-grpc-data-bin", "いちばん")
	grpc.SetTrailer(ctx, trailer)

	if req.Ping == "success" {
		return &pb.Pong{Pong: "Hello, world"}, nil
	} else if req.Ping == "failure" {
		st := status.New(codes.Aborted, "message")
		st, err := st.WithDetails(&pb.Pong{Pong: "Hello, world"})
		if err != nil {
			return nil, err
		}
		return nil, st.Err()
	}

	return nil, status.Errorf(codes.Unknown, "unexpected ping")
}

func testUnaryRPC(ctx context.Context, t *testing.T) {
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

	tests := []struct {
		name        string
		request     *pb.Ping
		want        *pb.Pong
		wantHeader  metadata.MD
		wantTrailer metadata.MD
		wantErr     error
	}{
		{
			name:    "Success",
			request: &pb.Ping{Ping: "success"},
			want:    &pb.Pong{Pong: "Hello, world"},
			wantHeader: metadata.New(map[string]string{
				"aaaa":             "bbb",
				"mb-grpc-data-bin": "いちばん",
				"content-type":     "application/grpc",
			}),
			wantTrailer: metadata.New(map[string]string{
				"bbbb":             "aaa",
				"mb-grpc-data-bin": "いちばん",
			}),
			wantErr: nil,
		},
		{
			name:    "Failure",
			request: &pb.Ping{Ping: "failure"},
			wantHeader: metadata.New(map[string]string{
				"aaaa":             "bbb",
				"mb-grpc-data-bin": "いちばん",
				"content-type":     "application/grpc",
			}),
			wantTrailer: func() metadata.MD {
				md := metadata.New(map[string]string{
					"bbbb":             "aaa",
					"mb-grpc-data-bin": "いちばん",
				})
				st := status.New(codes.Aborted, "message")
				st, err := st.WithDetails(&pb.Pong{Pong: "Hello, world"})
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
				st, err := st.WithDetails(&pb.Pong{Pong: "Hello, world"})
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
			got, err := client.PingPong(
				ctx,
				tt.request,
				grpc.Header(&header),
				grpc.Trailer(&trailer),
			)

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

func TestUnaryRPC(t *testing.T) {
	ctx := context.Background()
	testUnaryRPC(ctx, t)
}

func TestUnaryRPCProxyOnce(t *testing.T) {
	lis, err := net.Listen("tcp", "localhost:5569")
	if err != nil {
		t.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterServiceServer(grpcServer, &unaryRPCServer{})
	go func() {
		err := grpcServer.Serve(lis)
		if err != nil {
			t.Error(err)
		}
	}()

	time.Sleep(2 * time.Second)

	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, "proxy-mode", "proxyOnce")
	testUnaryRPC(ctx, t)
	grpcServer.GracefulStop()
	testUnaryRPC(ctx, t)
}
