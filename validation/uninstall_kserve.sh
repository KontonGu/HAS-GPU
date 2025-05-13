#!/bin/bash
# uninstall_kserve.sh
# Script to uninstall KServe and its dependencies

set -e  # Exit immediately if a command exits with non-zero status

echo "=== Uninstalling KServe and its dependencies ==="

# Uninstall KServe
echo "Uninstalling KServe components..."
kubectl delete -f https://github.com/kserve/kserve/releases/download/v0.14.1/kserve.yaml --ignore-not-found=true
kubectl delete -f https://github.com/kserve/kserve/releases/download/v0.14.1/kserve-cluster-resources.yaml --ignore-not-found=true

# Uninstall Cert Manager
echo "Uninstalling Cert Manager..."
kubectl delete -f https://github.com/cert-manager/cert-manager/releases/download/v1.17.0/cert-manager.yaml --ignore-not-found=true

# Uninstall Istio and Knative
echo "Uninstalling Istio and Knative components..."
kubectl delete -f https://github.com/knative/net-istio/releases/download/knative-v1.17.0/net-istio.yaml --ignore-not-found=true
kubectl delete -f https://github.com/knative/serving/releases/download/knative-v1.17.0/serving-core.yaml --ignore-not-found=true
kubectl delete -f https://github.com/knative/serving/releases/download/knative-v1.17.0/serving-crds.yaml --ignore-not-found=true

echo "=== KServe uninstallation completed ==="
