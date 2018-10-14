/*
Copyright 2018 The Kubernetes Authors.

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

package eventhandler_test

import (
	"github.com/tsungming/controller-runtime/pkg/ctrl"
	"github.com/tsungming/controller-runtime/pkg/ctrl/event"
	"github.com/tsungming/controller-runtime/pkg/ctrl/eventhandler"
	"github.com/tsungming/controller-runtime/pkg/ctrl/reconcile"
	"github.com/tsungming/controller-runtime/pkg/ctrl/source"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
)

var controller ctrl.Controller

// This example watches Pods and enqueues ReconcileRequests with the Name and Namespace of the Pod from
// the Event (i.e. change caused by a Create, Update, Delete).
func ExampleEnqueueHandler() {
	// controller is a ctrl.controller
	controller.Watch(
		&source.KindSource{Type: &corev1.Pod{}},
		&eventhandler.EnqueueHandler{},
	)
}

// This example watches ReplicaSets and enqueues a ReconcileRequest containing the Name and Namespace of the
// owning (direct) Deployment responsible for the creation of the ReplicaSet.
func ExampleEnqueueOwnerHandler_1() {
	// controller is a ctrl.controller
	controller.Watch(
		&source.KindSource{Type: &appsv1.ReplicaSet{}},
		&eventhandler.EnqueueOwnerHandler{
			OwnerType:    &appsv1.Deployment{},
			IsController: true,
		},
	)
}

// This example watches Deployments and enqueues a ReconcileRequest contain the Name and Namespace of different
// objects (of Type: MyKind) using a mapping function defined by the user.
func ExampleEnqueueMappedHandler() {
	// controller is a ctrl.controller
	controller.Watch(
		&source.KindSource{Type: &appsv1.Deployment{}},
		&eventhandler.EnqueueMappedHandler{
			ToRequests: eventhandler.ToRequestsFunc(func(a eventhandler.MapObject) []reconcile.ReconcileRequest {
				return []reconcile.ReconcileRequest{
					{NamespacedName: types.NamespacedName{
						Name:      a.Meta.GetName() + "-1",
						Namespace: a.Meta.GetNamespace(),
					}},
					{NamespacedName: types.NamespacedName{
						Name:      a.Meta.GetName() + "-2",
						Namespace: a.Meta.GetNamespace(),
					}},
				}
			}),
		})
}

// This example implements eventhandler.EnqueueHandler.
func ExampleEventHandlerFunc() {
	// controller is a ctrl.controller
	controller.Watch(
		&source.KindSource{Type: &corev1.Pod{}},
		eventhandler.EventHandlerFuncs{
			CreateFunc: func(q workqueue.RateLimitingInterface, e event.CreateEvent) {
				q.Add(reconcile.ReconcileRequest{NamespacedName: types.NamespacedName{
					Name:      e.Meta.GetName(),
					Namespace: e.Meta.GetNamespace(),
				}})
			},
			UpdateFunc: func(q workqueue.RateLimitingInterface, e event.UpdateEvent) {
				q.Add(reconcile.ReconcileRequest{NamespacedName: types.NamespacedName{
					Name:      e.MetaNew.GetName(),
					Namespace: e.MetaNew.GetNamespace(),
				}})
			},
			DeleteFunc: func(q workqueue.RateLimitingInterface, e event.DeleteEvent) {
				q.Add(reconcile.ReconcileRequest{NamespacedName: types.NamespacedName{
					Name:      e.Meta.GetName(),
					Namespace: e.Meta.GetNamespace(),
				}})
			},
			GenericFunc: func(q workqueue.RateLimitingInterface, e event.GenericEvent) {
				q.Add(reconcile.ReconcileRequest{NamespacedName: types.NamespacedName{
					Name:      e.Meta.GetName(),
					Namespace: e.Meta.GetNamespace(),
				}})
			},
		},
	)
}
