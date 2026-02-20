.PHONY: tidy test build-local build-linux-amd64 build-linux-arm64 clean

tidy:
	go mod tidy
	go fmt ./...

test:
	go test -v -race ./...

build-local:
	mkdir -p bin
	go build -o bin/eero-go ./cmd/example

build-linux-amd64:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o bin/eero-go-linux-amd64 ./cmd/example

build-linux-arm64:
	mkdir -p bin
	GOOS=linux GOARCH=arm64 go build -o bin/eero-go-linux-arm64 ./cmd/example

clean:
	rm -rf bin/
	rm -f .eero_session.json
