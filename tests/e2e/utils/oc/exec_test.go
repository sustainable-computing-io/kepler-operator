// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package oc_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sustainable.computing.io/kepler-operator/tests/e2e/utils/oc"
)

func Test_Exec(t *testing.T) {
	tt := []struct {
		exec     oc.Execer
		occmd    string
		scenario string
	}{
		{
			exec: oc.Exec().
				WithNamespace("openshift-logging").
				Pod("mypod").
				Container("elasticsearch").
				WithCmd("indices"),
			occmd:    "oc -n openshift-logging exec mypod -c elasticsearch -- indices",
			scenario: "run a command without args",
		},
		{
			exec: oc.Exec().
				WithNamespace("openshift-logging").
				WithPodGetter(oc.Get().
					WithNamespace("openshift-logging").
					Pod().
					Selector("component=elasticsearch").
					OutputJsonpath("{.items[0].metadata.name}")).
				Container("elasticsearch").
				WithCmd("es_util", " --query=\"_cat/aliases?v&bytes=m\""),
			occmd:    "oc -n openshift-logging exec $(oc -n openshift-logging get pod -l component=elasticsearch -o jsonpath={.items[0].metadata.name}) -c elasticsearch -- es_util --query=\"_cat/aliases?v&bytes=m\"",
			scenario: "exec a command with args",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exec.String(), tc.occmd)
		})
	}
}
