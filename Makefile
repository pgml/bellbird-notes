BIN_CLI=bbnotes

build-tui:
	go mod tidy
	go build -ldflags="-s -w" -o ${BIN_CLI} cmd/tui/main.go

install-tui-local:
	rsync -azP ${BIN_CLI} ~/.local/bin/
	go clean
	rm ${BIN_CLI}

install-tui:
	rsync -azP ${BIN_CLI} /usr/bin/
	go clean
	rm ${BIN_CLI}

clean:
	go clean
	rm ${BIN_CLI}
