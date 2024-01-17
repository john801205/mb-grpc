package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/john801205/mb-grpc/internal/mountebank"
	"github.com/john801205/mb-grpc/internal/proto"
	"github.com/john801205/mb-grpc/internal/service"
)

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

	host := ""
	if config.Host != nil {
		host = *config.Host
	}
	port := config.Port

	callbackURL := strings.Replace(config.CallbackURLTemplate, ":port", strconv.Itoa(port), 1)
	mbClient := mountebank.NewClient(callbackURL)

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

	registry, err := proto.Load(importDirs, protoFiles)
	if err != nil {
		log.Fatal(err)
	}

	genericService := service.New(registry, mbClient)
	grpcServer := grpc.NewServer()

	registry.RegisterGenericService(grpcServer, genericService)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()

	fmt.Println("grpc")
	grpcServer.Serve(lis)
}
