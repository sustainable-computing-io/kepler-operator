package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type keplerSADescription struct {
	Context            context.Context
	Client             client.Client
	Scheme             *runtime.Scheme
	SA                 *corev1.ServiceAccount
	Owner              metav1.Object
	clusterRole        *rbacv1.ClusterRole
	clusterRoleBinding *rbacv1.ClusterRoleBinding
}

func (d *keplerSADescription) Reconcile(l klog.Logger) (bool, error) {
	return reconcileBatch(l,
		d.ensureSA,
		d.ensureRole,
		d.ensureRoleBinding,
	)
}

func (d *keplerSADescription) ensureSA(l klog.Logger) (bool, error) {
	logger := l.WithValues("ServiceAccount", nameFor(d.SA))
	op, err := ctrlutil.CreateOrUpdate(d.Context, d.Client, d.SA, func() error {
		if err := ctrl.SetControllerReference(d.Owner, d.SA, d.Scheme); err != nil {
			logger.Error(err, "unable to set controller reference")
			return err
		}
		return nil
	})
	if err != nil {
		logger.Error(err, "ServiceAccount reconcile failed")
		return false, err
	}

	logger.V(1).Info("ServiceAccount reconciled", "operation", op)
	return true, nil
}

func (d *keplerSADescription) ensureRole(l klog.Logger) (bool, error) {

	d.clusterRole = &rbacv1.ClusterRole{
    TypeMeta: metav1.TypeMeta{
			Kind: "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kepler-clusterrole",
		},
	}

	logger := l.WithValues("ClusterRole", nameFor(d.clusterRole))
	_, err := d.createOrUpdateClusterRole(l)

	if err != nil {
		logger.Error(err, "ClusterRole reconcile failed")
		return false, err
	}
	return true, nil
}

func (d *keplerSADescription) ensureRoleBinding(l klog.Logger) (bool, error) {
	d.clusterRoleBinding = &rbacv1.ClusterRoleBinding{
    TypeMeta: metav1.TypeMeta{
      Kind: "ClusterRoleBinding",
    }
		ObjectMeta: metav1.ObjectMeta{
			Name: "kepler-clusterrole-binding",
		},
	}
	d.clusterRoleBinding.RoleRef = rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     "kepler-clusterrole",
	}
	d.clusterRoleBinding.Subjects = []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      d.SA.Name,
			Namespace: d.SA.Namespace,
		},
	}

	logger := l.WithValues("RoleBinding", nameFor(d.clusterRoleBinding))

	found := &rbacv1.ClusterRoleBinding{}
	err := d.Client.Get(context.TODO(), types.NamespacedName{Name: "kepler-clusterrole-binding", Namespace: ""}, found)

	if err != nil && !apierrors.IsNotFound(err) {
		return false, err
	}
	if apierrors.IsNotFound(err) {
		err = d.Client.Create(context.TODO(), d.clusterRoleBinding)
		if err != nil {
			return false, err
		}
	}
	err = d.Client.Update(context.TODO(), d.clusterRoleBinding)
	if err != nil {
		logger.Error(err, "ClusterRoleBinding reconcile failed")
		return false, err
	}
	logger.V(1).Info("ClusterRoleBinding reconciled", "clusterRoleBinding", d.clusterRoleBinding)
	return true, nil
}

func (d *keplerSADescription) createOrUpdateClusterRole(l klog.Logger) (*rbacv1.ClusterRole, error) {

	d.clusterRole = &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kepler-clusterrole",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"nodes/metrics", "nodes/proxy", "nodes/stats"},
				Verbs:     []string{"get", "watch", "list"},
			},
		},
	}

	logger := l.WithValues("kepler-clusterrole", d.clusterRole.ObjectMeta.Name)
	found := &rbacv1.ClusterRole{}
	err := d.Client.Get(context.TODO(), types.NamespacedName{Name: "kepler-clusterrole", Namespace: ""}, found)

	if err != nil && !apierrors.IsNotFound(err) {

		return nil, err
	}
	if apierrors.IsNotFound(err) {
		err = d.Client.Create(context.TODO(), d.clusterRole)
		if err != nil {
			return nil, err
		}
	}
	err = d.Client.Update(context.TODO(), d.clusterRole)
	if err != nil {
		return nil, err
	}
	logger.V(1).Info("ClusterRole", "clusterRole", d.clusterRole)
	return d.clusterRole, nil
}
