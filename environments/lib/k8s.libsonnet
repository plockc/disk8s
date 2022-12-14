// aliasing k8s-libsonnet to simply k8s, k8s-libsonnet is the jsonnet
// translation of the k8s OpenAPI
(import "github.com/jsonnet-libs/k8s-libsonnet/1.25/main.libsonnet")
// all containters usually want IfNotPresent, replace the default "Always"
+ {
  core+: { v1+: {
    container+: {
      new(name, image)::
        super.new(name, image) + super.withImagePullPolicy("IfNotPresent"),
    },
  }},
}
// add a special controllerContainer that largely follows kubebuilder defaults
// for liveness, readyness.  Also allows for leader election.
// rbac proxy is currently missing
+ {
  controllerContainer+:: {
    local container = $.core.v1.container,
    local livenessProbe = container.livenessProbe,
    local readinessProbe = container.readinessProbe,
    local resources = container.resources,
    local securityContext = container.securityContext,
    local envVar = $.core.v1.envVar,
    local envVarFromFieldRef = $.core.v1.envVar.valueFrom.fieldRef,

    new(name, image)::
      $.core.v1.container.new(name, image)
        +container.withArgs([
          "--leader-elect",
        ])
        +container.withCommand("/manager")
        // this concept is explained here: https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/
        // more values can be found: https://kubernetes.io/docs/concepts/workloads/pods/downward-api/#available-fields
        +container.withEnv([
            envVar.withName("K8S_POD_NAMESPACE")+
            envVarFromFieldRef.withFieldPath("metadata.namespace")
        ])
        +livenessProbe.withInitialDelaySeconds(15)
        +livenessProbe.withPeriodSeconds(20)
        +livenessProbe.httpGet.withPath("/healthz")
        +livenessProbe.httpGet.withPort(8081)
        +readinessProbe.withInitialDelaySeconds(5)
        +readinessProbe.withPeriodSeconds(10)
        +readinessProbe.httpGet.withPath("/readyz")
        +readinessProbe.httpGet.withPort(8081)
        +resources.withRequests({cpu: "10m", memory: "128Mi"})
        // need more memory to be able to compile in devspace, and want extra cpu
        +resources.withLimits({cpu: "1000m", memory: "768Mi"})
        +securityContext.withAllowPrivilegeEscalation(false)
        +securityContext.capabilities.withDrop("ALL")
  },
}
