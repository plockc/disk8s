### Setup

Uses kubebuilder, which currently is 404 for M1 Mac using arkade, so download direct.  Kubebuilder is simpler than Operator SDK (doesn't have the orchestrator lifecycle management, catalogs, bundles)
```
ark install kubebuilder || (curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH) && chmod 755 kubebuilder && mv kubebuilder /usr/local/bin)
```

Create scaffolding
```
kubebuilder init  --plugins=go/v4-alpha --domain disk8s.plockc.org --repo github.com/plockc/disk8s/disk8s-controller
kubebuilder create api --group disk8s --version v1alpha1 --kind Disk --resource --controller --plugins=go/v4-alpha
```

`Makefile` updates:
- `IMG` to use image repository name disk8s-controller
- PLATFORMS to drop `s390x` and `ppc64le` to save time for rare deployments when using build-dockerx, keeps raspi 64 support however

Updated spec and status in api/<version>/disk_types.go

Rebuild the reource model and manifests
```
make generate manifests
```

Install crd and deployment into k8s
```
make install deploy
```

Or run locally
```
make run
```

Or to avoid all the B.S.
```
go run ./main.go
```

---
Tanka is a heavier means to manage the applied manifests customized to various clusters.  It uses jsonnet and does package management with [jsonnet-bundler](https://github.com/jsonnet-bundler/jsonnet-bundler).

This project for stable environments will use argo-cd, sidestepping the need for tanka.

arkade does not support jsonnet/tanka/jsonnet-bundler, so install with brew
```
brew install jsonnet jsonnet-bundler
```

For this repo, which already has a version lock file, just
```
jb install
```

When building a new controller, initialize (just adds a jsonnetfile.json) in a subdirectory, "jsonnet", and download [k8s-libsonnet](https://github.com/jsonnet-libs/k8s-libsonnet) which is a generated library based on kubernetes OpenAPI specificatication along with some helpers from grafana's [ksonnet-util](https://github.com/grafana/jsonnet-libs/blob/master/ksonnet-util/util.libsonnet).
```
jb init
jb install github.com/jsonnet-libs/k8s-libsonnet/1.25@main
jb install github.com/grafana/jsonnet-libs/ksonnet-util
```

Run jsonnet
```
cd environments
jsonnet --yaml-stream -J vendor -J lib default/main.jsonnet
```
