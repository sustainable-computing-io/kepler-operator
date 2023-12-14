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

package e2e

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/test/oc"
)

const podSpecTemplate = `
apiVersion: v1
kind: Pod
metadata:
  name: log-generator
  labels:
    component: test
  namespace: %s
spec:
  containers:
    - name: log-generator
      image: quay.io/quay/busybox
      args: ["sh", "-c", "i=0; while true; do echo $i: Test message; i=$((i+1)) ; sleep 1; done"]
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop:
        - ALL
    seccompProfile:
      type: RuntimeDefault
`

func writePodSpec(t *testing.T, ns string) string {
	specfile := fmt.Sprintf("%s/podspec.yaml", os.TempDir())
	f, err := os.Create(specfile)
	assert.NoError(t, err, "fail to create temp file")
	podSpec := fmt.Sprintf(podSpecTemplate, ns)
	_, err = f.Write([]byte(podSpec))
	assert.NoError(t, err, "failed to write to temp file")
	return specfile
}

func getRandomNsName(t *testing.T) string {
	rand.Seed(time.Now().UnixNano())
	min := 0
	max := 99999
	num := rand.Intn(max-min+1) + min
	return fmt.Sprintf("e2e-oc-test-%05d\n", num)
}

func Test_Literal_Run(t *testing.T) {
	nsname := getRandomNsName(t)
	podspec := writePodSpec(t, nsname)

	var err error

	err = oc.Literal().From("oc create ns %s", nsname).Output()
	assert.NoError(t, err, "cannot create namespace")

	err = oc.Literal().From("oc label ns %s e2e-test-ns=true", nsname).Output()
	assert.NoError(t, err, "could not label ns")

	err = oc.Literal().From("oc apply -f %s", podspec).Output()
	assert.NoError(t, err, "could not apply podspec")

	podname, err := oc.Literal().From("oc -n %s get pod -l component=test -o jsonpath={.items[0].metadata.name}", nsname).Run()
	assert.Equal(t, podname, "log-generator")
	assert.NoError(t, err, "could not get pod")

	err = oc.Literal().From("oc -n %s wait --for=condition=Ready pod/log-generator", nsname).Output()
	assert.NoError(t, err, "could not wait for pod to get ready")

	out, err := oc.Literal().From("oc -n %s logs log-generator -f", nsname).RunFor(time.Second * 2)
	assert.NoError(t, err, "could not get pod logs for 2 secs")
	logs := strings.Split(out, "\n")
	assert.NotZero(t, len(logs))

	err = oc.Literal().From("oc -n %s exec log-generator -c log-generator -- ls -al", nsname).Output()
	assert.NoError(t, err, "could not exec into pod")

	err = oc.Literal().From("oc delete ns %s", nsname).Output()
	assert.NoError(t, err, "could not delete namespace")

	os.Remove(podspec)
}
