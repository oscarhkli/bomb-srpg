.DEFAULT_GOAL := test
.PHONY: fmt vet test test-race coverage build run web-dev web-test web-lint web-format web-typecheck web-build web-coverage gen-spec-index

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

web-coverage: web-format web-typecheck
	cd web && npm run test:run -- --coverage

gen-spec-index:
	go run scripts/gen-spec-index.go