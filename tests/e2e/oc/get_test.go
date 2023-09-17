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
