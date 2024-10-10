.PHONY: build
build:
	cd examples/global && go mod tidy && go build
	cd examples/hello-world && go mod tidy && go build
	cd examples/human-readable-logging && go mod tidy && go build
	cd examples/nil && go mod tidy && go build
	cd examples/zerolog-logging && go mod tidy && go build
