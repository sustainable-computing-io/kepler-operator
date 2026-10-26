// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGVKName(t *testing.T) {
	tt := []struct {
		name string
		obj  client.Object
		want string
	}{
		{
			name: "namespaced object includes the namespace",
			obj: &appsv1.DaemonSet{
				TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "power-monitor", Name: "kepler"},
			},
			want: "power-monitor/kepler (apps/v1, Kind=DaemonSet)",
		},
		{
			name: "cluster-scoped object omits the namespace",
			obj: &rbacv1.ClusterRole{
				TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "ClusterRole"},
				ObjectMeta: metav1.ObjectMeta{Name: "my-role"},
			},
			want: "my-role (rbac.authorization.k8s.io/v1, Kind=ClusterRole)",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, GVKName(tc.obj))
		})
	}
}

func TestStringMapMerge(t *testing.T) {
	tt := []struct {
		name  string
		base  StringMap
		other StringMap
		want  StringMap
	}{
		{
			name:  "merges disjoint keys",
			base:  StringMap{"a": "1"},
			other: StringMap{"b": "2"},
			want:  StringMap{"a": "1", "b": "2"},
		},
		{
			name:  "other overrides base on conflicting keys",
			base:  StringMap{"a": "1", "b": "2"},
			other: StringMap{"b": "3"},
			want:  StringMap{"a": "1", "b": "3"},
		},
		{
			name:  "merging an empty map returns the base entries",
			base:  StringMap{"a": "1"},
			other: StringMap{},
			want:  StringMap{"a": "1"},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.base.Merge(tc.other))
		})
	}
}

func TestStringMapAddIfNotEmpty(t *testing.T) {
	tt := []struct {
		name string
		key  string
		val  string
		want StringMap
	}{
		{name: "adds when key and value are set", key: "a", val: "1", want: StringMap{"a": "1"}},
		{name: "skips an empty key", key: "", val: "1", want: StringMap{}},
		{name: "skips an empty value", key: "a", val: "", want: StringMap{}},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, StringMap{}.AddIfNotEmpty(tc.key, tc.val))
		})
	}
}
