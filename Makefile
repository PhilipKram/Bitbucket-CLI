BINARY    := bb
MODULE    := github.com/PhilipKram/bitbucket-cli
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE      := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS   := -s -w \
	-X $(MODULE)/cmd.version=$(VERSION) \
	-X $(MODULE)/cmd.commit=$(COMMIT) \
	-X $(MODULE)/cmd.date=$(DATE)

.PHONY: build install clean test lint release-dry

build:
	go build -ldflags '$(LDFLAGS)' -o $(BINARY) .

install:
	go install -ldflags '$(LDFLAGS)' .

clean:
	rm -f $(BINARY)
	rm -rf dist/

test:
	go test ./...

lint:
	golangci-lint run

release-dry:
	goreleaser release --snapshot --clean
