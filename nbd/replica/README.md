## gRPC

Install protoc
```
brew install protobuf
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Update generated code
```
cd replica
protoc --go_out=./ --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=./ data-disk.proto
```

## Development

Use [devspace](https://www.devspace.sh/docs/getting-started/installation)

```
brew install devspace
```

devspace.yaml has already been updated to
- sync the parent directory to include the whole module
- change working directory to target package
- persist go module downloads
- labels selector to only work with the manifest
- skip pushing image to docker hub and use parent dir for docker context

```
workingDir: /app/replica
sync:
  - path: ../:/app
persistPaths:
  - path: /go/pkg/mod/cache/download
# comment out the imageSelector
labelSelector:
  app: nbd
images:
  app:
    image: replica:latest
    skipPush: true
    dockerfile: ../Dockerfile.replica
    context: ../
```

To start development, run

```
devspace dev
```

Then run the binary
```
go run ./cmd
```

Then edit locally, and Ctrl-C the process to pick up changes

## Debug

change the label selector to pick the container to replace
