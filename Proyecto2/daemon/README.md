# Daemon y eBPF

Daemon gestionador de contenedores. Corre como servicio systemd (`proyecto2-daemon`) para que se reinicie solo si falla y quede disponible sin necesidad de una terminal abierta.

## 1. Daemon

### 1.1. Requisitos previos

```bash
sudo apt update
sudo apt install golang-go -y
```

También necesita el módulo de kernel cargado [modulo](../modulo/README.md) y Docker con soporte para `docker compose`.

### 1.2. Arranque normal

Para evitar la compilación a mano, se puede ejecutar el [script](../scripts/start.sh) desde la raíz del proyecto. Este script:

1. Compila `main.go` → `daemon/daemon-bin`.
2. Levanta Grafana y Valkey (`docker compose up -d`).
3. La primera vez, instala `proyecto2-daemon.service.template` en `/etc/systemd/system/proyecto2-daemon.service`, sustituyendo `__WORKDIR__` por la ruta real de `daemon/` en tu máquina.
4. Arranca o reinicia el servicio con `systemctl`.

```bash
./scripts/start.sh
```

### 1.3. Ver logs

Al correr como servicio systemd, los logs van al journal, no a un archivo:

```bash
sudo journalctl -u proyecto2-daemon -f
```

### 1.4. Detener todo

```bash
./scripts/stop.sh
```

Este script detiene el servicio, limpia el cronjob, elimina los contenedores gestionados (los que llevan el label `perfil`) y baja Grafana/Valkey.

### 1.5. Reinstalar el servicio desde cero

Si cambias la ruta del repo o algo queda mal configurado:

```bash
sudo systemctl disable proyecto2-daemon
sudo rm /etc/systemd/system/proyecto2-daemon.service
sudo systemctl daemon-reload
./scripts/start.sh
```

El script start.sh se encarga de reinstlarlo limpio.

## 2. eBPF

### 2.1 Sonda eBPF

```bash
cd daemon

bpftool btf dump file /sys/kernel/btf/vmlinux format c > vmlinux.h

go mod tidy
go generate ./...
```

- `vmlinux.h` se encarga de describir las estructuras del kernel.
- `go mod tidy` descarga cilium/ebpf y go-redis, genera go.sum.
- `go generate ./..` compila ebpf_kill.c y genera ebpfkill_bpfel.go / .o

### 2.3 Compilar el Daemon

```bash
go build -o daemon-bin .
```
