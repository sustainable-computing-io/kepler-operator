// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package oc_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sustainable.computing.io/kepler-operator/tests/utils/oc"
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
