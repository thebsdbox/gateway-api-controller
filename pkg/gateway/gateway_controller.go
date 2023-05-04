/*
Copyright 2022 Dan Finneran.

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

package gateway

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/gateway-api/apis/v1beta1"
)

// GatewayReconciler reconciles a Cluster object
type GatewayReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	ControllerName string
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Cluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var gateway v1beta1.Gateway
	if err := r.Get(ctx, req.NamespacedName, &gateway); err != nil {
		if errors.IsNotFound(err) {
			// object not found, could have been deleted after
			// reconcile request, hence don't requeue
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch gateway object")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// your logic here
	log.Info("Reconciling Cluster", "Cluster", gateway.Name)

	// Retrieve all Gateway Classes
	// gatewaysClasses := &v1beta1.GatewayClassList{}

	// err := r.Client.List(ctx, gatewaysClasses, &client.ListOptions{})
	// if err != nil {
	// 	log.Error(err, "unable retrieve all gatway classes")
	// }
	//log.Info("Found gatewayclasses", "Cluster", gatewaysClasses)

	// Retrieve the gatewayclass referenced by this gateway
	gatewayClass := &v1beta1.GatewayClass{}
	key := types.NamespacedName{
		Namespace: gateway.Namespace,
		Name:      string(gateway.Spec.GatewayClassName),
	}
	err := r.Client.Get(ctx, key, gatewayClass, nil)
	if err != nil {
		log.Info(fmt.Sprintf("Wrong Class [%v]", key.String()))
	}
	log.Info("Found gatewayclass", "Cluster", gatewayClass)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.Gateway{}).
		Complete(r)
}
