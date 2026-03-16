BINARY    = dts

# Git version info (Windows cmd.exe compatible)
ifeq ($(OS),Windows_NT)
    VERSION  ?= $(shell git describe --tags --always --dirty 2>nul || echo dev)
    COMMIT   ?= $(shell git rev-parse --short HEAD 2>nul || echo none)
else
    VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
    COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
endif

LDFLAGS   = -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)"

# Detect OS for platform-specific commands
ifeq ($(OS),Windows_NT)
    EXE      = .exe
    RM       = del /f /q
else
    EXE      =
    RM       = rm -f
endif

.PHONY: build clean test vet fmt install build-linux build-darwin build-windows build-all integration-test

build:
	go build $(LDFLAGS) -o bin/$(BINARY)$(EXE) .

install:
	go install $(LDFLAGS) .

test:
	go test -v ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w .

integration-test:
	go test -tags integration -v -count=1 ./test/integration/...

clean:
ifeq ($(OS),Windows_NT)
	-if exist bin rd /s /q bin
else
	rm -rf bin
endif

# Cross-compilation targets (use GOOS/GOARCH as make vars for portability)
build-linux:
	$(MAKE) GOOS=linux GOARCH=amd64 _cross OUT=bin/$(BINARY)-linux-amd64

build-darwin:
	$(MAKE) GOOS=darwin GOARCH=arm64 _cross OUT=bin/$(BINARY)-darwin-arm64

build-windows:
	$(MAKE) GOOS=windows GOARCH=amd64 _cross OUT=bin/$(BINARY)-windows-amd64.exe

_cross:
	go build $(LDFLAGS) -o $(OUT) .

build-all: build-linux build-darwin build-windows
