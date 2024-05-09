run-worker: export GRPC_VERBOSITY=debug

build-worker-resize:
	@go build -o bin/workers/resize ./workers/resize

run-worker-resize: build-worker-resize
	@./bin/workers/resize

build-worker-invert:
	@go build -o bin/workers/invert ./workers/invert

run-worker-invert: build-worker-invert
	@./bin/workers/invert

build-client:
	@go build -o bin/client ./client

run-client: build-client
	@./bin/client


test:
	go test .

proto:
	protoc --go-grpc_out=grpc --go_out=grpc --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative service.proto