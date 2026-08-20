.PHONY: build test vet lint

build:
	go build -o socialsight ./cmd/socialsight

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run
