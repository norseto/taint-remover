# Use the native build platform to avoid emulating the Go compiler.
FROM --platform=$BUILDPLATFORM golang:1.26.6-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
ARG GITVERSION

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/main.go cmd/main.go
COPY version.go version.go
COPY api/ api/
COPY internal/controller/ internal/controller/
COPY internal/taints/ internal/taints/

# Cross-compile the manager binary for the target image platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build \
    -ldflags=all="-X github.com/norseto/taint-remover.GitVersion=${GITVERSION}" -a -o manager cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
