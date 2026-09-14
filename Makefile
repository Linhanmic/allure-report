.PHONY: test build install distro distro-all example

test:
	go test ./...

build:
	CGO_ENABLED=0 go run build/make.go

install:
	CGO_ENABLED=0 go run build/make.go --install

distro:
	CGO_ENABLED=0 go run build/make.go --distro

distro-all:
	CGO_ENABLED=0 go run build/make.go --all-platforms
	CGO_ENABLED=0 go run build/make.go --distro --all-platforms

example:
	cd examples/gauge-js && gauge run specs
