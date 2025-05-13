
# 1. install knative with istio

# knative serving components
kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.17.0/serving-crds.yaml

kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.17.0/serving-core.yaml


# network layer knative Istio
istioctl install -y

# kubectl apply -l knative.dev/crd-install=true -f https://github.com/knative/net-istio/releases/download/knative-v1.17.0/istio.yaml
#kubectl apply -f https://github.com/knative/net-istio/releases/download/knative-v1.17.0/istio.yaml

kubectl apply -f https://github.com/knative/net-istio/releases/download/knative-v1.17.0/net-istio.yaml



# Fetch External IP or CNAME
#kubectl --namespace istio-system get service istio-ingressgateway

# Verify installation
kubectl get pods -n knative-serving

# Concigure DNS (no DNS here)
kubectl patch configmap/config-domain \
      --namespace knative-serving \
      --type merge \
      --patch '{"data":{"example.com":""}}'

#Magic DNS
#kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.17.0/serving-default-domain.yaml


# 3. Cert Manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.17.0/cert-manager.yaml


# Wait for cert-manager pods to be ready
kubectl wait --for=condition=ready pod -l app=cert-manager -n cert-manager --timeout=120s
kubectl wait --for=condition=ready pod -l app=webhook -n cert-manager --timeout=120s
kubectl wait --for=condition=ready pod -l app=cainjector -n cert-manager --timeout=120s

# 4. install Kserve
kubectl apply --server-side --force-conflicts -f https://github.com/kserve/kserve/releases/download/v0.14.1/kserve.yaml

kubectl apply --server-side --force-conflicts -f https://github.com/kserve/kserve/releases/download/v0.14.1/kserve-cluster-resources.yaml


# change default deployment mode and ingress option
#kubectl patch configmap/inferenceservice-config -n kserve --type=strategic -p '{"data": {"deploy": "{\"defaultDeploymentMode\": \"RawDeployment\"}"}}'

#kubectl delete validatingwebhookconfiguration servingruntime.modelmesh-webhook-server.default

# Autoscaler
#kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.17.0/serving-hpa.yaml



kubectl patch configmap config-features -n knative-serving --type merge -p '{"data":{"kubernetes.podspec-persistent-volume-claim":"enabled","kubernetes.podspec-persistent-volume-write":"enabled"}}'