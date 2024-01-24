.PHONY: build
build:
	go build -race -o bin/mb-grpc cmd/mb-grpc/*.go

.PHONY: generate-test-proto
generate-test-proto:
	rm -f test/mb-grpc/pingpong/*.go
	protoc --proto_path=test/mb-grpc/pingpong --go_out=test/mb-grpc/pingpong --go_opt=paths=source_relative --go-grpc_out=test/mb-grpc/pingpong --go-grpc_opt=paths=source_relative test/mb-grpc/pingpong/pingpong.proto test/mb-grpc/pingpong/messages.proto

.PHONY: test
test: build
	go test -race -v ./...

.PHONY: clean
clean:
	rm -rf bin/
	rm -f test/mb-grpc/mb*.log
