#!/bin/bash

server=$(kubectl config view --minify --output jsonpath='{.clusters[*].cluster.server}')
name=$(kubectl get secrets --namespace=default -o json | jq -r '.items[] | select(.metadata.name | test("cloudpods-reader-token-")).metadata.name')
ca=$(kubectl get secret/$name --namespace=default -o jsonpath='{.data.ca\.crt}')
token=$(kubectl get secret/$name --namespace=default -o jsonpath='{.data.token}' | base64 --decode)
namespace=$(kubectl get secret/$name --namespace=default -o jsonpath='{.data.namespace}' | base64 --decode)

echo "
apiVersion: v1
kind: Config
clusters:
- name: default-cluster
  cluster:
    certificate-authority-data: ${ca}
    server: ${server}
contexts:
- name: default-context
  context:
    cluster: default-cluster
    namespace: default
    user: default-user
current-context: default-context
users:
- name: default-user
  user:
    token: ${token}
"
