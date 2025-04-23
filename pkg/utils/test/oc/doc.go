// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

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
