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

func Test_Literal(t *testing.T) {
	tt := []struct {
		literal  oc.ILiteral
		occmd    string
		scenario string
	}{
		{
			literal:  oc.Literal().From(" oc  apply    -f   ./tmp/podspec.yaml "),
			occmd:    "oc apply -f ./tmp/podspec.yaml",
			scenario: "tolerate spaces",
		},
		{
			literal:  oc.Literal().From("oc label node %s %s", "worker-node", "key1=val1 key2=val2"),
			occmd:    "oc label node worker-node key1=val1 key2=val2",
			scenario: "add labels to a node",
		},
		{
			literal:  oc.Literal().From("oc   "),
			occmd:    "command too small",
			scenario: "fetch pods from namespace with labels",
		},
		{
			literal:  oc.Literal().From("abc xyz"),
			occmd:    "error: command string must start with 'oc'",
			scenario: "",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.literal.String(), tc.occmd)
		})
	}
}
