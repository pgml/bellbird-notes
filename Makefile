BIN_CLI=bbnotes

build-cli:
	go mod tidy
	go build -o ${BIN_CLI} cmd/tui/main.go

install-cli-local:
	rsync -azP ${BIN_CLI} ~/.local/bin/
	go clean
	rm ${BIN_CLI}

install-cli:
	rsync -azP ${BIN_CLI} /usr/bin/
	go clean
	rm ${BIN_CLI}

clean:
	go clean
	rm ${BIN_CLI}
