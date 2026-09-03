.PHONY: run test lint build cover exporter docker-build compose clean

run:
	go run ./cmd/miniprom -config config.json

exporter:
	go run ./examples/exporter

test:
	go test -race ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	golangci-lint run

build:
	go build -o bin/miniprom ./cmd/miniprom
	go build -o bin/exporter ./examples/exporter

docker-build:
	docker build -t miniprom --target miniprom .

compose:
	docker compose up --build

clean:
	rm -rf bin coverage.out
