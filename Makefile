.DEFAULT_GOAL := test
.PHONY: fmt vet test coverage build run

fmt:
	go fmt ./...

vet: fmt
	go vet ./...

test: vet
	go test -cover -coverprofile=coverage.txt ./cli/... ./engine/...

coverage: test
	go tool cover -html=coverage.txt

build: vet
	go build -o bin/srpg-cli ./cmd/srpg-cli

build-server: vet
	go build -o bin/srpg-web ./cmd/srpg-web

run: build
	./bin/srpg-cli

run-server: build-server
	./bin/srpg-web
