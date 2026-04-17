# Multi-stage Dockerfile for FGFR1-ITD-seeker
# Image: ghcr.io/cchmc-research-mgps/fgfr1-itd-seeker:<tag>

# Global ARGs (must be declared before any FROM to be usable in FROM instructions)
ARG BASE_IMAGE_TAG=21-jre-alpine-3.22

# Stage 1: Build the Go application
# Use BUILDPLATFORM so Go compiles natively on the build machine (fast),
# then cross-compiles the binary for the target platform.
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

# Docker injects these automatically during multi-arch builds
ARG TARGETOS
ARG TARGETARCH

WORKDIR /build

# Copy all necessary files
COPY . .

# Cross-compile for the target platform via GOOS/GOARCH env vars
RUN apk add --no-cache make && \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} make build-release

# Stage 2: Create the runtime image with VarDict
FROM eclipse-temurin:${BASE_IMAGE_TAG}

LABEL maintainer="Somak Roy<roysomak4@gmail.com>" \
    function="Docker image with FGFR1-ITD-seeker" \
    org.opencontainers.image.source="https://github.com/roysomak4/FGFR1-ITD-seeker"

# app versions
ENV VARDICT_VER=1.8.3

# install vardict
RUN apk update && apk add --no-cache --virtual build_deps \
        wget \
        # install runtime dependencies
        && apk add --no-cache R perl pcre xz-libs libbz2 \
        # download vardict
        && wget "https://github.com/AstraZeneca-NGS/VarDictJava/releases/download/v${VARDICT_VER}/VarDict-${VARDICT_VER}.tar" \
        && tar -xvf VarDict-${VARDICT_VER}.tar \
        && mv VarDict-${VARDICT_VER} vardict_app \
        && rm VarDict-${VARDICT_VER}.tar \
        && mv vardict_app /usr/local/ \
        && ln -s /usr/local/vardict_app/bin/VarDict /usr/local/bin/vardict \
        && ln -s /usr/local/vardict_app/bin/var2vcf_valid.pl /usr/local/bin/var2vcf_valid.pl \
        && ln -s /usr/local/vardict_app/bin/var2vcf_paired.pl /usr/local/bin/var2vcf_paired.pl \
        && ln -s /usr/local/vardict_app/bin/teststrandbias.R /usr/local/bin/teststrandbias.R \
        && ln -s /usr/local/vardict_app/bin/testsomatic.R /usr/local/bin/testsomatic.R \
        # clean up install mess
        && apk del build_deps \
        && rm -rf /var/cache/apk/*

# Copy the compiled binary from builder stage
COPY --from=builder /build/release/fgfr1-itd-seeker* /usr/local/bin/
