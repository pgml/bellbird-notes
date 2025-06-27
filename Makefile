BIN_CLI = bbnotes
DATE = $(shell date +%Y%m%d%H)
GIT_HASH = $(shell git rev-parse --short HEAD)
PKG = bellbird-notes/app

build-dev:
	go mod tidy
	go build -ldflags "-X '$(PKG).DevVersion=g$(GIT_HASH)'" -o ${BIN_CLI} cmd/tui/main.go

build-release:
	go mod tidy
	go build -ldflags="-s -w" -o ${BIN_CLI} cmd/tui/main.go

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
