#!/usr/bin/env bash
# Guarda en docs/evidencias/ la salida de los comandos que pide la guía de evaluación
NS=mumnk8s
cd "$(dirname "$0")/.."
OUT=docs/evidencias; mkdir -p $OUT
GW=$(kubectl get gateway api-gateway -n $NS -o jsonpath='{.status.addresses[0].value}')
printf '{"country":"ESP","warplanes_in_air":21,"warships_in_water":9,"timestamp":"2026-10-24T09:00:00Z"}' > /tmp/reporte.json
run(){ local f="$OUT/$1.txt"; shift; { echo "\$ $*"; bash -c "$*" 2>&1; echo; } > "$f"; echo "guardado $f"; }

run 01-cluster       "gcloud container clusters list; kubectl get nodes -o wide"
run 02-gateway       "kubectl get gateway,httproute -n $NS; curl -i -X POST http://$GW/grpc-202307716 -H "Content-Type: application/json" \-d '{"country":"ESP","warplanes_in_air":12,"warships_in_water":7,"timestamp":"2026-10-23T08:00:00Z"}'"
run 03-despliegue    "kubectl get deployments,hpa,pods,svc -n $NS -o wide"
run 04-logs-cadena   "kubectl logs -n $NS -l app=api-rust --tail=10 --prefix; kubectl logs -n $NS -l app=go-api -c grpc-client --tail=5 --prefix; kubectl logs -n $NS -l app=go-server -c grpc-server --tail=5 --prefix; kubectl logs -n $NS -l app=go-server -c writer --tail=5 --prefix"
run 05-rabbitmq      "kubectl exec -n $NS deploy/rabbitmq -- rabbitmqctl list_queues name messages consumers"
run 06-operator      "kubectl get crd | grep mumnk8s; kubectl -n mumn-operator-system get pods; kubectl get valkeyinstances,grafanainstances,statefulset,pvc,svc -n $NS"
run 07-valkey        "kubectl exec -n $NS valkey-0 -- valkey-cli DBSIZE; kubectl exec -n $NS valkey-0 -- valkey-cli TTL max:planes; kubectl exec -n $NS valkey-0 -- valkey-cli INFO persistence | grep aof_enabled; kubectl exec -n $NS valkey-0 -- valkey-cli INFO replication | head -n 4"
run 08-zot           "oras repo ls $ZOT; oras repo tags $ZOT/go-api; oras manifest fetch $ZOT/locust-config:v1"
run 09-imagenes      "kubectl get pods -A -o custom-columns=NS:.metadata.namespace,POD:.metadata.name,IMAGENES:.spec.containers[*].image | grep -E 'NS|mumnk8s|mumn-operator'"
run 10-hpa           "kubectl get hpa -n $NS; kubectl describe hpa api-rust-hpa -n $NS | head -n 30"
run 11-falco         "helm list -n falco; kubectl get ds,pods -n falco"
