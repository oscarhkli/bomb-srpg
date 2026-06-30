.DEFAULT_GOAL := test
.PHONY: fmt vet test test-race coverage build run web-dev web-test web-lint web-build web-coverage

fmt:
	go fmt ./...

vet: fmt
	go vet ./...

test: vet
	go test -cover -coverprofile=coverage.txt ./cli/... ./engine/... ./server/...

test-race:
	go test -race ./cli/... ./engine/... ./server/...

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

web-dev:
	cd web && npm run dev

web-test:
	cd web && npm run test:run

web-lint:
	cd web && npm run lint && npm run typecheck

web-build:
	cd web && npm run build

web-coverage:
	cd web && npm run test:run -- --coverage