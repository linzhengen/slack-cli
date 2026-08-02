BINARY := slack-cli
VERSION ?= dev

.PHONY: build test e2e vet lint fmt-check install

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/$(BINARY) ./cmd/slack-cli

install:
	go install -ldflags "-X main.version=$(VERSION)" ./cmd/slack-cli

test:
	go test ./...

# Black-box tests that build the real binary and exec it as a subprocess
# against a fake Slack API server (internal/slacktest). Opt-in via a build
# tag, separate from `make test`, since it pays for a `go build` up front.
e2e:
	go test -tags=e2e ./e2e/...

vet:
	go vet ./...

lint:
	golangci-lint run ./...
	golangci-lint run --build-tags=e2e ./...

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needs to be run on:"; gofmt -l .; exit 1)
