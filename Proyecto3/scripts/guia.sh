#!/usr/bin/env bash

NS=mumnk8s
separador=$(printf '=%.0s' {1..20})
# Parte principal
gcloud container clusters list

# 1.1
echo "${separador} 1.1 Arquitectura general, despliegue (Gateway API) ${separador}"
kubectl get nodes -o wide
kubectl get gateway,httproute -n $NS
curl -i -X POST http://$GW/grpc-202307716 -H "Content-Type: application/json" \
-d '{"country":"ESP","warplanes_in_air":12,"warships_in_water":7,"timestamp":
"2026-10-23T08:00:00Z"}'

#1.2
echo "${separador} 1.2 API REST en Rust: HPA y deployments ${separador}"
kubectl get deployments,hpa -n $NS -o wide

#1.3
echo "${separador} 1.3 Servicios Go: Rust -> gRPC client -> gRPC server -> writer ${separador}"
kubectl logs -n $NS -l app=api-rust --tail=10 --prefix; kubectl logs -n $NS -l app=go-api -c grpc-client --tail=5 --prefix; kubectl logs -n $NS -l app=go-server -c grpc-server --tail=5 --prefix; kubectl logs -n $NS -l app=go-server -c writer --tail=5 --prefix

#1.4
echo "${separador} 1.4 Zot (VM fuera del cluster): ver la pestana del navegador ${separador}"

#1.5
echo "${separador} 1.5 RabbitMQ y consumidor ${separador}"
kubectl get pods -n $NS
kubectl exec -n $NS deploy/rabbitmq -- rabbitmqctl list_queues name messages consumers

#1.6
echo "${separador} 1.6 Valkey: Operator, CR, PVC y persistencia ${separador}"
kubectl get valkeyinstances,grafanainstances,statefulset,pvc,svc -n $NS
kubectl exec -n $NS valkey-0 -- valkey-cli DBSIZE

#1.7
echo "${separador} 1.7 Grafana (desplegado por el Operator): ver la pestana del navegador ${separador}"
kubectl get svc grafana -n $NS

#1.8
echo "${separador} 1.8 Falco: alerta en tiempo real ${separador}"
helm list -n falco; kubectl get ds,pods -n falco
echo "kubectl -n falco logs -f -l app.kubernetes.io/name=falco -c falco --max-log-requests 5 --prefix --tail=0"
echo "kubectl exec -n mumnk8s deploy/go-api-deploy -c api-rest -- sh -c 'mkdir -p /app && echo clave > /app/secretos && cat /app/secretos'"


cd ~/202307716_LAB_SO1_2S2026/Proyecto3
GW=$(kubectl get gateway api-gateway -n mumnk8s -o jsonpath='{.status.addresses[0].value}')
rm -rf /tmp/cfg && mkdir -p /tmp/cfg
oras pull $ZOT/locust-config:v1 -o /tmp/cfg
source ~/locust-env/bin/activate
CONFIG_FILE=/tmp/cfg/config.json locust -f locust/locustfile.py --host http://$GW --headless -u 20 -r 5 -t 60s 2>&1 | tail -n 8
deactivate
