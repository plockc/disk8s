local k = import "k8s.libsonnet";

[{
  _config:: {
    name: "disk8s-controller",
  },
  local deploy= k.apps.v1.deployment,
  local service = k.core.v1.service,
  local container = k.core.v1.container,

  disk8s: {
    deployment: deploy.new(name="disk8s-controller", replicas=1, containers=[
      container.new($._config.name, "plockc/"+$._config.name),
    ]),
  }
}]
