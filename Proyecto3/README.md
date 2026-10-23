# Proyecto 3 - Sistemas Operativos 1

## Descargar el codigo del repositorio (opcional)

Si se desea utilizar el codigo Go utilizado, entonces se debe de clonar el repositorio utilizando git

```bash
git clone git@github.com:DiegoEdue/202307716_LAB_SO1_2S2026.git
cd 202307716_LAB_SO1_2S2026
```

Para este proyecto se utilizo una VM, la ISO que se utilizo fue Ubuntu Server 24 LTS. La activación de SSH es opcional pero facilita su manejo.


## 1. Arquitectura 

```
Proyecto3/
├── api-rust/
│   ├── Cargo.toml
│   ├── Dockerfile
│   └── src/
│       └── main.rs
├── go-api/
│   ├── main.go
│   ├── Dockerfile
│   └── go.mod
├── go-consumer/
│   ├── main.go
│   ├── Dockerfile
│   └── go.mod
├── go-grpc-server/
│   ├── main.go
│   ├── Dockerfile
│   └── go.mod
├── k8s/
│   ├── 01-namespace.yaml
│   ├── 10-rabbitmq.yaml
│   ├── 20-go-server.yaml
│   ├── 21-go-api.yaml
│   ├── 22-go-consumer.yaml
│   ├── 30-api-rust.yaml
│   ├── 31-hpa.yaml
│   ├── 40-gateway.yaml
│   └── 41-httproute.yaml
├── locust/
├── operator/
├── proto/
│   └── report.proto
├── docs/
└── docker-compose.dev.yml
```

## 2. Instalaciones

### 2.1 Utilidades base

```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y curl wget jq unzip git make build-essential \
ca-certificates gnupg apt-transport-https python3-venv python3-pip
```

### 2.2 Go

```bash
GO_VERSION=$(curl -s "https://go.dev/VERSION?m=text" | head -1)
curl -LO https://go.dev/dl/${GO_VERSION}.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf ${GO_VERSION}.linux-amd64.tar.gz
rm ${GO_VERSION}.linux-amd64.tar.gz

echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc
go version
```

### 2.3 Rust

```bash
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
source "$HOME/.cargo/env"
rustc --version && cargo --version
```

### 2.4 Protocol Buffers y plugins de Go

```bash
sudo apt install -y protobuf-compiler
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

protoc --version
protoc-gen-go --version
protoc-gen-go-grpc --version
```
### 2.5 Google Cloud CLI, kubectl y el plugin de GKE

```bash
curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg
echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee /etc/apt/sources.list.d/google-cloud-sdk.list
sudo apt update
sudo apt install -y google-cloud-cli google-cloud-cli-gke-gcloud-auth-plugin kubectl

gcloud --version
kubectl version --client
gke-gcloud-auth-plugin --version
```

Para iniciar sesión en WSL se debe utilizar:

```bash
gcloud auth login --no-launch-browser
gcloud config set project PROJECT_ID
```

### 2.6 Helm

```bash
curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
helm version --short
```

### 2.7 Operator SDK

```bash
ARCH=amd64
OS=linux
curl -LO https://github.com/operator-framework/operator-sdk/releases/latest/download/operator-sdk_${OS}_${ARCH}
chmod +x operator-sdk_${OS}_${ARCH}
sudo mv operator-sdk_${OS}_${ARCH} /usr/local/bin/operator-sdk
operator-sdk version
```

### 2.8 Locust (en entorno virtual)

Va en un entorno virtual para no ensuciar el Python del sistema:

```bash
python3 -m venv ~/locust-env
source ~/locust-env/bin/activate
pip install locust
locust --version
deactivate

# Para usarlo despues:
source ~/locust-env/bin/activate
```

### 2.9 ORAS (para el OCI Artifact en ZOT)

No se utiliza snap porque WSL puede ocasionar fallas, a pesar de que es más fácil se debe hacer de la siguiente forma.

```bash
ORAS_VERSION=$(curl -s https://api.github.com/repos/oras-project/oras/releases/latest | jq -r .tag_name | sed 's/^v//')
curl -LO https://github.com/oras-project/oras/releases/download/v${ORAS_VERSION}/oras_${ORAS_VERSION}_linux_amd64.tar.gz
tar -xzf oras_${ORAS_VERSION}_linux_amd64.tar.gz oras
sudo mv oras /usr/local/bin/
rm oras_${ORAS_VERSION}_linux_amd64.tar.gz
oras version
```

### 2.10 Dapr CLI

```bash
wget -q https://raw.githubusercontent.com/dapr/cli/master/install/install.sh -O - | /bin/bash
dapr --version
```

### 2.11 Docker

Al estar en Windows se utiliza Docker Deskstop así que lo que hay que hacer para que funcione en Ubuntu con WSL es lo siguiente:

1. Abrir Docker Deskstop
2. Ir a Settings, luego a Resources, después a WSL Integration, activar Ubuntu y marcar también la casilla Enable integration with my default WSL distro.
3. Haz clic en Apply & restart.
4. Verificar en Ubuntu con el siguiente comando: `docker run hello-world`

## 3. Configuraciones Generales

### 3.1 go-api

Ejecutar los siguientes comandos dentro de la carpeta de go-api/

```bash
cd go-api
go mod init go-api
protoc --proto_path=../proto --go_out=. --go-grpc_out=. ../proto/report.proto
go get google.golang.org/grpc google.golang.org/protobuf
```

Despues creamos el archivo `main.go`, actualizamos dependendencias y comprobamos que se hayan creado los documentos `report.pb.go` y `report_grpc.pb.go` dentro de la carpeta `proto/`

```bash
go mod tidy
ls proto
```

Creamos el archivo `Dockerfile` y `.dockerignore` a este le colocamos `*.md`.

### 3.2 Rust

Ejecutamos dentro de la carpeta `api-rust/` el siguiente comando

```bash
cargo init --name api-rust
```

Esto creara la carpeta con ese nombre, el archivo `main.rs` dentro de src/ y `Cargo.toml`. Modificaremos las dependencias del archivo `.toml` colocando las dependencias a mano. 

Las dependencias que hay que colocar son las siguientes:

```toml
axum = "0.7"
tokio = { version = "1", features = ["rt-multi-thread", "macros"] }
serde = { version = "1", features = ["derive"] }
reqwest = { version = "0.12", default-features = false, features = ["json"] }
```

Despues ejecuta el siguiente comando para compilarlo.

```bash
cargo build
```

Luego creamos el `Dockerfile`.

### 3.3 Prueba local con Docker Compose

En la raíz del proyecto creamos un archivo `docker-compose.dev.yml` para poder hacer pruebas de manera local. Hace que los dos contenedores compartan red, como si fuera un pod de Kubernetes.

#### 3.4 Pruebas de funcionamiento

```bash
docker compose -f docker-compose.dev.yml up --build
```

En otra terminal se deben probar los siguientes comandos:

```bash
curl -i localhost:8080/

curl -i -X POST localhost:8080/grpc-202307716 -H "Content-Type: application/json" \
  -d '{"country":"ESP","warplanes_in_air":10,"warships_in_water":5,"timestamp":"2026-10-21T10:00:00Z"}'

curl -i -X POST localhost:8080/grpc-202307716 -H "Content-Type: application/json" \
  -d '{"country":"XXX","warplanes_in_air":10,"warships_in_water":5,"timestamp":"2026-10-21T10:00:00Z"}'
```

Debería de responder con lo siguiente:
1. `200 OK`
2. `500` "error al contactar el servidor gRPC"
3. `400` "pais invalido"

Al terminar ejecutar el siguiente comando.
```bash
docker compose -f docker-compose.dev.yml down
```

### 3.5 Proto

Creamos la carpeta proto y adentro creamos el archivo con la extensión `.proto`.

### 3.6 go-grpc-server

Este servicio hace dos cosas:

- Escucha gRPC en el puerto 50051 y recibe **WarReportRequest**.
- Publica el reporte en una cola de RabbitMQ.

Ejecutar los siguientes comandos dentro de la carpeta de `go-grpc-server/`.

```bash
go mod init go-grpc-server
protoc --proto_path=../proto --go_out=. --go-grpc_out=. ../proto/report.proto
cp ../go-api/Dockerfile ../go-api/.dockerignore .
```

Despues creamos el archivo `main.go`, actualizamos dependendencias y comprobamos que se hayan creado los documentos `report.pb.go` y `report_grpc.pb.go` dentro de la carpeta `proto/`

```bash
go mod tidy
ls proto
```

Creamos el archivo `Dockerfile` y `.dockerignore` a este le colocamos `*.md`.

### 3.7 Prueba local con Docker Compose

En la raíz del proyecto actualizamos el archivo `docker-compose.dev.yml` para poder hacer pruebas de manera local. El servicio llamado `go-server-svc` para que `GRPC_SERVER_ADDR` sea igual que en Kubernetes.

#### 3.8 Pruebas de funcionamiento

```bash
cd ~/202307716_LAB_SO1_2S2026/proyecto3
docker compose -f docker-compose.dev.yml up --build -d
docker compose -f docker-compose.dev.yml logs go-writer go-server-svc

curl -i -X POST localhost:8080/grpc-202307716 -H "Content-Type: application/json" -d '{"country":"ESP","warplanes_in_air":10,"warships_in_water":5,"timestamp":"2026-10-21T10:00:00Z"}'

docker compose -f docker-compose.dev.yml exec rabbitmq rabbitmqctl list_queues name messages
```

El resultado que se espera es `200` y `reports 1`. Luego se hace una prueba con 100 peticiones aleatorias.

```bash
for i in $(seq 1 100); do
  curl -s -o /dev/null -X POST localhost:8080/grpc-202307716 -H "Content-Type: application/json" \
    -d '{"country":"ESP","warplanes_in_air":10,"warships_in_water":5,"timestamp":"2026-10-21T10:00:00Z"}' &
done; wait
docker compose -f docker-compose.dev.yml exec rabbitmq rabbitmqctl list_queues name messages
```

El resultado que se espera es `reports 101` y la cola se puede supervisar en `http://localhost:15672` con la credenciales del documento `docker-compose.dev.yml`.

### 3.9 Go-consumer

**Consumer en Go + Valkey**: El Consumer lee la cola reports, procesa cada mensaje y lo guarda en Valkey. Primero lo probamos con un Valkey en Docker, y más adelante el Operator lo reemplazará en GKE.

Las keys creadas que se colocaran en Grafana son:

- Máximo y mínimo de aviones y barcos: `max:planes`, `min:planes`, `max:ships` y `min:ships`
- Top de países: `top:planes` y `top:ships`
- Moda: `freq:planes` y `freq:ships`
- Reportes por país: `reports:count`
- Serie temporal del país asignado: `ts:esp`

Ejecutar los siguientes comandos dentro de la carpeta de `go-consumer/`

```bash
go mod init go-consumer
cp ../go-api/Dockerfile ../go-api/.dockerignore .
```

Creamos el archivo `main.go` y actualizamos dependencias.

```bash
go get github.com/rabbitmq/amqp091-go github.com/redis/go-redis/v9
go mod tidy
```

Creamos el archivo `Dockerfile` y `.dockerignore` a este le colocamos `*.md`. 

### 3.10 Prueba local con Docker Compose

En la raíz del proyecto actualizamos el archivo `docker-compose.dev.yml` para poder hacer pruebas de manera local. Agregamos los servicios de `valkey` y `go-consumer`.

#### 3.11 Pruebas de funcionamiento

```bash
cd ~/202307716_LAB_SO1_2S2026/proyecto3
docker compose -f docker-compose.dev.yml up --build -d
docker compose -f docker-compose.dev.yml logs go-consumer

vk() { docker compose -f docker-compose.dev.yml exec -T valkey valkey-cli "$@"; }
vk HGETALL reports:count
vk ZREVRANGE top:planes 0 4 WITHSCORES
vk XLEN ts:esp
vk TTL max:planes
```

El resultado que se espera es `ESP 101`, `ESP 1010`, `101` y un TTL cercano a **86400**. Luego se hace una prueba con datos variados y aleatorios.

```bash
for i in $(seq 1 200); do
  C=$(shuf -e USA RUS CHN ESP GMT -n1)
  P=$((RANDOM % 51)); S=$((RANDOM % 31))
  curl -s -o /dev/null -X POST localhost:8080/grpc-202307716 -H "Content-Type: application/json" \
    -d "{\"country\":\"$C\",\"warplanes_in_air\":$P,\"warships_in_water\":$S,\"timestamp\":\"$(date -u +%FT%TZ)\"}" &
done; wait

vk GET max:planes; vk GET min:planes; vk GET max:ships; vk GET min:ships
vk ZREVRANGE top:planes 0 4 WITHSCORES
vk ZREVRANGE freq:planes 0 0 WITHSCORES
vk HGETALL reports:count
docker compose -f docker-compose.dev.yml exec rabbitmq rabbitmqctl list_queues name messages
```

El resultado que se espera son máximos y mínimos distintos de 10 y 5, cinco países en el top, la suma de los reportes igual a 301 y la cola de RabbitMQ en 0.

### 3.12 Locust

Locust lee un archivo `config.json` con la ruta, los países y los rangos. Este archivo es el `OCI Artifact` que se sube a `Zot`.

###  3.13 Operator para Valkey 

Desde la ruta principal:

```shell
mkdir -p operator && cd operator
operator-sdk init --plugins helm --domain mumnk8s.io --project-name mumn-operator
operator-sdk create api --group cache --version v1alpha1 --kind ValkeyInstance
operator-sdk create api --group cache --version v1alpha1 --kind GrafanaInstance
```

Cada create api genera un chart de ejemplo en `helm-charts/<kind>/`, un CRD y un CR de muestra en `config/samples/`. 

Veras **valkeyinstance** y **grafanainstance** en **helm-charts**, además de los CRD de ambos.

### 3.14 Helm-charts

Ahora borra las plantillas de ejemplo que genero create api.

```shell
rm -rf helm-charts/valkeyinstance/templates/* helm-charts/grafanainstance/templates/*
ls helm-charts
```

Despues dentro de `helm-charts/valkeyinstance/` hacemos lo siguiente:
- Modificamos el `values.yaml`
- Creamos dentro de templates `service.yaml`, `statefulset.yaml`, con ese ultimo logramos persistencia, gracias a --appendonly yes

Despues dentro de `helm-charts/grafanainstance/`:
- Modificamos el `values.yaml`
- Creamos dentro de templates `secret.yaml`, `configmap-provisioning.yaml`, `configmap-dashboards.yaml`, `deployment.yaml`, `service.yaml`, 

Creamos la carpeta `dashboards` y ponemos `mumn.json` este contendra un pequeño ejemplo de como estara grafana.

### 3.15 Recursos personalizados (CR) y permisos

#### 3.15.1 samples
Dentro de samples modificamos `config/samples/cache_v1alpha1_grafanainstance.yaml` y `config/samples/cache_v1alpha1_valkeyinstance.yaml`

LoadBalancer te da una IP pública para abrir Grafana desde el navegador. Si se desea quitar 

#### 3.15.2 rbac

En el archivo `config/rbac/role.yaml`, al final de la lsita de `rules:` agregamos 

```yaml
- apiGroups: [""]
  resources: ["configmaps", "secrets", "services", "persistentvolumeclaims", "serviceaccounts"]
  verbs: ["create", "delete", "get", "list", "patch", "update", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets"]
  verbs: ["create", "delete", "get", "list", "patch", "update", "watch"]
```

Si falta alguno, el Operator lo dirá en sus logs con un error forbidden

### 3.16 Probamos
Para validar ejecutamos desde operator/
```shell
helm lint helm-charts/valkeyinstance
helm lint helm-charts/grafanainstance
helm template valkey helm-charts/valkeyinstance
helm template grafana helm-charts/grafanainstance
docker pull grafana/grafana:11.4.0 && docker pull valkey/valkey:8
```

### 3.17 Grafana

Creamos un dashboard de manera local, para comprobar que todo funciona localmente y así subir todo funcionando a GCP.

Creamos la carpeta `grafana-dev/` y dentro `datasource.yaml`.

Agregamos el servicio a `docker-compose.dev.yml`.

Armamos todo el dasboard, comprobamos que los datos se ven correctamente y finalmente exportamos el dasboard y lo pegamor en la siguiente ruta `operator/helm-charts/grafanainstance/dashboards/mumn.json`.