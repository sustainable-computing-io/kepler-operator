// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package controller

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
