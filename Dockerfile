# Multi-stage Dockerfile for FGFR1-ITD-seeker
# Image: ghcr.io/cchmc-research-mgps/fgfr1-itd-seeker:<tag>

# Stage 1: Build the Go application
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy all necessary files
COPY . .

# Build using Makefile
RUN apk add --no-cache make && \
    make build-release-all

# Stage 2: Create the runtime image with VarDict
FROM eclipse-temurin:17-jre-alpine

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
        # clean up install mess
        && apk del build_deps \
        && rm -rf /var/cache/apk/*

# Copy the compiled binary from builder stage
COPY --from=builder /build/release/fgfr1-itd-seeker* /usr/local/bin/
COPY --from=builder /build/bedfiles /usr/local/bin/
