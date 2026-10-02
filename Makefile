.DEFAULT_GOAL := check
.PHONY: check tidy generate fmt vet lint test test-race protocol-replay http-consumer cover-html bench-all
GO_MODULE := $(shell go list -m)
GO_FILES := $(shell find . -type f -name '*.go')

check: tidy generate fmt vet lint test test-race protocol-replay http-consumer cover-html

tidy:
	go mod tidy

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

protocol-replay:
	GOINERTIA_CLIENT_REPLAY=1 go test -race -count=5 -run '^TestPinnedClientProtocolReplay$$' ./

http-consumer:
	cd integration/nethttp-consumer && go vet ./... && go build ./... && go test -race -count=5 ./...
	@set -eu; cd integration/nethttp-consumer; task_deps=$$(mktemp); trap 'rm -f "$$task_deps"' EXIT; \
	go list -deps -test ./... > "$$task_deps"; \
	if grep -E '(^github.com/gofiber/|^github.com/valyala/fasthttp)' "$$task_deps"; then exit 1; fi

bench-all:
	go test -bench=. -benchmem ./...

cover-html:
	@go test -coverprofile=./coverage.text -covermode=atomic $(shell go list ./...)
	@go tool cover -html=./coverage.text -o ./cover.html && rm ./coverage.text

PORT ?= 8383
run-example-base:
	go run examples/basic-app/main.go -port $(PORT)
