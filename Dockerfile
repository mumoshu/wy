# `--platform=$BUILDPLATFORM` makes sure that golang image is always native to the build machine
# See https://www.docker.com/blog/faster-multi-platform-builds-dockerfile-cross-compilation-guide/#:~:text=Preparing%20Dockerfile
FROM --platform=$BUILDPLATFORM golang:1.17 as builder

WORKDIR /workspace

ENV GO111MODULE=on \
  CGO_ENABLED=0

# Copy the Go Modules manifests
COPY go.mod go.sum ./

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . .

ARG TARGETOS TARGETARCH

# Build
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -a -o wy ./

WORKDIR /workspace/aws

# Build our own minimalistic `aws eks get-token` command
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -a -o aws ./

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /workspace/wy .
COPY --from=builder /workspace/aws/aws /bin/aws

USER nonroot:nonroot

ENTRYPOINT ["/wy"]
