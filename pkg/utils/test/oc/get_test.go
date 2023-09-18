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
