#!/usr/bin/env bash
# Chequeo previo a la evaluación (envía 1 reporte de prueba)
NS=mumnk8s; FAIL=0
ok(){ echo "OK     $1"; }; bad(){ echo "FALLA  $1"; FAIL=1; }
CTX=$(kubectl config current-context)
[[ "$CTX" == *_mumn-diego ]] && ok "contexto: $CTX" || bad "contexto equivocado: $CTX"
N=$(kubectl get nodes --no-headers | grep -c ' Ready')
[ "$N" -ge 2 ] && ok "$N nodos Ready" || bad "solo $N nodos Ready"
P=$(kubectl get pods -n $NS --no-headers | awk '{split($2,a,"/"); if ($3!="Running" || a[1]!=a[2]) print $1" "$2" "$3}')
[ -z "$P" ] && ok "todos los pods Ready en $NS" || { bad "pods con problema:"; echo "$P"; }
O=$(kubectl get pods -n mumn-operator-system --no-headers | grep -c ' 1/1 .*Running')
[ "$O" -ge 1 ] && ok "operator Running" || bad "operator no está Running"
GW=$(kubectl get gateway api-gateway -n $NS -o jsonpath='{.status.addresses[0].value}')
C=$(curl -s -o /dev/null -w '%{http_code}' -m 10 -X POST http://$GW/grpc-202307716 -H 'Content-Type: application/json' -d '{"country":"ESP","warplanes_in_air":1,"warships_in_water":1,"timestamp":"2026-10-24T08:00:00Z"}')
[ "$C" = "200" ] && ok "Gateway $GW responde 200" || bad "Gateway devolvió $C"
GI=$(kubectl get svc grafana -n $NS -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
curl -s -m 10 http://$GI/api/health | grep -q '"database": "ok"' && ok "Grafana $GI sano" || bad "Grafana no responde en $GI"
D=$(kubectl exec -n $NS valkey-0 -- valkey-cli DBSIZE 2>/dev/null)
[ -n "$D" ] && ok "Valkey DBSIZE: $D" || bad "Valkey no responde"
Q=$(kubectl exec -n $NS deploy/rabbitmq -- rabbitmqctl -q list_queues name messages --no-table-headers 2>/dev/null | awk '{print $2}')
ok "cola reports: ${Q:-?} mensajes"
R=$(oras repo ls "${ZOT:?define ZOT}" 2>/dev/null | wc -l)
[ "$R" -ge 9 ] && ok "Zot: $R repositorios" || bad "Zot: solo $R repositorios"
EXT=$(kubectl get pods -n $NS -o jsonpath='{range .items[*]}{range .spec.containers[*]}{.image}{"\n"}{end}{end}' | grep -vc "^$ZOT/")
[ "$EXT" = 0 ] && ok "todas las imágenes de $NS salen de Zot" || bad "$EXT contenedores con imagen que no es de Zot"
kubectl get hpa -n $NS --no-headers
exit $FAIL
F=$(kubectl get ds falco -n falco -o jsonpath='{.status.numberReady}/{.status.desiredNumberScheduled}' 2>/dev/null)
[ -n "$F" ] && [ "${F%/*}" = "${F#*/}" ] && ok "Falco listo ($F)" || bad "Falco no está listo ($F)"
