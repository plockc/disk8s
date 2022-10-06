## Development

Install Rancher Desktop, which will use lima and k3s

Had trouble with port forwarding with containerd, with moby, can do normal kubectl port forward or port forward using the desktop UI.

Given the more complicated networking and the intent is dev only, not going to worry about replacing klipper

## vcluster

Install vcluster for creating virtual clusters within the local k8s cluster.

```
vcluster create test
```

The kubeconfig will be set to the vcluster in all shells, to restore to the parent context

```
vcluster disconnect
```

## devspace

```
brew install devspace
```


## Argo CD

Runs agent on cluster, avoids configuring a CI system to interact with the cluster, as usually cluster already has access to containers

Also able to reconcile automatically

Best practice to separate the system configurations in separate repo, avoids CI runs for non-code changes:  CI only upates the deployment.yaml

Argo supports yaml, helm, kustomize

