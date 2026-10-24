#!/usr/bin/env bash
# Uso: ./scripts/loadtest.sh <etiqueta> [usuarios] [spawn_rate] [duracion]
set -euo pipefail
LABEL="${1:?etiqueta, ej: a-go1-valkey1}"; USERS="${2:-200}"; RATE="${3:-20}"; DUR="${4:-4m}"
NS=mumnk8s
cd "$(dirname "$0")/.."
mkdir -p results
GW=$(kubectl get gateway api-gateway -n $NS -o jsonpath='{.status.addresses[0].value}')
rm -rf /tmp/cfg && mkdir -p /tmp/cfg && oras pull "${ZOT:?define ZOT}/locust-config:v1" -o /tmp/cfg >/dev/null

echo "Esperando que la cola de RabbitMQ esté vacía..."
for i in $(seq 1 60); do
  Q=$(kubectl exec -n $NS deploy/rabbitmq -- rabbitmqctl -q list_queues name messages --no-table-headers | awk '{print $2}')
  [ "${Q:-1}" = "0" ] && break
  sleep 5
done

MON="results/${LABEL}-monitor.txt"; : > "$MON"
(
  while true; do
    {
      echo "=== $(date -u +%T) ==="
      kubectl get hpa api-rust-hpa -n $NS --no-headers
      kubectl get pods -n $NS --no-headers | awk '{print $1, $2, $3}'
      kubectl top pods -n $NS --no-headers
      echo -n "cola: "; kubectl exec -n $NS deploy/rabbitmq -- rabbitmqctl -q list_queues name messages --no-table-headers | tr '\n' ' '; echo
    } >> "$MON" 2>&1
    sleep 10
  done
) &
MON_PID=$!
trap 'kill $MON_PID 2>/dev/null || true' EXIT

source ~/locust-env/bin/activate
CONFIG_FILE=/tmp/cfg/config.json locust -f locust/locustfile.py --host "http://$GW" \
  --headless -u "$USERS" -r "$RATE" -t "$DUR" --csv="results/$LABEL" --only-summary 2>&1 \
  | tail -n 30 | tee "results/${LABEL}-summary.txt" || true
deactivate
echo "Datos guardados en results/${LABEL}*"
