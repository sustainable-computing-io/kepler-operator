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

package oc

import (
	"fmt"
	"strings"
)

// Execer is interface for collecting arguments for Exec command
type AdmTopCommand interface {
	Command

	//ForContainers to retrieve `oc adm top` of containers
	ForContainers() AdmTopCommand

	//NoHeaders to not include noHeaders
	NoHeaders() AdmTopCommand
}

type admTop struct {
	*runner
	namespace  string
	podname    string
	containers bool
	noHeaders  bool
}

// AdmTop creates an 'oc adm top' command
func AdmTop(namespace, name string) AdmTopCommand {
	e := &admTop{
		runner:    &runner{},
		namespace: namespace,
		podname:   name,
	}
	e.collectArgsFunc = e.args
	return e
}

func (e *admTop) ForContainers() AdmTopCommand {
	e.containers = true
	return e
}

func (e *admTop) NoHeaders() AdmTopCommand {
	e.noHeaders = true
	return e
}

func (e *admTop) String() string {
	args := []string{}
	if e.containers {
		args = append(args, "--containers")
	}
	if e.noHeaders {
		args = append(args, "--no-headers")
	}
	return fmt.Sprintf("adm -n %s top pod %s %s", e.namespace, e.podname, strings.Join(args, " "))
}

// creates command args to be used by runner
func (e *admTop) args() []string {
	ocargs := sanitizeArgs(e.String())
	return ocargs
}
