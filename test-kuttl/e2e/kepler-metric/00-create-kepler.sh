#! /bin/bash

set -e -o pipefail

kubectl -n "$NAMESPACE" apply -f - <<EOF
---
apiVersion: kepler.system.sustainable.computing.io/v1alpha1
kind: Kepler
metadata:
  labels:
    app.kubernetes.io/name: kepler
    app.kubernetes.io/instance: kepler
    app.kubernetes.io/part-of: kepler-operator
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/created-by: kepler-operator
  name: kepler
  # namespace: kepler
spec:
  # TODO(user): Add fields here
  collector:
    image: quay.io/sustainable_computing_io/kepler:latest 
EOF

kubectl wait --for=condition=Ready pod --all -n "$NAMESPACE" --timeout 12m;
kubectl get pods -n "$NAMESPACE";
kubectl get svc -n "$NAMESPACE";

