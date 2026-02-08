VERSION := $(shell cat version.txt)
BINARY_NAME := fgfr1-itd-seeker
DOCKER_IMAGE := ghcr.io/cchmc-research-mgps/fgfr1-itd-seeker

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

# Docker build
.PHONY: docker-build
docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

# Docker build and push
.PHONY: docker-push
docker-push: docker-build
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest

# Run docker container
.PHONY: docker-run
docker-run:
	docker run --rm -v $(PWD)/testdata:/data $(DOCKER_IMAGE):$(VERSION)

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
