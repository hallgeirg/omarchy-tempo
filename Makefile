GO ?= go

.PHONY: build test vet race
build:
	cd src && $(GO) build -buildvcs=false -trimpath -ldflags="-s -w" -o ../bin/tempo-native .
test:
	cd src && $(GO) test -buildvcs=false ./...
vet:
	cd src && $(GO) vet ./...
race:
	cd src && $(GO) test -buildvcs=false -race ./...
