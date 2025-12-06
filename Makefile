VERSION := $(shell cat version.txt)
BINARY_NAME := fgfr1-itd-seeker

# Development build (with debug info)
.PHONY: build
build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY_NAME) main.go

# Production build (optimized, smaller)
.PHONY: build-release
build-release:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY_NAME) main.go

.PHONY: install
install:
	go install -ldflags "-s -w -X main.version=$(VERSION)" main.go

.PHONY: run
run:
	go run -ldflags "-s -w -X main.version=$(VERSION)" main.go
