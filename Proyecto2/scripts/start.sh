#!/bin/bash

set -e

SERVICE_NAME="proyecto2-daemon"
SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CARNET="202307716"

cd "$SCRIPT_DIR"

echo "-----[1] Imagen intrusa-----"
if ! docker image inspect "intruso-${CARNET}:latest" >/dev/null 2>&1; then
    echo "[INFO] Construyendo intruso-${CARNET}:latest..."
    docker build -t "intruso-${CARNET}:latest" scripts/
else
    echo "[INFO] intruso-${CARNET}:latest ya existe, se omite el build."
fi

echo "-----[2] Sonda eBPF-----"
if [ ! -f daemon/ebpfkill_bpfel.go ]; then
    echo "[INFO] Primera vez: instalando dependencias y generando bindings..."
    command -v clang >/dev/null 2>&1 || sudo apt-get install -y clang libbpf-dev "linux-headers-$(uname -r)"
    # clang y llvm son paquetes separados: revisamos llvm-strip aparte,
    # porque clang puede existir sin que llvm-strip este instalado.
    if ! command -v llvm-strip >/dev/null 2>&1; then
        sudo apt-get install -y llvm
        if ! command -v llvm-strip >/dev/null 2>&1; then
            strip_bin=$(ls /usr/bin/llvm-strip-* 2>/dev/null | head -n1)
            [ -n "$strip_bin" ] && sudo ln -s "$strip_bin" /usr/bin/llvm-strip
        fi
    fi
    command -v bpftool >/dev/null 2>&1 || sudo apt-get install -y bpftool
    [ -f daemon/vmlinux.h ] || bpftool btf dump file /sys/kernel/btf/vmlinux format c > daemon/vmlinux.h
    (cd daemon && go mod tidy && go generate ./...)
else
    echo "[INFO] Bindings de eBPF ya generados, se omite."
fi

echo "-----[3] Compilando daemon-----"
if ! (cd daemon && go build -o daemon-bin .); then
    echo "[ERROR] No se pudo compilar el daemon"
    exit 1
fi
echo "Daemon compilado correctamente"

echo "-----[4] Grafana y Valkey-----"
cd "$SCRIPT_DIR/grafana-valkey"

if docker ps -a --format '{{.Names}}' | grep -qx "valkey-db"; then
    echo "[INFO] Eliminando contenedor previo valkey-db para evitar conflicto..."
    docker rm -f valkey-db >/dev/null
fi

docker compose up -d
cd "$SCRIPT_DIR"

echo "-----[5] Servicio systemd-----"
if [ ! -f "/etc/systemd/system/${SERVICE_NAME}.service" ]; then
    echo "[INFO] Instalando unidad systemd (primera vez)..."
    if [ ! -f daemon/proyecto2-daemon.service.template ]; then
        echo "[ERROR] Falta daemon/proyecto2-daemon.service.template"
        exit 1
    fi
    sed "s#__WORKDIR__#${SCRIPT_DIR}/daemon#g" daemon/proyecto2-daemon.service.template \
        | sudo tee "/etc/systemd/system/${SERVICE_NAME}.service" >/dev/null
    sudo systemctl daemon-reload
    sudo systemctl enable "$SERVICE_NAME"
fi

sudo systemctl restart "$SERVICE_NAME"

echo "Sistema iniciado correctamente"
echo "Logs del daemon: sudo journalctl -u ${SERVICE_NAME} -f"
echo "Grafana disponible en: http://localhost:3001"