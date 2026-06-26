.PHONY: all build dev test clean

APP_NAME=packetmind
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X github.com/packetmind/packetmind/internal/version.Version=$(VERSION) -X github.com/packetmind/packetmind/internal/version.BuildTime=$(BUILD_TIME) -X github.com/packetmind/packetmind/internal/version.Commit=$(COMMIT)"

all: build

build:
	wails build $(LDFLAGS)

dev:
	wails dev

build-gui:
	cd gui && npm run build

dev-gui:
	cd gui && npm run dev

test:
	go test -v -race ./...

clean:
	if exist bin rmdir /s /q bin
	if exist data rmdir /s /q data

lint:
	golangci-lint run

fmt:
	go fmt ./...
