#!/usr/bin/env bash
# Copia las imagenes de terceros a Zot para que el cluster descargue todo de ahi
set -euo pipefail
ZOT="${ZOT:?define ZOT}"
mirror() { docker pull "$1" && docker tag "$1" "$ZOT/$2" && docker push "$ZOT/$2"; }
mirror rabbitmq:3-management rabbitmq:3-management
mirror valkey/valkey:8 valkey:8
mirror grafana/grafana:11.4.0 grafana:11.4.0
