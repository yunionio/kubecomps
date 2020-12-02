#!/bin/bash

PROJ_DIR="$(dirname $(dirname "${BASH_SOURCE[0]}"))"
MANIFESTS_DIR="$PROJ_DIR/manifests"
HELM_DIR="$MANIFESTS_DIR/helm"
STATIC_DIR="$PROJ_DIR/static"

rm -rf "$STATIC_DIR"

CHARTS=()
readarray -d '' CHARTS < <(find "$HELM_DIR" -mindepth 1 -maxdepth 1 -type d -print0)

for chart in "${CHARTS[@]}"; do
    echo package "$chart"
    helm package "$chart" --destination "$STATIC_DIR"
done
