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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"
	"sigs.k8s.io/gateway-api/apis/v1beta1"
)

// TCPRouteReconciler reconciles a Cluster object
type TCPRouteReconciler struct {
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
func (r *TCPRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var TCPRoute v1alpha2.TCPRoute
	if err := r.Get(ctx, req.NamespacedName, &TCPRoute); err != nil {
		if errors.IsNotFound(err) {
			// This will attempt to reconcile the services by deleting the service attached to this TCP Route
			return r.deleteService(ctx, req)
		}
		log.Error(err, "unable to fetch Directions object")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// Find all parent resources (more than likely just one but YOLO)
	for x := range TCPRoute.Spec.ParentRefs {
		// Namespace logic!
		var gatewayNamespace string
		if TCPRoute.Spec.ParentRefs[x].Namespace != nil {
			gatewayNamespace = string(*TCPRoute.Spec.ParentRefs[x].Namespace)
		} else {
			gatewayNamespace = TCPRoute.Namespace
		}

		// Find the parent gateway!
		key := types.NamespacedName{
			Namespace: gatewayNamespace,
			Name:      string(TCPRoute.Spec.ParentRefs[x].Name),
		}

		gateway := &v1beta1.Gateway{}
		err := r.Client.Get(ctx, key, gateway, nil)
		if err != nil {
			log.Info(fmt.Sprintf("Unknown Gateway [%v]", key.String()))
		} else {

			// Find our listener
			if TCPRoute.Spec.ParentRefs[x].SectionName != nil {
				listener := &v1beta1.Listener{}
				for x := range gateway.Spec.Listeners {
					if gateway.Spec.Listeners[x].Name == *TCPRoute.Spec.ParentRefs[x].SectionName {
						listener = &gateway.Spec.Listeners[x]
					}
				}
				if listener != nil {
					// We've found our listener!
					// At this point we have our entrypoint

					// Now to parse our backends  ¯\_(ツ)_/¯
					for y := range TCPRoute.Spec.Rules {
						for z := range TCPRoute.Spec.Rules[y].BackendRefs {
							// Namespace logic!
							var serviceNamespace string
							if TCPRoute.Spec.Rules[y].BackendRefs[z].Namespace != nil {
								serviceNamespace = string(*TCPRoute.Spec.ParentRefs[x].Namespace)
							} else {
								serviceNamespace = TCPRoute.Namespace
							}
							err = r.reconcileService(ctx, string(TCPRoute.Spec.Rules[y].BackendRefs[z].Name), serviceNamespace, TCPRoute.Name, int(listener.Port), int(*TCPRoute.Spec.Rules[y].BackendRefs[z].Port))
							if err != nil {
								return ctrl.Result{}, err
							}
						}

					}

				} else {
					log.Info(fmt.Sprintf("Unknown Listener on gateway [%s]", *TCPRoute.Spec.ParentRefs[x].SectionName))
				}
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *TCPRouteReconciler) deleteService(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// We will get ALL services in the namespace
	var services v1.ServiceList
	err := r.List(ctx, &services, &client.ListOptions{Namespace: req.Namespace})
	if err != nil {
		return ctrl.Result{}, err
	}
	for x := range services.Items {
		// Find out if we manage this item AND it references this TCPRoute object
		if services.Items[x].Annotations["gateway-api-controller"] == r.ControllerName && services.Items[x].Annotations["parent-tcp-route"] == req.Name {
			err = r.Delete(ctx, &services.Items[x], &client.DeleteOptions{})
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *TCPRouteReconciler) reconcileService(ctx context.Context, name, namespace, parentName string, port, targetport int) error {
	// does our service exist?
	var service v1.Service
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	err := r.Get(ctx, key, &service)
	// if errors.IsNotFound(err) {

	// } else {
	//
	if err != nil {
		return err
	}

	// Create our service
	service.Name = name + "-gw-api"
	service.Namespace = namespace
	service.ResourceVersion = ""
	service.Spec.ClusterIP = ""
	service.Spec.ClusterIPs = []string{}
	// Initialise the labels
	service.Annotations = map[string]string{}
	service.Annotations["gateway-api-controller"] = r.ControllerName
	service.Annotations["parent-tcp-route"] = parentName
	// Set service configuration
	service.Spec.Type = v1.ServiceTypeLoadBalancer
	service.Spec.Ports = []v1.ServicePort{
		{
			TargetPort: intstr.FromInt(targetport),
			Port:       int32(port),
		},
	}
	err = r.Create(ctx, &service, &client.CreateOptions{})
	if err != nil {
		return err
	}
	// All gravy
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TCPRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha2.TCPRoute{}).
		Complete(r)
}
