package controllers

import (
	"k8s.io/klog/v2"
)

// reconcileFunc is a function that partially reconciles an object. It returns a
// bool indicating whether reconciling should continue and an error.
type reconcileFunc func(klog.Logger) (bool, error)

// reconcileBatch steps through a list of reconcile functions until one returns
// false or an error.
func reconcileBatch(l klog.Logger, reconcileFuncs ...reconcileFunc) (bool, error) {
	for _, f := range reconcileFuncs {
		if cont, err := f(l); !cont || err != nil {
			return cont, err
		}
	}
	return true, nil
}
