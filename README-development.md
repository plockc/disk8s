### Setup

Uses [OperatorSDK](https://sdk.operatorframework.io/)
```
brew install operator-sdk
```

Creted scaffolding
```
operator-sdk init  --plugins=go/v4-alpha --domain plockc.github.io --repo github.com/plockc/disk8s
operator-sdk create api --group disk8s --version v1alpha1 --kind Disk --resource --controller --plugins=go/v4-alpha
```

Updated spec and status in api/<version>/disk_types.go

Rebuild the reource model and manifests
`make generate manifests`

