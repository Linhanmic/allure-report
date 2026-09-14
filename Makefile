.PHONY: test build install distro

test:
	go test ./...

build:
	go run build/make.go

install:
	go run build/make.go --install

distro:
	go run build/make.go --distro
