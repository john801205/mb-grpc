package proto

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/john801205/mb-grpc/internal/log"
)

type Registry struct {
	files *protoregistry.Files
}

func loadFile(files map[string]*descriptorpb.FileDescriptorProto, fileName string) error {
	_, err := protoregistry.GlobalFiles.FindFileByPath(fileName)
	if err == nil {
		return nil
	} else if err != protoregistry.NotFound {
		return err
	}

	file, ok := files[fileName]
	if !ok {
		return fmt.Errorf("no descriptor found for %s", fileName)
	}

	for _, dep := range file.GetDependency() {
		err := loadFile(files, dep)
		if err != nil {
			return err
		}
	}

	fd, err := protodesc.NewFile(file, protoregistry.GlobalFiles)
	if err != nil {
		return err
	}

	err = protoregistry.GlobalFiles.RegisterFile(fd)
	if err != nil {
		return err
	}

	for i := 0; i < fd.Enums().Len(); i++ {
		enumDesc := fd.Enums().Get(i)
		enum := dynamicpb.NewEnumType(enumDesc)
		err := protoregistry.GlobalTypes.RegisterEnum(enum)
		if err != nil {
			return err
		}
	}

	for i := 0; i < fd.Messages().Len(); i++ {
		msgDesc := fd.Messages().Get(i)
		msg := dynamicpb.NewMessageType(msgDesc)
		err := protoregistry.GlobalTypes.RegisterMessage(msg)
		if err != nil {
			return err
		}
	}

	for i := 0; i < fd.Extensions().Len(); i++ {
		extDesc := fd.Extensions().Get(i)
		ext := dynamicpb.NewExtensionType(extDesc)
		err := protoregistry.GlobalTypes.RegisterExtension(ext)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadFiles(files *descriptorpb.FileDescriptorSet) error {
	descriptors := make(map[string]*descriptorpb.FileDescriptorProto)
	for _, file := range files.GetFile() {
		descriptors[file.GetName()] = file
	}

	for _, file := range files.GetFile() {
		err := loadFile(descriptors, file.GetName())
		if err != nil {
			return err
		}
	}

	return nil
}

func Load(importDirs, protoFiles []string) (*Registry, error) {
	tmpDir, err := os.MkdirTemp("", "mb-grpc")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	tmpFilePath := filepath.Join(tmpDir, "proto.desc")

	var protocArgs []string
	protocArgs = append(protocArgs, "--include_imports")
	protocArgs = append(protocArgs, "--include_source_info")
	protocArgs = append(protocArgs, "--descriptor_set_out="+tmpFilePath)
	for _, importDir := range importDirs {
		protocArgs = append(protocArgs, "--proto_path="+importDir)
	}
	protocArgs = append(protocArgs, protoFiles...)

	cmd := exec.Command("protoc", protocArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Errorf("protoc output: %s", output)
		return nil, err
	}

	data, err := os.ReadFile(tmpFilePath)
	if err != nil {
		return nil, err
	}

	desc := new(descriptorpb.FileDescriptorSet)
	err = proto.Unmarshal(data, desc)
	if err != nil {
		return nil, err
	}

	err = loadFiles(desc)
	if err != nil {
		return nil, err
	}

	files, err := protodesc.NewFiles(desc)
	if err != nil {
		return nil, err
	}

	return &Registry{files: files}, nil
}

func (r *Registry) FindMethodDescriptorByName(name string) (protoreflect.MethodDescriptor, error) {
	slices := strings.SplitN(name, "/", 3)
	if len(slices) != 3 {
		return nil, fmt.Errorf("unknown method format: %s", name)
	}

	service := protoreflect.FullName(slices[1])
	method := protoreflect.Name(slices[2])
	desc, err := r.files.FindDescriptorByName(service)
	if err != nil {
		return nil, err
	}

	serviceDesc, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("invalid service descriptor type")
	}

	methodDesc := serviceDesc.Methods().ByName(method)
	if methodDesc == nil {
		return nil, fmt.Errorf("method not found: %s", method)
	}

	return methodDesc, nil
}

type GenericService interface {
	HandleUnaryCall(any, context.Context, func(any) error, grpc.UnaryServerInterceptor) (any, error)
	HandleStreamCall(any, grpc.ServerStream) error
}

func (r *Registry) RegisterGenericService(
	server *grpc.Server,
	genericService GenericService,
) {
	r.files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for i := 0; i < fd.Services().Len(); i++ {
			service := fd.Services().Get(i)

			serviceDesc := grpc.ServiceDesc{
				ServiceName: string(service.FullName()),
				HandlerType: (*any)(nil),
				Metadata:    fd.Path(),
			}

			for j := 0; j < service.Methods().Len(); j++ {
				method := service.Methods().Get(j)

				if method.IsStreamingClient() || method.IsStreamingServer() {
					streamDesc := grpc.StreamDesc{
						StreamName:    string(method.Name()),
						Handler:       genericService.HandleStreamCall,
						ServerStreams: method.IsStreamingServer(),
						ClientStreams: method.IsStreamingClient(),
					}
					serviceDesc.Streams = append(serviceDesc.Streams, streamDesc)
				} else {
					methodDesc := grpc.MethodDesc{
						MethodName: string(method.Name()),
						Handler:    genericService.HandleUnaryCall,
					}
					serviceDesc.Methods = append(serviceDesc.Methods, methodDesc)
				}
			}

			server.RegisterService(&serviceDesc, genericService)
		}

		return true
	})
}
