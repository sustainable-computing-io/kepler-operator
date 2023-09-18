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

package oc_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/test/oc"
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
