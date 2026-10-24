#!/usr/bin/env bash
# Abre la interfaz web de Locust en http://localhost:8089 (no inicia la prueba)
set -euo pipefail
cd "$(dirname "$0")/.."
GW=$(kubectl get gateway api-gateway -n mumnk8s -o jsonpath='{.status.addresses[0].value}')
rm -rf /tmp/cfg && mkdir -p /tmp/cfg
oras pull "${ZOT:?define ZOT}/locust-config:v1" -o /tmp/cfg >/dev/null \
  || { echo "AVISO: Zot no responde, uso la copia local de config.json"; cp locust/config.json /tmp/cfg/; }
source ~/locust-env/bin/activate
echo "Abre http://localhost:8089   (host: http://$GW)"
CONFIG_FILE=/tmp/cfg/config.json exec locust -f locust/locustfile.py --host "http://$GW" -u 200 -r 20
