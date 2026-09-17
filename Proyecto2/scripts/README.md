# Guía de ejecución

## Prerrequisitos

Tener instalado Go y docker

### 1. Actualizar repositorios del sistema

```bash
sudo apt update
```

### 2. Instalar Go

El compilador de Go es necesario para compilar el código fuente del daemon.

```bash
sudo apt install -y golang
```

### 3. Instalar Docker y Docker Compose

```bash
sudo apt install -y docker.io docker-compose-v2
```

### 4. Instalar Cron

El servicio de tareas programadas `cron` es obligatorio para la creación automática de contenedores.

```bash
sudo apt install -y cron
sudo systemctl enable cron
sudo systemctl start cron
```

### 5. Configurar permisos de Docker (Importante)

Si requieres usar `sudo` para ejecutar comandos de Docker (ej. `sudo docker ps`), el sistema fallará al crear contenedores de forma automática. El script programa un *cronjob* a nombre del usuario actual, por lo que cuando el cron intenta ejecutar `docker run` en segundo plano, el sistema denegará el acceso por falta de permisos.

Para solucionar este problema, es obligatorio agregar tu usuario al grupo de Docker:

```bash
sudo usermod -aG docker $USER
```

Para aplicar los cambios en el grupo sin reiniciar la terminal, ejecuta:

```bash
newgrp docker
```

## Ejecución

### 1. Permisos de ejecución

Asegurar que los scripts de control tengan los permisos necesarios para ser ejecutados.

```bash
chmod +x start.sh stop.sh cronjob.sh cargar_modulo.sh descargar_modulo.sh
```

### 2. Iniciar el sistema

El script hace lo siguiente:

1. Compila `main.go` → `daemon/daemon-bin`.
2. Levanta Grafana y Valkey (`docker compose up -d`).
3. La primera vez, instala `proyecto2-daemon.service.template` en `/etc/systemd/system/proyecto2-daemon.service`, sustituyendo `__WORKDIR__` por la ruta real de `daemon/` en tu máquina.
4. Arranca o reinicia el servicio con `systemctl`.

```bash
./start.sh
```

Una vez finalizado, la interfaz de Grafana estará disponible en el puerto 3001. Para acceder, ingresar desde el navegador web: `http://localhost:3001`

### 3. Verificar el Cronjob

Para confirmar que la tarea programada se ha registrado exitosamente en tu usuario, puedes listar las tareas actuales ejecutando:

```bash
crontab -l
```

Deberías ver una instrucción indicando la ejecución de `daemon-bin crear-contenedor`.

### 4. Detener el sistema

Este script detiene el servicio, limpia el cronjob, elimina los contenedores gestionados (los que llevan el label `perfil`) y baja Grafana/Valkey.

```bash
./stop.sh
```
