.PHONY: build
build:
	cd examples/global && go build
	cd examples/hello-world && go build
	cd examples/nil && go build
