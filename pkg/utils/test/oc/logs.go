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

// Logs is interface for collecting arguments for Logs command
type LogsCommand interface {
	Command

	// argument for option -n
	WithNamespace(string) LogsCommand

	// argument for podname
	WithPod(string) LogsCommand

	// argument for option -c
	WithContainer(string) LogsCommand
}

type logConfig struct {
	*runner
	namespace string

	podname   string
	container string
}

// Exec creates an 'oc exec' command
func Logs() LogsCommand {
	e := &logConfig{
		runner: &runner{},
	}
	e.collectArgsFunc = e.args
	return e
}

func (e *logConfig) WithNamespace(namespace string) LogsCommand {
	e.namespace = namespace
	return e
}

func (e *logConfig) WithPod(podname string) LogsCommand {
	e.podname = podname
	return e
}

func (e *logConfig) WithContainer(container string) LogsCommand {
	e.container = strings.ToLower(container)
	return e
}

// creates command args to be used by runner
func (e *logConfig) args() []string {
	namespaceStr := ""
	if e.namespace != "" {
		namespaceStr = fmt.Sprintf("-n %s", e.namespace)
	}
	containerStr := ""
	if e.container != "" {
		containerStr = fmt.Sprintf("-c %s", e.container)
	}
	ocargs := sanitizeArgs(fmt.Sprintf("%s logs %s %s", namespaceStr, e.podname, containerStr))
	return ocargs
}
