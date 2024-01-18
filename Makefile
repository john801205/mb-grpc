.PHONY: build
build:
	go build -race -o bin/mb-grpc cmd/mb-grpc/*.go

generate-test-proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative test/mb-grpc/pingpong/pingpong.proto

.PHONY: test
test: build
	go test -race -v ./...
