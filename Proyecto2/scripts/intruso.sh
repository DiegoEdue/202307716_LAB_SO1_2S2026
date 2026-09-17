#!/bin/sh

echo "[intruso] Contenedor iniciado, comenzando simulacion de accesos sospechosos"

while true; do
    ts=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[intruso] $ts intentando leer /etc/shadow"
    cat /etc/shadow 2>&1 | head -n 1
    echo "[intruso] $ts intentando leer /etc/passwd"
    cat /etc/passwd 2>&1 | head -n 3
    echo "[intruso] $ts intentando leer /proc/1/environ"
    cat /proc/1/environ 2>&1
    echo ""
    sleep 5
done