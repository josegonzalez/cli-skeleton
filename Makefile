.PHONY: build
build:
	cd examples/global && go build
	cd examples/hello-world && go build
	cd examples/human-readable-logging && go build
	cd examples/nil && go build
	cd examples/zerolog-logging && go build
