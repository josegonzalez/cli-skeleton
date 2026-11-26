VERSION ?= 0.0.0-test
LDFLAGS := -X main.Version=$(VERSION)

.PHONY: build
build:
	cd cmd/cli-skeleton && go build -ldflags "$(LDFLAGS)"
	cd examples/global && go mod tidy && go build -ldflags "$(LDFLAGS)"
	cd examples/hello-world && go mod tidy && go build -ldflags "$(LDFLAGS)"
	cd examples/human-readable-logging && go mod tidy && go build -ldflags "$(LDFLAGS)"
	cd examples/nested && go mod tidy && go build -ldflags "$(LDFLAGS)"
	cd examples/nil && go mod tidy && go build -ldflags "$(LDFLAGS)"
	cd examples/zerolog-logging && go mod tidy && go build -ldflags "$(LDFLAGS)"

.PHONY: test
test:
	go test -race ./...
	cd cmd/cli-skeleton && go build -ldflags "$(LDFLAGS)"
	cd examples/global && go test -race ./...
	cd examples/hello-world && go test -race ./...
	cd examples/human-readable-logging && go test -race ./...
	cd examples/nested && go test -race ./...
	cd examples/nil && go test -race ./...
	cd examples/zerolog-logging && go test -race ./...

.PHONY: test-integration
test-integration: build
	bats test
