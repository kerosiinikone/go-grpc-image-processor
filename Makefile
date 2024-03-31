build:
	go build -o bin/main .

run: build
	./bin/main

test:
	go test .

docker-run:
	docker-compose up -d
 