local k = import "k8s.libsonnet";
local kustomized = import "kustomized.json";

[r for r in kustomized if r.kind != "Deployment" && r.kind != "Service" ] + [{
  _config:: {
    name: "disk8s-controller",
  },

  local deploy= k.apps.v1.deployment,
  local controllerContainer = k.controllerContainer,
  local gitVer = if std.extVar("gitVer") == "" then "latest" else std.extVar("gitVer"),

  deployment: deploy.new(name="disk8s-controller", replicas=1, containers=[
    controllerContainer.new("manager", "plockc/"+$._config.name+":"+gitVer)
    +k.core.v1.container.withEnvMixin([
        k.core.v1.envVar.new("GIT_VERSION", gitVer),
    ]),
  ])
  //+deploy.spec.template.spec.securityContext.withRunAsNonRoot(true)
  +deploy.spec.template.spec.withTerminationGracePeriodSeconds(10)
  +deploy.spec.template.spec.withServiceAccountName($._config.name+"-manager")
}.deployment] 
