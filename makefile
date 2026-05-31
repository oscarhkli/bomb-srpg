.DEFAULT_GOAL := test
.PHONY: fmt vet test coverage

fmt:
	go fmt ./...

vet: fmt
	go vet ./...

test:
	go test -cover -coverprofile=coverage.txt ./...

coverage: test
	go tool cover -html=coverage.txt

