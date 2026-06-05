.DEFAULT_GOAL := test
.PHONY: fmt vet test coverage

fmt:
	go fmt ./...

vet: fmt
	go vet ./...

test: vet
	go test -cover ./engine/... ./cmd/cli/... -coverprofile=coverage.out

coverage: test
	go tool cover -html=coverage.txt

