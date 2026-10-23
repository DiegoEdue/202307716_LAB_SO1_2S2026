#!/usr/bin/env bash
# Uso: ./scripts/build-push.sh api-rust go-api go-grpc-server go-consumer
set -euo pipefail
ZOT="${ZOT:?define ZOT, ej: export ZOT=1.2.3.4.sslip.io}"
TAG="${TAG:-v1}"
cd "$(dirname "$0")/.."
for svc in "$@"; do
  docker build -t "$ZOT/$svc:$TAG" "./$svc"
  docker push "$ZOT/$svc:$TAG"
done
