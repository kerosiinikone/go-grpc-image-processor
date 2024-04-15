run-worker: export GRPC_VERBOSITY=debug

build-worker:
	@go build -o bin/workers ./workers

run-worker: build-worker
	@./bin/workers

build-client:
	@go build -o bin/client ./client

run-client: build-client
	@./bin/client


test:
	go test .

proto:
	protoc --go-grpc_out=grpc --go_out=grpc --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative service.proto