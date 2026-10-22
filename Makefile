.DEFAULT_GOAL := test
.PHONY: fmt vet test test-race lint lint-all pre-push install-hooks coverage build build-server run run-server web-dev web-format web-typecheck web-test web-lint web-build web-e2e web-coverage gen-spec-index

fmt:
	go fmt ./...

vet: fmt
	go vet ./...

test: vet
	go test -cover -coverprofile=coverage.txt ./cli/... ./cmd/srpg-web/... ./cpu/... ./engine/... ./server/...

test-race: vet
	go test -race ./cli/... ./cmd/srpg-web/... ./cpu/... ./engine/... ./server/...

lint:
	golangci-lint run ./cli/... ./cmd/... ./cpu/... ./engine/... ./server/...

lint-all:
	golangci-lint run --new-from-rev= ./cli/... ./cmd/... ./cpu/... ./engine/... ./server/...

pre-push: lint test web-lint web-test

install-hooks:
	printf '#!/bin/sh\nexec make pre-push\n' > $$(git rev-parse --git-path hooks)/pre-push
	chmod +x $$(git rev-parse --git-path hooks)/pre-push

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

web-format:
	cd web && npm run format

web-typecheck:
	cd web && npm run typecheck

web-test: web-format web-typecheck
	cd web && npm run test:run

web-lint:
	cd web && npm run format:check && npm run lint

web-build: web-format web-typecheck
	cd web && npm run build

web-e2e: build-server
	cd web && npm run build:e2e && E2E_PORT=$$(node -e "const s=require('net').createServer().listen(0,()=>{console.log(s.address().port);s.close()})") npm run e2e -- $(if $(E2E_BROWSER),--project $(E2E_BROWSER))

web-coverage: web-format web-typecheck
	cd web && npm run test:run -- --coverage

gen-spec-index:
	go run scripts/gen-spec-index.go