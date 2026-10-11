# syntax=docker/dockerfile:1
# Multi-stage build for the dsx-lab binary (init/serve/check/scenario
# subcommands). Pinned, verified base image tags; no :latest.

FROM golang:1.25.11-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/dsx-lab ./cmd/dsx-lab

# Minimal, multi-platform (amd64/arm64), non-root-by-default final image.
# Compose overrides the runtime uid/gid explicitly (see compose.yaml); this
# default only matters for a bare `docker run` of the image.
FROM gcr.io/distroless/static-debian12:nonroot AS final
COPY --from=builder /out/dsx-lab /usr/local/bin/dsx-lab
ENTRYPOINT ["/usr/local/bin/dsx-lab"]
