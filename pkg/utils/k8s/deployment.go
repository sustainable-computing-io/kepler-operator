// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeploymentBuilder struct {
	d appsv1.Deployment
}

func (b DeploymentBuilder) Build() *appsv1.Deployment {
	return &b.d
}

func Deployment(ns, name string) *DeploymentBuilder {
	return &DeploymentBuilder{
		d: appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
			},
		},
	}
}

func (b *DeploymentBuilder) WithName(name string) *DeploymentBuilder {
	b.d.ObjectMeta.Name = name
	return b
}

func (b *DeploymentBuilder) WithLabels(labels map[string]string) *DeploymentBuilder {
	b.d.ObjectMeta.Labels = labels
	return b
}
