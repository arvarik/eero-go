.PHONY: tidy test lint build-local build-linux-amd64 build-linux-arm64 clean setup

tidy:
	go mod tidy
	go fmt ./...

test:
	go test -v -race ./...

lint:
	@echo "=> Running golangci-lint..."
	# Assuming golangci-lint is installed locally or in PATH
	golangci-lint run ./...

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

setup:
	@echo "=> Configuring local git hooks..."
	git config core.hooksPath .githooks
	chmod +x .githooks/*
	@echo "✅ Pre-commit hooks installed."
