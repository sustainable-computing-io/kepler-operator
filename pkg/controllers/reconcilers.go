/*
Copyright 2023.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sustainable.computing.io/kepler-operator/pkg/reconciler"
)

func resourceReconcilers(fn reconcileFn, resources ...client.Object) []reconciler.Reconciler {
	rs := []reconciler.Reconciler{}
	for _, res := range resources {
		rs = append(rs, fn(res))
	}
	return rs
}

// TODO: decide if this this should move to reconciler
type reconcileFn func(client.Object) reconciler.Reconciler

// newUpdaterWithOwner returns a reconcileFn that update the resource and
// sets the owner reference to the owner
func newUpdaterWithOwner(owner metav1.Object) reconcileFn {
	return func(obj client.Object) reconciler.Reconciler {
		return &reconciler.Updater{Owner: owner, Resource: obj}
	}
}

// deleteResource is a resourceFn that deletes resources
func deleteResource(obj client.Object) reconciler.Reconciler {
	return &reconciler.Deleter{Resource: obj}
}
