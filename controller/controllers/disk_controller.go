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

	disk8sv1alpha1 "github.com/plockc/disk8s/disk8s-controller/api/v1alpha1"
)

const replicatedDiskPrefix = "nbd-server"

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

	var disk disk8sv1alpha1.Disk
	if err := r.Get(ctx, req.NamespacedName, &disk); err != nil {
		l.Info("Disk " + req.Name + " has been deleted")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	///////////
	// Ensure Persistent Volume Claim for Replicated Disk
	///////////
	pvcName := "nbd-0-" + req.Name
	oMeta := metav1.ObjectMeta{Name: pvcName, Namespace: os.Getenv("K8S_POD_NAMESPACE")}
	var pvc = corev1.PersistentVolumeClaim{ObjectMeta: oMeta}
	res, err := controllerutil.CreateOrUpdate(ctx, r.Client, &pvc, func() error {
		size, err := resource.ParseQuantity("100Mi")
		if err != nil {
			return err
		}
		mutateNbdServerPersistentVolumeClaim(&pvc, size)
		controllerutil.SetControllerReference(&disk, &pvc, r.Scheme)
		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	l.Info("Replicated Disk Persistent Volume " + string(res))

	///////////
	// Ensure Deployment for Replicated Disk
	///////////
	deployName := replicatedDiskPrefix + "-" + req.Name
	oMeta = metav1.ObjectMeta{Name: deployName, Namespace: os.Getenv("K8S_POD_NAMESPACE")}
	var deploy = appsv1.Deployment{ObjectMeta: oMeta}
	res, err = controllerutil.CreateOrUpdate(ctx, r.Client, &deploy, func() error {
		mutateNbdServerDeployment(&deploy, req.Name, pvcName)
		controllerutil.SetControllerReference(&disk, &deploy, r.Scheme)
		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	l.Info("Replicated Disk Deployment " + string(res))

	///////////
	// Ensure Service for Replicated Disk
	///////////
	serviceName := "nbd-" + req.Name
	oMeta = metav1.ObjectMeta{Name: serviceName, Namespace: os.Getenv("K8S_POD_NAMESPACE")}
	var svc = corev1.Service{ObjectMeta: oMeta}
	res, err = controllerutil.CreateOrUpdate(ctx, r.Client, &svc, func() error {
		mutateNbdServerService(&svc, req.Name)
		controllerutil.SetControllerReference(&disk, &svc, r.Scheme)
		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	l.Info("Replicated Disk Service " + string(res))

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DiskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&disk8sv1alpha1.Disk{}).
		Complete(r)
}

func mutateNbdServerPersistentVolumeClaim(pvc *corev1.PersistentVolumeClaim, size resource.Quantity) {
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

func mutateNbdServerService(svc *corev1.Service, diskName string) {
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
						Image:           "nbd-server:latest",
						ImagePullPolicy: corev1.PullIfNotPresent,
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
