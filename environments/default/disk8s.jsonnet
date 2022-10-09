local k = import "k8s.libsonnet";
local kustomized = import "kustomized.json";

[r for r in kustomized if r.kind != "Deployment" && r.kind != "Service" ] + [{
  _config:: {
    name: "disk8s-controller",
  },

  local deploy= k.apps.v1.deployment,
  local controllerContainer = k.controllerContainer,

  deployment: deploy.new(name="disk8s-controller", replicas=1, containers=[
    controllerContainer.new("manager", "plockc/"+$._config.name+":latest"),
  ])
  +deploy.spec.template.spec.securityContext.withRunAsNonRoot(true)
  +deploy.spec.template.spec.withTerminationGracePeriodSeconds(10)
  +deploy.spec.template.spec.withServiceAccountName($._config.name+"-controller-manager")
}.deployment] 
