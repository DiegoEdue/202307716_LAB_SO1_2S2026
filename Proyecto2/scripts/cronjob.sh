#!/bin/bash
# Pensado para ser invocado por crontab (por eso usa rutas absolutas y
# hace logging a archivo: cron no muestra stdout naen ningu terminal).
# El Daemon en Go es quien registra/desregistra este cronjob (cada 1 min).

set -uo pipefail   # sin -e: si un docker run falla, seguimos con los demas

DOCKER="/usr/bin/docker"
LOGFILE="/var/log/proyecto2_deploy.log"
IMAGE_INTRUSO="intruso-202307716:latest"   # debe coincidir con el tag del `docker build`

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" >> "$LOGFILE"
}

run_bajo() {
    "$DOCKER" run -d --label perfil=bajo alpine sleep 240 >> "$LOGFILE" 2>&1
}

run_alto_ram() {
    "$DOCKER" run -d --label perfil=alto roldyoran/go-client >> "$LOGFILE" 2>&1
}

run_alto_cpu() {
    "$DOCKER" run -d --label perfil=alto alpine sh -c \
        "while true; do echo '2^20' | bc > /dev/null; sleep 2; done" >> "$LOGFILE" 2>&1
}

run_intruso() {
    "$DOCKER" run -d --label perfil=intruso "$IMAGE_INTRUSO" >> "$LOGFILE" 2>&1
}

FUNCS=(run_bajo run_alto_ram run_alto_cpu run_intruso)

log "Desplegando 5 contenedores..."
for i in $(seq 1 5); do
    idx=$((RANDOM % ${#FUNCS[@]}))
    ${FUNCS[$idx]}
    if [ $? -ne 0 ]; then
        log "fallo al desplegar con ${FUNCS[$idx]}"
    fi
done
log "Despliegue completado."