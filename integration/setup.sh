#!/bin/bash
#
# Bootstraps the app platform in the kind cluster used by the integration
# tests. This replaces `apptestctl bootstrap --install-operators=true` and
# reproduces what apptestctl v0.26.0 does in cmd/bootstrap/runner.go:
#
#   1. install the app platform CRDs and wait for them to be established
#   2. create the `giantswarm-critical` priority class
#   3. create the `giantswarm` namespace
#   4. install app-operator and chart-operator with helm
#   5. create the in-cluster `chartmuseum` catalog CR
#   6. create the chartmuseum network policy
#   7. install chartmuseum as an app CR and wait for it to be deployed
#
# The PSP RBAC apptestctl creates is left out, PodSecurityPolicy was removed in
# Kubernetes 1.25 and the tests run on a much newer kind node.
#
# Requires: kind, kubectl, helm, curl. All of them are installed by the
# architect `integration-test` CircleCI job.

set -euo pipefail

# Versions pinned by apptestctl v0.26.0 (cmd/bootstrap/runner.go).
APP_OPERATOR_VERSION="6.7.0"
CHART_OPERATOR_VERSION="2.35.0"
CHARTMUSEUM_VERSION="3.9.3"

CHARTMUSEUM_CATALOG_HELM_INDEX_URL="https://chartmuseum.github.io/charts"
CHARTMUSEUM_CATALOG_NAME="apptestctl-chartmuseum"
CHARTMUSEUM_CATALOG_STORAGE_URL="http://chartmuseum:8080/"
CHARTMUSEUM_NAME="chartmuseum"
CONTROL_PLANE_CATALOG_STORAGE_URL="https://giantswarm.github.io/control-plane-catalog/"
NAMESPACE="giantswarm"

# App platform CRDs. Version matches github.com/giantswarm/apiextensions-application
# in go.mod. apptestctl additionally installs CRDs it does not own (VPA,
# prometheus-operator, kyverno, cilium, gateway-api) so that the optional
# resources of the operator charts get rendered. They are all guarded by
# `.Capabilities.APIVersions.Has` checks, so they are not needed here.
APIEXTENSIONS_APPLICATION_VERSION="v0.6.2"
CRD_GROUP="application.giantswarm.io"
CRD_VERSION="v1alpha1"
CRD_PLURALS=(appcatalogentries appcatalogs apps catalogs charts)

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

KUBECONFIG="${TMP_DIR}/kubeconfig"
kind get kubeconfig > "${KUBECONFIG}"
export KUBECONFIG

# retry runs the given command until it succeeds or the timeout expires.
retry() {
  local description="$1"
  local timeout="$2"
  shift 2

  local deadline=$((SECONDS + timeout))

  until "$@"; do
    if [[ ${SECONDS} -ge ${deadline} ]]; then
      echo "timed out after ${timeout}s waiting for ${description}"
      return 1
    fi
    echo "waiting for ${description}"
    sleep 5
  done
}

ensure_crds() {
  echo "installing app platform CRDs"

  local plural
  for plural in "${CRD_PLURALS[@]}"; do
    kubectl apply -f "https://raw.githubusercontent.com/giantswarm/apiextensions-application/${APIEXTENSIONS_APPLICATION_VERSION}/config/crd/${CRD_GROUP}_${plural}.yaml"
  done

  # Wait until every CRD is established and served in API discovery. Installing
  # the operator charts straight after creating the CRDs races the API server's
  # discovery refresh and makes helm fail with "resource mapping not found".
  for plural in "${CRD_PLURALS[@]}"; do
    kubectl wait --for=condition=Established --timeout=120s "crd/${plural}.${CRD_GROUP}"

    retry "${plural} to appear in API discovery" 120 \
      bash -c "kubectl get --raw '/apis/${CRD_GROUP}/${CRD_VERSION}' | grep -q '\"name\":\"${plural}\"'"
  done
}

ensure_priority_class() {
  echo "creating priorityclass giantswarm-critical"

  kubectl apply -f - <<EOF
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: giantswarm-critical
value: 1000000000
globalDefault: false
description: "This priority class is used by giantswarm kubernetes components."
EOF
}

ensure_namespace() {
  echo "creating namespace ${NAMESPACE}"

  kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: ${NAMESPACE}
EOF

  retry "namespace ${NAMESPACE} to become active" 60 \
    bash -c "[[ \"\$(kubectl get namespace ${NAMESPACE} -o jsonpath='{.status.phase}')\" == 'Active' ]]"
}

install_operator() {
  local name="$1"
  local version="$2"

  if helm status "${name}" --namespace "${NAMESPACE}" > /dev/null 2>&1; then
    echo "${name} already installed"
    return
  fi

  echo "installing ${name} ${version}"

  # The release name has to match the chart name, this is how app-operator
  # decides it is the unique instance reconciling app CRs labelled with
  # app-operator.giantswarm.io/version: 0.0.0.
  #
  # isManagementCluster is set to true so chart-operator uses ClusterFirst DNS
  # settings.
  cat > "${TMP_DIR}/operator-values.yaml" <<EOF
isManagementCluster: "true"
operatorkit:
  resyncPeriod: "20s"

provider:
  kind: "aws"

verticalPodAutoscaler:
  enabled: false
EOF

  helm install "${name}" "${name}" \
    --repo "${CONTROL_PLANE_CATALOG_STORAGE_URL}" \
    --version "${version}" \
    --namespace "${NAMESPACE}" \
    --values "${TMP_DIR}/operator-values.yaml"
}

install_operators() {
  install_operator "app-operator" "${APP_OPERATOR_VERSION}"
  install_operator "chart-operator" "${CHART_OPERATOR_VERSION}"
}

# create_catalog creates a catalog CR in the default namespace. The optional
# third argument is the value of the app-operator.giantswarm.io/version label.
create_catalog() {
  local name="$1"
  local url="$2"
  local app_operator_version="${3:-}"

  echo "creating ${name} catalog cr"

  local labels=""
  if [[ -n "${app_operator_version}" ]]; then
    labels=$(printf '\n  labels:\n    app-operator.giantswarm.io/version: "%s"' "${app_operator_version}")
  fi

  kubectl apply -f - <<EOF
apiVersion: ${CRD_GROUP}/${CRD_VERSION}
kind: Catalog
metadata:
  name: ${name}
  namespace: default${labels}
spec:
  description: ${name}
  title: ${name}
  logoURL: ""
  storage:
    type: helm
    URL: ${url}
  repositories:
  - type: helm
    URL: ${url}
EOF
}

install_catalogs() {
  create_catalog "${CHARTMUSEUM_NAME}" "${CHARTMUSEUM_CATALOG_STORAGE_URL}"
}

ensure_chartmuseum_resources() {
  echo "ensuring additional chartmuseum resources"

  kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: chartmuseum
  namespace: ${NAMESPACE}
spec:
  podSelector:
    matchLabels:
      app: chartmuseum
      release: chartmuseum
  policyTypes:
  - Ingress
  - Egress
  egress: []
  ingress:
  - ports:
    - protocol: TCP
      port: 8080
EOF
}

install_chartmuseum() {
  create_catalog "${CHARTMUSEUM_CATALOG_NAME}" "${CHARTMUSEUM_CATALOG_HELM_INDEX_URL}" "0.0.0"

  echo "creating ${CHARTMUSEUM_NAME} user values configmap"

  kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: ${CHARTMUSEUM_NAME}-user-values
  namespace: ${NAMESPACE}
data:
  values: |
    persistence:
      enabled: "true"
    serviceAccount:
      name: "chartmuseum"
      create: "true"
    env:
      open:
        ALLOW_OVERWRITE: true
        DISABLE_API: false
    probes:
      readiness:
        initialDelaySeconds: 10
EOF

  echo "creating ${CHARTMUSEUM_NAME} app cr"

  kubectl apply -f - <<EOF
apiVersion: ${CRD_GROUP}/${CRD_VERSION}
kind: App
metadata:
  name: ${CHARTMUSEUM_NAME}
  namespace: ${NAMESPACE}
  labels:
    app.kubernetes.io/name: ${CHARTMUSEUM_NAME}
    app-operator.giantswarm.io/version: "0.0.0"
spec:
  catalog: ${CHARTMUSEUM_CATALOG_NAME}
  name: ${CHARTMUSEUM_NAME}
  namespace: ${NAMESPACE}
  version: ${CHARTMUSEUM_VERSION}
  kubeConfig:
    inCluster: true
  userConfig:
    configMap:
      name: ${CHARTMUSEUM_NAME}-user-values
      namespace: ${NAMESPACE}
EOF

  retry "app cr ${NAMESPACE}/${CHARTMUSEUM_NAME} to be deployed" 1200 \
    bash -c "[[ \"\$(kubectl -n ${NAMESPACE} get app ${CHARTMUSEUM_NAME} -o jsonpath='{.status.release.status}')\" == 'deployed' ]]"
}

wait_for_chartmuseum() {
  echo "waiting for ready ${CHARTMUSEUM_NAME} deployment"

  retry "deployment ${NAMESPACE}/${CHARTMUSEUM_NAME} to be ready" 300 \
    kubectl rollout status "deployment/${CHARTMUSEUM_NAME}" --namespace "${NAMESPACE}" --timeout=60s
}

echo "bootstrapping app platform components"

ensure_crds
ensure_priority_class
ensure_namespace
install_operators
install_catalogs
ensure_chartmuseum_resources
install_chartmuseum
wait_for_chartmuseum

echo "app platform components are ready"

# see "sigs.k8s.io/cluster-api" in go.mod
CAPI_VERSION="v1.0.4"

kubectl apply -f "https://raw.githubusercontent.com/kubernetes-sigs/cluster-api/${CAPI_VERSION}/config/crd/bases/cluster.x-k8s.io_clusters.yaml"
