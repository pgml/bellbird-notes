BIN_CLI = bbnotes
DATE = $(shell date +%Y%m%d%H)
GIT_HASH = g$(shell git rev-parse --short HEAD)
PKG = bellbird-notes/app
GOFLAGS = -trimpath
VERSION := $(shell git describe --tags --abbrev=0 || echo "dev")

build-dev:
	go mod tidy
	go build $(GOFLAGS) -ldflags="\
		-X '$(PKG).dev=true' \
		-X '$(PKG).version=$(VERSION)' \
		-X '$(PKG).commit=$(GIT_HASH)'" \
		-o ${BIN_CLI} cmd/tui/main.go

build-release:
	go mod tidy
	go build $(GOFLAGS) -ldflags="\
		-s -w -X '$(PKG).version=$(VERSION)' \
		-X '$(PKG).commit=$(GIT_HASH)'" \
		-o ${BIN_CLI} cmd/tui/main.go

install-local:
	rsync -azP ${BIN_CLI} ~/.local/bin/
	go clean
	rm ${BIN_CLI}

install:
	rsync -azP ${BIN_CLI} /usr/bin/
	go clean
	rm ${BIN_CLI}

clean:
	go clean
	rm ${BIN_CLI}

.PHONY: test

test:
	go test ./...

test-verbose:
	go test ./... -v
