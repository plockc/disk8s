## Argo CD

Runs agent on cluster, avoids configuring a CI system to interact with the cluster, as usually cluster already has access to containers

Also able to reconcile automatically

Best practice to separate the system configurations in separate repo, avoids CI runs for non-code changes:  CI only upates the deployment.yaml

Argo supports yaml, helm, kustomize

