# Build the manager binary
FROM golang:1.19 as builder
ARG TARGETOS
ARG TARGETARCH

RUN apt update && apt install -y upx

RUN mkdir /data
RUN chown 65532:65532 /data

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod

# have no dependencies,yet
# COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/ cmd/
COPY ./*.go ./

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -a -o nbd-server cmd/main.go

# compress the binary
RUN upx --best --lzma nbd-server

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
ENV DISK_PATH=/data/disk8s.data
COPY --from=builder /workspace/nbd-server .
COPY --from=builder /data .
USER 65532:65532

ENTRYPOINT ["/nbd-server"]
