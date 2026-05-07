# Dockerfile References: https://docs.docker.com/engine/reference/builder/
# This dockerfile uses a multi-stage build system to reduce the image footprint.

######
# Build frontend
######
FROM --platform=${BUILDPLATFORM} node:lts-alpine AS frontend
# Set the working directory
WORKDIR /build
# Download dependencies
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
# Set dist output directory
ENV DIST_OUT_DIR="dist"
# Copy the sources to the working directory
COPY frontend .
# Build the frontend
RUN npm run build

######
# Build backend
######
FROM --platform=${BUILDPLATFORM} golang:1.26-alpine AS builder
# Set the working directory
WORKDIR /build
# Download dependencies
COPY go.mod go.sum ./
RUN go mod download
# Copy the sources to the working directory
COPY ./cmd ./cmd
COPY ./internal ./internal
# Copy the frontend build result
COPY --from=frontend /build/dist/ ./internal/app/api/core/frontend-dist/
# Set the build version from arguments
ARG BUILD_VERSION
# Split to cross-platform build
ARG TARGETARCH
# Build the application
RUN CGO_ENABLED=0 GOARCH=${TARGETARCH} go build -o /build/dist/wg-portal \
  -ldflags "-w -s -extldflags '-static' -X 'github.com/h44z/wg-portal/internal.Version=${BUILD_VERSION}'" \
  -tags netgo \
  cmd/wg-portal/main.go

######
# Export binaries
######
FROM scratch AS binaries
COPY --from=builder /build/dist/wg-portal /

######
# Final image
######
######
# Build amneziawg-tools (awg + awg-quick) from upstream
# We need these in the final image because AmneziaController shells
# out to them. amneziawg-tools is forked from wireguard-tools — no
# Alpine package exists, so we build from source. Pin AWG_TOOLS_REF
# to a tagged release for reproducibility.
######
FROM alpine:3.23 AS amneziawg-tools-builder
ARG AWG_TOOLS_REF=v1.0.20260223
RUN apk add --no-cache git make gcc musl-dev linux-headers bash
WORKDIR /build
RUN git clone --depth 1 --branch ${AWG_TOOLS_REF} \
    https://github.com/amnezia-vpn/amneziawg-tools.git . \
    && cd src \
    && make WITH_BASHCOMPLETION=no WITH_WGQUICK=yes WITH_SYSTEMDUNITS=no \
    && strip --strip-all wg awg awg-quick 2>/dev/null || true

######
# Final image
######
FROM alpine:3.23
# Install OS-level dependencies. wireguard-tools provides 'wg' for
# vanilla WG interfaces; awg/awg-quick come from the builder stage
# above for AmneziaWG support (BNet-a2rn).
RUN apk add --no-cache bash curl iptables nftables openresolv wireguard-tools tzdata
# Setup timezone
ENV TZ=UTC
# Copy binaries
COPY --from=builder /build/dist/wg-portal /app/wg-portal
# Copy amneziawg userspace tools into PATH so AmneziaController can
# locate them via exec.LookPath. The amneziawg-tools build emits a
# single `wg` binary that dispatches on argv[0] — installing it as
# /usr/bin/awg makes it act as the awg CLI. awg-quick is the linux
# bash variant of the wg-quick wrapper, supplied separately.
COPY --from=amneziawg-tools-builder /build/src/wg /usr/bin/awg
COPY --from=amneziawg-tools-builder /build/src/wg-quick/linux.bash /usr/bin/awg-quick
RUN chmod +x /usr/bin/awg /usr/bin/awg-quick \
    && /usr/bin/awg --version 2>&1 | head -1 || true
# Set the Current Working Directory inside the container
WORKDIR /app
# Expose default ports for metrics, web and wireguard
EXPOSE 8787/tcp
EXPOSE 8888/tcp
EXPOSE 51820/udp
# the database and config file can be mounted from the host
VOLUME [ "/app/data", "/app/config" ]
# Command to run the executable
ENTRYPOINT [ "/app/wg-portal" ]
