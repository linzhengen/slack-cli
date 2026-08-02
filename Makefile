BINARY := slack-cli
VERSION ?= dev

.PHONY: build test vet lint fmt-check install

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/$(BINARY) ./cmd/slack-cli

install:
	go install -ldflags "-X main.version=$(VERSION)" ./cmd/slack-cli

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needs to be run on:"; gofmt -l .; exit 1)
