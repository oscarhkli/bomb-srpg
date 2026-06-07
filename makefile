.DEFAULT_GOAL := test
.PHONY: fmt vet test coverage build run

fmt:
	go fmt ./...

vet: fmt
	go vet ./...

test: vet
	go test -cover -coverprofile=coverage.txt ./engine/... ./cmd/cli/... 

coverage: test
	go tool cover -html=coverage.txt

build: vet
	go build -o bin/srpg-cli cmd/srpg-cli.go

run: build
	./bin/srpg-cli
	