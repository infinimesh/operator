/*
Copyright 2019 infinimesh, inc.

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

package platform

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	extensionsv1vbeta1 "k8s.io/api/extensions/v1beta1"

	infinimeshv1beta1 "github.com/infinimesh/operator/pkg/apis/infinimesh/v1beta1"
)

var logger = logf.Log.WithName("controller")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Platform Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePlatform{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("platform-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Platform
	err = c.Watch(&source.Kind{Type: &infinimeshv1beta1.Platform{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create
	// Uncomment watch a Deployment created by Platform - change this for objects you create
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &infinimeshv1beta1.Platform{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &infinimeshv1beta1.Platform{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &extensionsv1vbeta1.Ingress{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &infinimeshv1beta1.Platform{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &infinimeshv1beta1.Platform{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcilePlatform{}

// ReconcilePlatform reconciles a Platform object
type ReconcilePlatform struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Platform object and makes changes based on the state read
// and what is in the Platform.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  The scaffolding writes
// a Deployment as an example
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets;services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=extensions,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infinimesh.infinimesh.io,resources=platforms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infinimesh.infinimesh.io,resources=platforms/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubedb.com,resources=postgreses,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcilePlatform) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Platform instance
	instance := &infinimeshv1beta1.Platform{}

	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if err := r.reconcileDgraph(request, instance); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.reconcileMqtt(request, instance); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.reconcileRegistry(request, instance); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.reconcileApiserver(request, instance); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.reconcileApiserverRest(request, instance); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.reconcileNodeserver(request, instance); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.reconcileTelemetryRouter(request, instance); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.reconcileTwin(request, instance); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.reconcileFrontend(request, instance); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.reconcileTimeseries(request, instance); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
