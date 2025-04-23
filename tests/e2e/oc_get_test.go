// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/test/oc"
)

func Test_GetNodes(t *testing.T) {
	cmd := oc.Get().Resource("nodes", "").OutputJsonpath("{.items[*].metadata.name}")
	res, err := cmd.Run()
	assert.NoError(t, err, "failed to get node names")
	nodes := strings.Split(res, " ")
	assert.NotZero(t, len(nodes))
}
