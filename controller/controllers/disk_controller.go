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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	disk8sv1alpha1 "github.com/plockc/disk8s/disk8s-controller/api/v1alpha1"
)

// DiskReconciler reconciles a Disk object
type DiskReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=disk8s.plockc.org,resources=disks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=disk8s.plockc.org,resources=disks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=disk8s.plockc.org,resources=disks/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

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
		l.Error(err, "unable to fetch Disk")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	deployNamespacedName := types.NamespacedName{
		Namespace: os.Getenv("K8S_POD_NAMESPACE"),
		Name:      "replicated-disk-" + req.Name,
	}
	l.Info("NAMESSPACE is " + deployNamespacedName.Namespace)

	var deploy appsv1.Deployment
	if err := r.Get(ctx, deployNamespacedName, &deploy); err != nil {
		l.Error(err, "unable to list child Pods")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DiskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&disk8sv1alpha1.Disk{}).
		Complete(r)
}
