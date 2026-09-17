#!/bin/bash

set -e

SERVICE_NAME="proyecto2-daemon"
SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

cd "$SCRIPT_DIR"

echo "--------------------------------"
echo "Deteniendo daemon..."
sudo systemctl stop "$SERVICE_NAME" && echo "Daemon detenido" || echo "[WARN] No se pudo detener el daemon"

echo "--------------------------------"
echo "Eliminando cronjob..."
crontab -l 2>/dev/null | grep -v "proyecto2-so1-cronjob" | crontab - 2>/dev/null || true
echo "Cronjob eliminado"

echo "--------------------------------"
echo "Eliminando contenedores gestionados del proyecto..."
docker ps -aq --filter label=perfil | xargs -r docker rm -f >/dev/null 2>&1 || true
echo "Contenedores gestionados eliminados"

echo "--------------------------------"
echo "Deteniendo docker-compose..."

cd "$SCRIPT_DIR/grafana-valkey"
docker compose down

cd "$SCRIPT_DIR"

echo "--------------------------------"
echo "Descargando modulo de kernel..."
bash scripts/descargar_modulo.sh

echo "--------------------------------"
echo "Sistema detenido correctamente"