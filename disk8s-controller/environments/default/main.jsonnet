local k = import "k8s.libsonnet";

[{
  _config:: {
    name: "disk8s-controller",
  },
  local deploy= k.apps.v1.deployment,
  local controllerContainer = k.controllerContainer,

  disk8s: {
    deployment: deploy.new(name="disk8s-controller", replicas=1, containers=[
      controllerContainer.new($._config.name, "plockc/"+$._config.name),
    ]),
  }
}]
