/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	disk8sv1alpha1 "github.com/plockc/disk8s/controller/api/v1alpha1"
)

const (
	replicatedDiskPrefix = "nbd-server"
	replicaDiskPrefix    = "replica"
)

var gitVersionLdFlag string

func init() {
	gitVersionLdFlag = os.Getenv("GIT_VERSION")
	if os.Getenv("GIT_VERSION") == "" {
		gitVersionLdFlag = "latest"
	}
	fmt.Println("Using container image tag " + gitVersionLdFlag)
}

// DiskReconciler reconciles a Disk object
type DiskReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=disk8s.plockc.org,resources=disks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=disk8s.plockc.org,resources=disks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=disk8s.plockc.org,resources=disks/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Disk object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *DiskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	l.Info("reconciling")

	namespace := os.Getenv("K8S_POD_NAMESPACE")
	var disk disk8sv1alpha1.Disk
	if err := r.Get(ctx, req.NamespacedName, &disk); err != nil {
		l.Info("Disk " + req.Name + " has been deleted")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	///////////
	// Ensure Persistent Volume Claim for Replicated Disk
	///////////
	size, err := resource.ParseQuantity("100Mi")
	if err != nil {
		return ctrl.Result{}, err
	}
	oMeta := metav1.ObjectMeta{Name: "nbd-" + req.Name, Namespace: namespace}
	var pvc = corev1.PersistentVolumeClaim{ObjectMeta: oMeta}
	if err = createOrUpdate(ctx, r, &disk, &pvc, func(pvc *corev1.PersistentVolumeClaim, _, name string) {
		mutateNbdServerPersistentVolumeClaim(pvc, size)
	}); err != nil {
		return ctrl.Result{}, err
	}

	///////////
	// Ensure Deployment for Replicated Disk
	///////////
	oMeta = metav1.ObjectMeta{Name: replicatedDiskPrefix + "-" + req.Name, Namespace: namespace}
	var deploy = appsv1.Deployment{ObjectMeta: oMeta}
	if err = createOrUpdate(ctx, r, &disk, &deploy, func(deploy *appsv1.Deployment, diskName, _ string) {
		mutateNbdServerDeployment(deploy, req.Name, pvc.GetName())
	}); err != nil {
		return ctrl.Result{}, err
	}

	///////////
	// Ensure Service for Replicated Disk
	///////////
	oMeta = metav1.ObjectMeta{Name: "nbd-" + req.Name, Namespace: namespace}
	var svc = corev1.Service{ObjectMeta: oMeta}
	if err = createOrUpdate(ctx, r, &disk, &svc, mutateNbdServerService); err != nil {
		return ctrl.Result{}, err
	}

	///////////
	// Ensure StatefulSet for Replica (Data) Disk
	///////////
	oMeta = metav1.ObjectMeta{Name: replicaDiskPrefix + "-" + req.Name, Namespace: namespace}
	ss := appsv1.StatefulSet{ObjectMeta: oMeta}
	if err = createOrUpdate(ctx, r, &disk, &ss, func(ss *appsv1.StatefulSet, diskName, _ string) {
		mutateReplicaStatefulSet(ss, diskName, size)
	}); err != nil {
		return ctrl.Result{}, err
	}

	///////////
	// Ensure Service for Replica (Data) Disk
	///////////
	oMeta = metav1.ObjectMeta{Name: "replica-" + req.Name, Namespace: namespace}
	svc = corev1.Service{ObjectMeta: oMeta}
	if err = createOrUpdate(ctx, r, &disk, &svc, mutateReplicaService); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// expects the Metadata is already set for Name and GVK for the object
func createOrUpdate[Obj client.Object](ctx context.Context, r *DiskReconciler, disk client.Object, o Obj, mutator func(o Obj, diskName, name string)) error {
	l := log.FromContext(ctx)
	res, err := controllerutil.CreateOrUpdate(ctx, r.Client, o, func() error {
		mutator(o, disk.GetName(), o.GetName())
		return controllerutil.SetControllerReference(disk, o, r.Scheme)
	})
	if err != nil {
		return err
	}
	l.Info(o.GetObjectKind().GroupVersionKind().Kind + " " + o.GetName() + " " + string(res))
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DiskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&disk8sv1alpha1.Disk{}).
		Complete(r)
}

func mutateReplicaPersistentVolumeClaim(pvc *corev1.PersistentVolumeClaim, size resource.Quantity) {
	fs := corev1.PersistentVolumeFilesystem
	pvc.Spec = corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		VolumeMode:  &fs,
		Resources: corev1.ResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceStorage: size,
			},
		},
	}
}

func mutateReplicaService(svc *corev1.Service, diskName, _ string) {
	nbd := "nbd"
	serviceType := corev1.ServiceInternalTrafficPolicyCluster
	// external traffic coming in will try to use the local node
	// PureLB documents this is supported - see External Traffic Policy section
	// https://purelb.gitlab.io/docs/how_it_works/
	// slightly dated kubernetes reference:
	// https://www.asykim.com/blog/deep-dive-into-kubernetes-external-traffic-policies
	svc.Spec = corev1.ServiceSpec{
		Type:                  corev1.ServiceTypeClusterIP,
		InternalTrafficPolicy: &serviceType,
		IPFamilies:            []corev1.IPFamily{corev1.IPv4Protocol},
		Ports: []corev1.ServicePort{
			{
				AppProtocol: &nbd,
				Name:        "nbd",
				Port:        10808,
			},
		},
		Selector: map[string]string{
			"app.kubernetes.io/name":      "disk8s-storage",
			"app.kubernetes.io/instance":  diskName,
			"app.kubernetes.io/component": "storage-replica",
		},
	}
}

func mutateReplicaStatefulSet(ss *appsv1.StatefulSet, diskName string, size resource.Quantity) {
	var replicas int32 = 1
	var termGracePeriod int64 = 10
	fs := corev1.PersistentVolumeFilesystem
	name := "replica-" + diskName
	ss.Spec = appsv1.StatefulSetSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app.kubernetes.io/name":      "disk8s-storage",
				"app.kubernetes.io/instance":  diskName,
				"app.kubernetes.io/component": "storage-replica",
				"app.kubernetes.io/part-of":   "replicated-disk",
			},
		},
		ServiceName: name,
		PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
			WhenDeleted: appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
			WhenScaled:  appsv1.RetainPersistentVolumeClaimRetentionPolicyType,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app.kubernetes.io/name":      "disk8s-storage",
					"app.kubernetes.io/instance":  diskName,
					"app.kubernetes.io/component": "storage-replica",
					"app.kubernetes.io/part-of":   "replicated-disk",
				},
			},
			Spec: corev1.PodSpec{
				TerminationGracePeriodSeconds: &termGracePeriod,
				Containers: []corev1.Container{
					{
						Name:            "disk",
						Image:           "plockc/replica:" + gitVersionLdFlag,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Ports: []corev1.ContainerPort{
							{
								Name:          "grpc",
								Protocol:      corev1.ProtocolTCP,
								ContainerPort: 10808,
							},
						},
						VolumeMounts: []corev1.VolumeMount{{
							Name: "data", MountPath: "/data",
						}},
					},
				},
			},
		},
		VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "data",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					VolumeMode:  &fs,
					Resources: corev1.ResourceRequirements{
						Requests: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceStorage: size,
						},
					},
				},
			},
		},
	}
}

func mutateNbdServerPersistentVolumeClaim(pvc *corev1.PersistentVolumeClaim, size resource.Quantity) {
	fs := corev1.PersistentVolumeFilesystem
	// the pvc.Spec can be modified by k8s controllers, so just modify fields we manage
	pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	pvc.Spec.VolumeMode = &fs
	pvc.Spec.Resources = corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceStorage: size,
		},
	}
}

func mutateNbdServerService(svc *corev1.Service, diskName, _ string) {
	f := false
	nbd := "nbd"
	// external traffic coming in will try to use the local node
	// PureLB documents this is supported - see External Traffic Policy section
	// https://purelb.gitlab.io/docs/how_it_works/
	// slightly dated kubernetes reference:
	// https://www.asykim.com/blog/deep-dive-into-kubernetes-external-traffic-policies
	svc.Spec = corev1.ServiceSpec{
		Type:                          corev1.ServiceTypeLoadBalancer,
		AllocateLoadBalancerNodePorts: &f,
		ExternalTrafficPolicy:         corev1.ServiceExternalTrafficPolicyTypeLocal,
		IPFamilies:                    []corev1.IPFamily{corev1.IPv4Protocol},
		Ports: []corev1.ServicePort{
			{
				AppProtocol: &nbd,
				Name:        "nbd",
				Port:        10809,
			},
		},
		Selector: map[string]string{
			replicatedDiskPrefix: diskName,
		},
	}
}

func mutateNbdServerDeployment(deploy *appsv1.Deployment, diskName, pvcName string) {
	var replicas int32 = 1
	deploy.Spec = appsv1.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				replicatedDiskPrefix: diskName,
			},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					replicatedDiskPrefix: diskName,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            "disk",
						Image:           "plockc/nbd-server:" + gitVersionLdFlag,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Env:             []corev1.EnvVar{{Name: "REMOTE_STORAGE", Value: "replica-" + diskName + "-0.replica-sample:10808"}},
						Ports: []corev1.ContainerPort{
							{
								Name:          "nbd",
								Protocol:      corev1.ProtocolTCP,
								ContainerPort: 10809,
							},
						},
						VolumeMounts: []corev1.VolumeMount{{
							Name: "data", MountPath: "/data",
						}},
					},
				},
				Volumes: []corev1.Volume{{Name: "data", VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName},
				}}},
			},
		},
	}
}

func getKind(s *runtime.Scheme, o client.Object) string {
	k := o.GetObjectKind().GroupVersionKind().Kind
	if k != "" {
		return k
	}
	gvks, _, err := s.ObjectKinds(o)
	if err != nil {
		return ""
	}
	for _, gvk := range gvks {
		if len(gvk.Kind) == 0 {
			continue
		}
		if len(gvk.Version) == 0 || gvk.Version == runtime.APIVersionInternal {
			continue
		}
		o.GetObjectKind().SetGroupVersionKind(gvk)
		break
	}
	return ""
}
