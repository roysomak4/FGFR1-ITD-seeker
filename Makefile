VERSION := $(shell cat version.txt)
BINARY_NAME := fgfr1-itd-seeker
DOCKER_IMAGE := ghcr.io/roysomak4/fgfr1-itd-seeker
BASE_IMAGE_TAG := 21-jre-alpine-3.22
IMAGE_TAG := v$(VERSION)-$(BASE_IMAGE_TAG)

# Development build (with debug info) for current platform
.PHONY: build
build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY_NAME) main.go

# Build for all platforms (development)
.PHONY: build-all
build-all:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY_NAME)-$(VERSION)-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY_NAME)-$(VERSION)-darwin-arm64 main.go
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY_NAME)-$(VERSION)-linux-amd64 main.go

# Production build (optimized, smaller) for current platform
.PHONY: build-release
build-release:
	mkdir -p release
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o release/$(BINARY_NAME) main.go

# Build release binaries for all platforms
.PHONY: build-release-all
build-release-all:
	mkdir -p release
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o release/$(BINARY_NAME)-$(VERSION)-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o release/$(BINARY_NAME)-$(VERSION)-darwin-arm64 main.go
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o release/$(BINARY_NAME)-$(VERSION)-linux-amd64 main.go

# Build multi-arch image locally (loads into local Docker daemon, single platform only)
# Use this to test the image on your current machine before pushing.
.PHONY: docker-build
docker-build:
	docker buildx build --platform linux/$(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') \
		--load \
		--build-arg BASE_IMAGE_TAG=$(BASE_IMAGE_TAG) \
		-t $(DOCKER_IMAGE):$(IMAGE_TAG) \
		-t $(DOCKER_IMAGE):latest \
		.

# Build multi-arch image (linux/amd64 + linux/arm64) and push to registry
# Requires: docker buildx create --use (once per machine)
.PHONY: docker-push
docker-push:
	docker buildx build --platform linux/amd64,linux/arm64 \
		--push \
		--build-arg BASE_IMAGE_TAG=$(BASE_IMAGE_TAG) \
		-t $(DOCKER_IMAGE):$(IMAGE_TAG) \
		-t $(DOCKER_IMAGE):latest \
		.

# Run docker container
.PHONY: docker-run
docker-run:
	docker run --rm -v $(PWD)/testdata:/data $(DOCKER_IMAGE):$(IMAGE_TAG)

.PHONY: install
install:
	go install -ldflags "-s -w -X main.version=$(VERSION)" main.go

.PHONY: run
run:
	go run -ldflags "-X main.version=$(VERSION)" main.go

.PHONY: clean
clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	rm -rf release/
