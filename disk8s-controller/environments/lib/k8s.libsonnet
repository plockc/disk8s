(import "github.com/jsonnet-libs/k8s-libsonnet/1.25/main.libsonnet")
+ {
  core+: { v1+: {
    container+: {
      new(name, image)::
        super.new(name, image) + super.withImagePullPolicy("IfNotPresent"),
    },
  }},
}
+ {
  controllerContainer+:: {
      new(name, image)::
      $.core.v1.container.new(name, image),
  },
}
