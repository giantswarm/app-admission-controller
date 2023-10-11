#!/bin/bash

apptestctl bootstrap --kubeconfig="$(kind get kubeconfig)" --install-operators=true

# see "sigs.k8s.io/cluster-api" in go.mod
CAPI_VERSION="v1.0.4"

KUBE_CONFIG=$(kind get kubeconfig) kubectl apply -f "https://raw.githubusercontent.com/kubernetes-sigs/cluster-api/${CAPI_VERSION}/config/crd/bases/cluster.x-k8s.io_clusters.yaml"
