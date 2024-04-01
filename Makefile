build:
	go build -o bin/main .

run: build
	./bin/main

test:
	go test .

proto:
	protoc --go-grpc_out=grpc --go_out=grpc --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative .proto/user.proto