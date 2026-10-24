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

Generación de carga contra el Gateway.
```shell
GW=$(kubectl get gateway api-gateway -n mumnk8s -o jsonpath='{.status.addresses[0].value}')
mkdir -p /tmp/cfg && oras pull $ZOT/locust-config:v1 -o /tmp/cfg
source ~/locust-env/bin/activate
CONFIG_FILE=/tmp/cfg/config.json locust -f locust/locustfile.py --host http://$GW --headless -u 200 -r 20 -t 4m 2>&1 | tail -n 8
deactivate
kubectl get pods -n mumnk8s -l app=api-rust
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

```bash
mkdir -p operator && cd operator
operator-sdk init --plugins helm --domain mumnk8s.io --project-name mumn-operator
operator-sdk create api --group cache --version v1alpha1 --kind ValkeyInstance
operator-sdk create api --group cache --version v1alpha1 --kind GrafanaInstance
```

Cada create api genera un chart de ejemplo en `helm-charts/<kind>/`, un CRD y un CR de muestra en `config/samples/`. 

Veras **valkeyinstance** y **grafanainstance** en **helm-charts**, además de los CRD de ambos.

### 3.14 Helm-charts

Ahora borra las plantillas de ejemplo que genero create api.

```bash
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
```bash
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


## 4. Infraestructura del Container Registry (ZOT)

### 4.1 Preparación de GCP y creación de VM de Zot

Aqui vamos a levantar Zot Registry en una máquina virtual externa (VM en GCP con tipo de instancia E2) configurado con HTTPS.

```bash
# Seleccionar el proyecto y activar las APIs
gcloud config set project project-5fea5468-0f77-43ea-8e6
gcloud config set compute/region us-central1
gcloud config set compute/zone us-central1-a
gcloud services enable compute.googleapis.com container.googleapis.com

# Revisamos las cuotas para ver si tenemos espacio para crear la VM y el clúster
gcloud compute regions describe us-central1 --format=json \
  | jq -r '.quotas[] | select(.metric|test("^(CPUS|IN_USE_ADDRESSES|STATIC_ADDRESSES|DISKS_TOTAL_GB|SSD_TOTAL_GB)$")) | "\(.metric)\t\(.usage)/\(.limit)"'

# Revisa qué existe, debido a que es una cuenta compartida. Para no repetir algún nombre.
gcloud compute instances list
gcloud compute addresses list
gcloud compute firewall-rules list --format="table(name,targetTags.list())"
gcloud container clusters list

# Reservar una IP fija (para que no cambie si reinicias la VM, anote la IP del segundo comando)
gcloud compute addresses create zot-ip --region us-central1

# Abrir los puertos web (80 y 443)
gcloud compute firewall-rules create allow-web-zot \
  --allow tcp:80,tcp:443 --direction INGRESS --target-tags zot

# Crear la VM
gcloud compute instances create zot-vm-diego-diego \
  --zone us-central1-a \
  --machine-type e2-small \
  --image-family ubuntu-2404-lts-amd64 --image-project ubuntu-os-cloud \
  --boot-disk-size 30GB --boot-disk-type pd-standard \
  --address zot-ip-diego --tags zot-diego \
  --description "Zot registry Diego España 202307716"

# Guardar la IP fija en una variable de entorno
ZOT_IP=$(gcloud compute addresses describe zot-ip --region us-central1 --format='value(address)')
echo "ZOT_IP=$ZOT_IP"
echo "export ZOT=${ZOT_IP}.sslip.io" >> ~/.bashrc && source ~/.bashrc
echo $ZOT

# Entrar a la VM
gcloud compute ssh zot-vm-diego --zone us-central1-a
```
La primera vez te pedirá crear una clave SSH (puedes dejar la contraseña vacía). Si todo sale bien, verás el prompt de la VM, algo como usuario@zot-vm-diego:~$.

Si alguno de estos comandos pide habilitar una API o confirmar algo, responde que sí.

Si desea detener la vm para ahorrar recuersos ejecuta: 

```bash
gcloud compute instances stop zot-vm-diego --zone us-central1-a    # apagar
gcloud compute instances start zot-vm-diego --zone us-central1-a   # encender
```

Si se desea borrar la vm, ejecutar:

```bash
gcloud compute instances delete zot-vm-diego --zone=us-central1-a
```

- IP estática: `35.239.210.169`

### 4.2 Dentro de la VM ZOT

Una vez dentro de la VM ejecutar

```bash
sudo apt update && sudo apt install -y jq apache2-utils curl

# Descargamos zot
URL=$(curl -s https://api.github.com/repos/project-zot/zot/releases/latest \
  | jq -r '.assets[] | select(.name=="zot-linux-amd64") | .browser_download_url')
echo "$URL"
sudo curl -L -o /usr/local/bin/zot "$URL"
sudo chmod +x /usr/local/bin/zot
zot --version

# Usuario del servicio, carpetas y contraseña
sudo useradd --system --no-create-home --shell /usr/sbin/nologin zot
sudo mkdir -p /etc/zot /var/lib/zot
sudo htpasswd -B -c /etc/zot/htpasswd mumn
```
El último comando te pide la contraseña dos veces (así no queda en el historial). El usuario será mumn. Guarda esa contraseña, porque se usara en docker login y en el imagePullSecret de GKE.

Configuramos el zot

```bash
sudo tee /etc/zot/config.json > /dev/null <<'EOF'
{
  "distSpecVersion": "1.1.0",
  "storage": { "rootDirectory": "/var/lib/zot" },
  "http": {
    "address": "127.0.0.1",
    "port": "5000",
    "auth": { "htpasswd": { "path": "/etc/zot/htpasswd" } }
  },
  "log": { "level": "info" },
  "extensions": {
    "search": { "enable": true },
    "ui": { "enable": true }
  }
}
EOF
sudo chown -R zot:zot /etc/zot /var/lib/zot
sudo chmod 640 /etc/zot/htpasswd
```
Zot solo escucha en 127.0.0.1. Nadie de internet llega a él directamente, sino a través de Caddy con HTTPS.

Servicio systemd

```bash
sudo tee /etc/systemd/system/zot.service > /dev/null <<'EOF'
[Unit]
Description=Zot OCI registry
After=network.target

[Service]
User=zot
ExecStart=/usr/local/bin/zot serve /etc/zot/config.json
Restart=always

[Install]
WantedBy=multi-user.target
EOF
sudo systemctl daemon-reload
sudo systemctl enable --now zot
sudo systemctl status zot --no-pager | head -n 5
curl -i http://127.0.0.1:5000/v2/
```
Debes ver active (running) y que el curl devuelva 401 Unauthorized. Eso es lo correcto, porque pide credenciales.

```bash
# Instalar Caddy
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install -y caddy

# Configurar Caddy
IP=$(curl -s -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip)
printf '%s.sslip.io {\n    reverse_proxy 127.0.0.1:5000\n}\n' "$IP" | sudo tee /etc/caddy/Caddyfile
sudo systemctl reload caddy
sleep 15
curl -i https://$IP.sslip.io/v2/
```
### 4.3 Desde la pc

Probar desde la pc

```bash
docker login $ZOT -u mumn
curl -s -u mumn https://$ZOT/v2/_catalog; echo
```

Pasamos a la pc. Creamos un script `build-push.sh` y `mirror-imgages`, finalmente ejecutamos.

```bash
cat > scripts/build-push.sh <<'EOF'
#!/usr/bin/env bash
# Uso: ./scripts/build-push.sh api-rust go-api go-grpc-server go-consumer
set -euo pipefail
ZOT="${ZOT:?define ZOT, ej: export ZOT=1.2.3.4.sslip.io}"
TAG="${TAG:-v1}"
cd "$(dirname "$0")/.."
for svc in "$@"; do
  docker build -t "$ZOT/$svc:$TAG" "./$svc"
  docker push "$ZOT/$svc:$TAG"
done
EOF

cat > scripts/mirror-images.sh <<'EOF'
#!/usr/bin/env bash
# Copia las imágenes de terceros a Zot para que el clúster descargue todo de ahí
set -euo pipefail
ZOT="${ZOT:?define ZOT}"
mirror() { docker pull "$1" && docker tag "$1" "$ZOT/$2" && docker push "$ZOT/$2"; }
mirror rabbitmq:3-management rabbitmq:3-management
mirror valkey/valkey:8 valkey:8
mirror grafana/grafana:11.4.0 grafana:11.4.0
EOF
chmod +x scripts/*.sh
./scripts/build-push.sh api-rust go-api go-grpc-server go-consumer
./scripts/mirror-images.sh
curl -s -u mumn https://$ZOT/v2/_catalog; echo
```

### 4.4 OCI Artifact

```bash
cd locust
oras login $ZOT -u mumn
oras push $ZOT/locust-config:v1 config.json:application/json
mkdir -p /tmp/cfg && oras pull $ZOT/locust-config:v1 -o /tmp/cfg && cat /tmp/cfg/config.json
cd ..
```

## 5. Creación del clúster

Se usaran 2 nodos e2-standard-2.

```bash
gcloud container clusters create mumn-diego \
  --zone us-central1-a --num-nodes 2 \
  --machine-type e2-standard-2 \
  --disk-type pd-standard --disk-size 35 \
  --gateway-api=standard \
  --async

gcloud container clusters list
```

Debes ver 2 nodos Ready.

```bash
gcloud container clusters get-credentials mumn-diego --zone us-central1-a
kubectl config current-context
kubectl get nodes -o wide

kubectl apply -f k8s/01-namespace.yaml

# Clave aleatoria de RabbitMQ (hex, para que no rompa la URL)
RMQ_PASS=$(openssl rand -hex 16)
kubectl -n mumnk8s create secret generic rabbitmq-creds \
  --from-literal=user=mumn --from-literal=pass=$RMQ_PASS \
  --from-literal=url="amqp://mumn:$RMQ_PASS@rabbitmq:5672/"
unset RMQ_PASS

# Credenciales para que los nodos descarguen de Zot (la contraseña se pide sin mostrarla)
read -rsp "Contraseña de Zot: " ZOT_PASS; echo
kubectl -n mumnk8s create secret docker-registry zot-creds \
  --docker-server=$ZOT --docker-username=mumn --docker-password="$ZOT_PASS"
unset ZOT_PASS

kubectl -n mumnk8s get secrets
```

Para reducir el consumo mientras no se usa el clúster, es necesario quitarle los nodos y al volverlo a usar ponerselos de nuevo.

```bash
# Apagar nodos (los pods quedan Pending; PVC, Gateway e IP se conservan)
gcloud container clusters resize mumn-diego --node-pool default-pool --num-nodes 0 --zone us-central1-a

# Volver a encender, y esperar 3 a 5 minutos
gcloud container clusters resize mumn-diego --node-pool default-pool --num-nodes 2 --zone us-central1-a
kubectl get pods -A
```

### 5.1 Poner Zot en YAML y desplegar

```bash
sed -i "s/ZOT_HOST/$ZOT/g" k8s/*.yaml operator/config/samples/*.yaml
grep -rn "ZOT_HOST" k8s operator/config || echo "sin marcadores pendientes"
grep -n "image:" k8s/*.yaml

kubectl apply -f k8s/
kubectl -n mumnk8s get pods -w
```

Espera a que todos estén Running (Ctrl+C para salir). Luego revisa el Gateway, que tarda entre 3 y 10 minutos en recibir una IP:

```bash
kubectl get gateway,httproute -n mumnk8s
kubectl get hpa -n mumnk8s
```

Cuando ADDRESS tenga una IP y esté PROGRAMMED True:

```bash
GW=$(kubectl get gateway api-gateway -n mumnk8s -o jsonpath='{.status.addresses[0].value}')
echo $GW
curl -i -X POST http://$GW/grpc-202307716 -H "Content-Type: application/json" \
  -d '{"country":"ESP","warplanes_in_air":12,"warships_in_water":7,"timestamp":"2026-10-23T08:00:00Z"}'
kubectl logs -l app=api-rust -n mumnk8s --tail=10
```

Resultado esperado: 200 “Reporte de esp procesado”. Después verifica que llegó a Valkey:

```bash
kubectl exec -n mumnk8s deploy/valkey-temp -- valkey-cli HGETALL reports:count
```

## 5.2 Operator en GKE

Dentro de la carpeta `operator/` ejecutamos lo siguiente

```bash
grep -rn "kube-rbac-proxy" config/ || echo "sin kube-rbac-proxy"
grep -n "serviceaccounts\|statefulsets\|configmaps\|persistentvolumeclaims" config/rbac/role.yaml
```

Ajustamos el Deployment del Operator con más memoria.

```bash
sed -i 's/^\(\s*\)serviceAccountName: controller-manager/\1serviceAccountName: controller-manager\n\1imagePullSecrets:\n\1- name: zot-creds/' config/manager/manager.yaml
sed -i 's/memory: 128Mi/memory: 512Mi/' config/manager/manager.yaml
grep -n -B1 -A2 "imagePullSecrets" config/manager/manager.yaml
grep -n "memory" config/manager/manager.yaml
```

Construimos, subimos y desplegamos el operator.

```bash
export IMG=$ZOT/mumn-operator:v1
make docker-build IMG=$IMG
make docker-push IMG=$IMG

kubectl create namespace mumn-operator-system
read -rsp "Contraseña de Zot: " ZOT_PASS; echo
kubectl -n mumn-operator-system create secret docker-registry zot-creds \
  --docker-server=$ZOT --docker-username=mumn --docker-password="$ZOT_PASS"
unset ZOT_PASS

make deploy IMG=$IMG
kubectl -n mumn-operator-system get pods
kubectl get crd | grep mumnk8s
```

Cada create api genera un chart de ejemplo en `helm-charts/<kind>/`, un CRD y un CR de muestra en `config/samples/`. Luego muestra la estructura:

```bash
ls helm-charts
ls config/samples
ls config/crd/bases
```

Veras **valkeyinstance** y **grafanainstance** en **helm-charts**, además de los CRD de ambos.

### 5.3 Valkey y Grafana

Creamos Valkey y Grafana con los CR.

```bash
kubectl apply -f config/samples/cache_v1alpha1_valkeyinstance.yaml

GF_PASS=$(openssl rand -base64 12 | tr -d '/+=')
sed "s/CAMBIA-ESTA-CLAVE/$GF_PASS/" config/samples/cache_v1alpha1_grafanainstance.yaml | kubectl apply -f -
echo "Grafana -> usuario: admin  clave: $GF_PASS   (anótala, no la subas al repo)"
unset GF_PASS

kubectl -n mumnk8s get valkeyinstances,grafanainstances,statefulset,pods,pvc,svc
```

#### 5.3.1 Probamos
Para validar ejecutamos desde `operator/`
```bash
helm lint helm-charts/valkeyinstance
helm lint helm-charts/grafanainstance
helm template valkey helm-charts/valkeyinstance
helm template grafana helm-charts/grafanainstance
```

### 5.4 Desplegar el resto y probar el flujo

Revisa que ya no este valkey temporal, despues ejecuta:

```bash
kubectl apply -f k8s/
kubectl get pods -n mumnk8s

# Esto se ejecuta despues
kubectl get gateway,httproute -n mumnk8s
```

Todos los pods deben estar Running. Repite el ultimo comando hasta que el Gateway muestre una IP y PROGRAMMED True

Cuando ya tengas la IP:

```bash
GW=$(kubectl get gateway api-gateway -n mumnk8s -o jsonpath='{.status.addresses[0].value}')
curl -i -X POST http://$GW/grpc-202300580 \
  -H "Content-Type: application/json" \
  -d '{"country":"esp","warplanes_in_air":10,"warships_in_water":5,"timestamp":"2026-10-19T10:00:00Z"}'

kubectl exec -n mumnk8s valkey-0 -- valkey-cli HGETALL reports:count
```

Esperado: 200 y esp con valor 1. Esto prueba el flujo completo con el Valkey del Operator.

```bash
kubectl get svc grafana -n mumnk8s
```

Entra a `http://<EXTERNAL-IP>`con usuario admin y la contraseña de tu CR.

Si grafana no abre ejecutar lo siguiente

### 5.5 Falco

```bash
helm repo add falcosecurity https://falcosecurity.github.io/charts && helm repo update
helm upgrade --install falco falcosecurity/falco -n falco --create-namespace \
  --version 9.2.1 -f falco/falco-values.yaml --wait --timeout 10m
kubectl -n falco get pods
```

### 5.6 Al terminar la sesión

```bash
kubectl delete -f operator/config/samples/ --ignore-not-found
kubectl delete pvc --all -n mumnk8s
gcloud container clusters delete mumnk8s --zone us-central1-a
gcloud compute instances stop zot-vm-diego --zone us-central1-a
gcloud compute disks list
```
