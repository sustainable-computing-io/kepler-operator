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

// oc package is a DSL for running oc/kubectl command from within go programs.
// Each oc command is exposed as an interface, with methods to  collect command specific arguments.
// The collected arguments are passed to runner to execute the commad.
//
// A feature of this DSL is ability to compose oc commands.
// e.g. oc.Exec can be composed with oc.Get to fetch arguments for oc.Exec.
//
// The following oc.Exec command
//oc.Exec().
//  WithNamespace("openshift-logging").
//  WithPodGetter(oc.Get().
//	  WithNamespace("openshift-logging").
//	  Pod().
//	  Selector("component=elasticsearch").
//	  OutputJsonpath("{.items[0].metadata.name}")).
//  Container("elasticsearch").
//  WithCmd("es_util", " --query=\"_cat/aliases?v&bytes=m\"")
//
//  is equivalent to "oc -n openshift-logging exec $(oc -n openshift-logging get pod -l component=elasticsearch -o jsonpath={.items[0].metadata.name}) -c elasticsearch -- es_util --query=\"_cat/aliases?v&bytes=m\""
