.DEFAULT_GOAL := check
GO_MODULE := $(shell go list -m)
GO_FILES := $(shell find . -type f -name '*.go')

check: generate fmt vet lint test test-race cover-html

generate:
	go generate ./...

fmt:
	go fmt ./...
	gofumpt -l -w $(GO_FILES)
	gci write -s standard -s default -s "prefix($(GO_MODULE))" .

lint:
	golangci-lint run -v --fix --timeout=5m ./...

vet:
	go vet ./...

test:
	go test ./...

test-race:
	go test -race -count=5 ./...

bench-all:
	go test -bench=. -benchmem ./...

cover-html:
	@go test -coverprofile=./coverage.text -covermode=atomic $(shell go list ./...)
	@go tool cover -html=./coverage.text -o ./cover.html && rm ./coverage.text

PORT ?= 8383
run-example-base:
	go run examples/basic-app/main.go -port $(PORT)
