// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package oc_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sustainable.computing.io/kepler-operator/tests/utils/oc"
)

func Test_Get(t *testing.T) {

	tt := []struct {
		getter   oc.Getter
		occmd    string
		scenario string
	}{
		{
			getter:   oc.Get().Resource("nodes", "").OutputJsonpath("{.items[*].metadata.name}"),
			occmd:    "oc get nodes -o jsonpath={.items[*].metadata.name}",
			scenario: "get all node names",
		},
		{
			getter: oc.Get().
				Pod().
				WithNamespace("test-log-gen").
				Selector("component=test").
				OutputJsonpath("{.items[0].metadata.name}"),
			occmd:    "oc -n test-log-gen get pod -l component=test -o jsonpath={.items[0].metadata.name}",
			scenario: "fetch pods from namespace with labels",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.getter.String(), tc.occmd)
		})
	}
}
