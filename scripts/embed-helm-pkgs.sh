#!/bin/bash

set -e

PROJ_DIR="$(dirname $(dirname "${BASH_SOURCE[0]}"))"
MANIFESTS_DIR="$PROJ_DIR/manifests"
HELM_DIR="$MANIFESTS_DIR/helm"
GRAFANA_DSB_DIR="$MANIFESTS_DIR/grafana-dashboards"
STATIC_DIR="$PROJ_DIR/static"

rm -rf "$STATIC_DIR"

CHARTS=(
    manifests/helm/monitor-stack
    manifests/helm/monitor-stack-v2
    manifests/helm/minio
    manifests/helm/thanos
    manifests/helm/fluent-bit
    manifests/helm/aws-load-balancer-controller
    manifests/helm/aws-ebs-csi-driver
)
#readarray -d '' CHARTS < <(find "$HELM_DIR" -mindepth 1 -maxdepth 1 -type d -print0)

for chart in "${CHARTS[@]}"; do
    echo package "$chart"
    helm package "$chart" --destination "$STATIC_DIR"
done

# cp others to STATIC_DIR
cp -a "$GRAFANA_DSB_DIR"/* "$STATIC_DIR"
