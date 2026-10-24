# Manual Técnico — Proyecto 3

| | |
|---|---|
| **Carnet** | 202307716 |
| **País asignado** | España (`ESP`) |
| **Ruta del flujo base** | `/grpc-202307716` |
| **Proyecto GCP** | `project-5fea5468-0f77-43ea-8e6` |
| **Clúster GKE** | `mumn-diego` (`us-central1-a`) |
| **Registry** | Zot sobre HTTPS en una VM de GCP, fuera del clúster |

## Índice

1. [Arquitectura general](#1-arquitectura-general)
2. [Flujo completo de datos](#2-flujo-completo-de-datos)
3. [Configuración de Gateway API](#3-configuración-de-gateway-api)
4. [Comunicación REST y gRPC](#4-comunicación-rest-y-grpc)
5. [Uso de RabbitMQ](#5-uso-de-rabbitmq)
6. [Operator: CRDs, Helm charts y reconciliación](#6-diseño-y-funcionamiento-del-operator)
7. [Configuración de HPA](#7-configuración-de-hpa)
8. [Zot: publicación y consumo de imágenes, OCI Artifact](#8-publicaciónconsumo-de-imágenes-desde-zot)
9. [Falco](#9-falco)
10. [Pruebas realizadas y conclusiones](#10-pruebas-realizadas-y-conclusiones)
11. [Guía de instalación](#11-guía-de-instalación-paso-a-paso)
12. [Extra: Dapr](#12-extra-dapr)

---

## 1. Arquitectura general

Se siguió la arquitectura del enunciado: Locust genera reportes militares en JSON, que entran por un Gateway de Kubernetes, pasan por una API en Rust y por dos microservicios en Go (comunicados por gRPC), se encolan en RabbitMQ, un consumidor los guarda en Valkey y Grafana los visualiza. Valkey y Grafana los despliega un Operator propio.

![Arquitectura](./img/arquitectura.png)

### 1.1 Componentes

| Componente | Tecnología | Recursos en Kubernetes | Imagen (Zot) | Réplicas |
|---|---|---|---|---|
| Generador de carga | Locust (Python), ejecutado desde WSL | — | — | — |
| Entrada | Gateway API de GKE (balanceador L7 global) | `Gateway api-gateway`, `HTTPRoute api-route` | — | — |
| API REST | Rust (axum, tokio, reqwest) | Deployment `api-rust-deploy`, Service `api-rust-service`, HPA | `api-rust:v1` | 1 a 3 (HPA) |
| Go · Deployment 1 | API REST + cliente gRPC | Deployment `go-api-deploy` (2 contenedores), Service `go-api-svc` | `go-api:v1` (`MODE=rest` y `MODE=client`) | 1 |
| Go · Deployment 2 | Servidor gRPC + writer | Deployment `go-server-deploy` (2 contenedores), Service headless `go-server-svc` | `go-grpc-server:v1` (`MODE=server` y `MODE=writer`) | 1 (2 en pruebas) |
| Broker | RabbitMQ 3 (management) | Deployment y Service `rabbitmq` | `rabbitmq:3-management` | 1 |
| Consumidor | Go | Deployment `go-consumer-deploy` | `go-consumer:v1` | 1 |
| Almacenamiento | Valkey 8 | StatefulSet `valkey`, PVC `data-valkey-0`, Service `valkey-svc` (creados por el Operator) | `valkey:8` | 1 (2 en pruebas) |
| Visualización | Grafana 11.4.0 + plugin Redis | Deployment y Service `grafana` (creados por el Operator) | `grafana:11.4.0` | 1 |
| Operator | operator-sdk, plugin Helm | Deployment `mumn-operator-controller-manager` | `mumn-operator:v2` | 1 |
| Seguridad | Falco 0.45.1 (chart 9.2.1) | DaemonSet `falco` (namespace `falco`) | imagen oficial | 1 por nodo |

### 1.2 Infraestructura

- **Clúster GKE Standard** `mumn-diego`, zona `us-central1-a`, 2 nodos `e2-standard-2` con disco de 35 GB (`pd-standard`) y Gateway API habilitada (`--gateway-api=standard`).
- **VM del registry** `zot-vm-diego` (`e2-small`, Ubuntu 24.04), con IP reservada `zot-ip-diego` y regla de firewall `allow-web-zot-diego` (TCP 80 y 443).
- Se usaron instancias **E2** en lugar de N1 porque son más baratas.
- Namespaces: `mumnk8s` (aplicación), `mumn-operator-system` (Operator) y `falco`.

![Nodos del clúster](./img/evidencia14.png)

### 1.3 Estructura del repositorio

```text
Proyecto3/
├── proto/report.proto          # contrato gRPC
├── api-rust/                   # API REST en Rust
├── go-api/                     # Deployment 1: REST + cliente gRPC
├── go-grpc-server/             # Deployment 2: servidor gRPC + writer
├── go-consumer/                # Consumidor RabbitMQ -> Valkey
├── k8s/                        # Manifiestos de la aplicación (01 a 41)
├── operator/                   # Operator (Helm): charts de Valkey y Grafana, CRs, RBAC
├── falco/                      # Regla personalizada y values del chart
├── locust/                     # locustfile.py y config.json (OCI Artifact)
├── scripts/                    # build-push, mirror-images, loadtest, preflight, evidencias
├── results/                    # Resultados de las pruebas de carga
├── docs/                       # Este manual, imágenes y evidencias
└── docker-compose.dev.yml      # Entorno local de desarrollo
```

### 1.4 Decisiones de diseño

- **Una imagen por servicio Go, dos contenedores por pod.** Cada binario Go tiene un modo (`MODE`) que decide su rol. Los dos contenedores de un pod comparten red y se hablan por `localhost`, lo que cumple "dos contenedores por deployment" sin construir cuatro imágenes distintas.
- **RabbitMQ desacopla la ingesta del almacenamiento.** Si Valkey falla, los reportes se acumulan en la cola y se procesan al volver.
- **Escritura atómica en Valkey.** El consumidor actualiza todas las estadísticas con un único script Lua, así que varias réplicas del consumidor no se pisan.
- **Todo lo que corre en `mumnk8s` sale de Zot**, incluidos RabbitMQ, Valkey y Grafana (copiados al registry).

---

## 2. Flujo completo de datos

```mermaid
sequenceDiagram
    autonumber

    participant L as Locust
    participant G as Gateway GCP
    participant R as API Rust
    participant A as Go API REST
    participant C as gRPC Client
    participant S as gRPC Server
    participant W as Go Writer
    participant Q as RabbitMQ
    participant K as Consumer
    participant V as Valkey
    participant D as Grafana

    L->>G: POST /grpc-202307716
    G->>R: HTTP :80 → :8080
    R->>A: POST /report

    A->>C: POST /send
    C->>S: gRPC SendReport

    S->>W: POST /publish
    W->>Q: Publicar en "reports"

    Q-->>W: Confirmación
    W-->>S: 200 OK
    S-->>C: WarReportResponse
    C-->>A: JSON con estado
    A-->>R: 200 OK
    R-->>G: 200 OK

    Q->>K: Entregar mensaje
    K->>V: Ejecutar script Lua atómico

    D->>V: GET / HGET / HGETALL / XRANGE
```

### 2.1 Mensaje

```json
{
  "country": "ESP",
  "warplanes_in_air": 21,
  "warships_in_water": 9,
  "timestamp": "2026-10-24T09:00:00Z"
}
```

Locust genera el país al azar entre `USA`, `RUS`, `CHN`, `ESP` y `GTM`, los aviones entre 0 y 50 y los barcos entre 0 y 30. Estos rangos y la ruta salen de `config.json`, publicado como OCI Artifact en Zot. El cliente gRPC convierte `GTM` al valor `gmt` del enum del `.proto`; en el dashboard Guatemala aparece como `GMT`.

### 2.2 Paso a paso

1. Locust envía el JSON al Gateway, que lo dirige a `api-rust-service`.
2. **Rust** recibe en `/grpc-202307716`, lo valida como JSON y lo reenvía a `go-api-svc`. Registra una línea de log por reporte.
3. **`api-rest`** recibe en `/report` y lo envía al contenedor vecino `grpc-client` por `127.0.0.1:9090`.
4. **`grpc-client`** valida el país, lo convierte al enum y llama a `SendReport` en `go-server-svc:50051`.
5. **`grpc-server`** entrega el reporte al **`writer`** por `127.0.0.1:9091`.
6. El **writer** publica en la cola `reports` de RabbitMQ y espera la confirmación del broker. Solo entonces responde 200, y la respuesta vuelve por la misma cadena.
7. El **consumer** toma el mensaje, ejecuta el script Lua en Valkey y confirma (`ack`).
8. **Grafana** consulta Valkey y dibuja el dashboard.

### 2.3 Modelo de datos en Valkey

El consumidor mantiene 12 llaves. Todas llevan **TTL de 24 h** (`EXPIRE 86400`, renovado en cada escritura) y el stream se recorta a unas 20 000 entradas.

| Llave | Tipo | Contenido | Panel de Grafana |
|---|---|---|---|
| `max:planes`, `min:planes`, `max:ships`, `min:ships` | string | Extremos de un reporte individual | Cuatro paneles de máximo y mínimo |
| `top:planes`, `top:ships` | hash (país → suma) | Total acumulado por país | Top 5 |
| `freq:planes`, `freq:ships` | zset (valor → veces) | Frecuencia de cada valor | Base para la moda |
| `mode:planes`, `mode:ships` | string | Valor más frecuente (se actualiza en cada escritura) | Moda |
| `reports:count` | hash (país → total) | Reportes recibidos por país | Reportes recibidos (`ESP`) |
| `ts:esp` | stream | Un punto por reporte de España (`planes`, `ships`) | Evolución temporal |

La serie temporal usa el momento en que el consumidor procesa el reporte (el ID del stream) y no el campo `timestamp` del JSON.

![Evidencia del flujo](./img/evidencia2.png)

---

## 3. Configuración de Gateway API

El clúster se creó con `--gateway-api=standard`. El **Gateway** aprovisiona un balanceador de carga HTTP externo global en Google Cloud, y el **HTTPRoute** decide a qué servicio va cada ruta.

`k8s/40-gateway.yaml`

```yaml
# Gateway: crea un balanceador HTTP externo global en GCP
apiVersion: gateway.networking.k8s.io/v1beta1
kind: Gateway
metadata:
  name: api-gateway
  namespace: mumnk8s
spec:
  gatewayClassName: gke-l7-global-external-managed
  listeners:
    - name: http
      protocol: HTTP
      port: 80
```

`k8s/41-httproute.yaml`

```yaml
# Ruta del flujo estandar (gRPC)
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: api-route
  namespace: mumnk8s
spec:
  parentRefs:
    - name: api-gateway
  rules:
    - matches:
        - path:
            type: PathPrefix
            value: /grpc-202307716
      backendRefs:
        - name: api-rust-service
          port: 80
```

- `gke-l7-global-external-managed` es la clase de GKE para el Application Load Balancer externo global administrado.
- El backend es `api-rust-service` (ClusterIP, puerto 80 a 8080). GKE lo conecta al balanceador con grupos de endpoints de red (NEG), así que el tráfico va directo a los pods.
- El balanceador comprueba la salud de los pods con una petición a `/` y espera un `200`. La API de Rust responde `OK` en esa ruta y el Deployment declara una `readinessProbe` equivalente.
- Cualquier ruta que no sea `/grpc-202307716` devuelve 404 desde el balanceador.
- El listener es **HTTP en el puerto 80**. No se configuró TLS en el Gateway (ver limitaciones en 10.7).

```bash
kubectl get gateway,httproute -n mumnk8s
GW=$(kubectl get gateway api-gateway -n mumnk8s -o jsonpath='{.status.addresses[0].value}')
curl -i -X POST http://$GW/grpc-202307716 -H "Content-Type: application/json" \
  -d '{"country":"ESP","warplanes_in_air":12,"warships_in_water":7,"timestamp":"2026-10-24T09:00:00Z"}'
```

![Gateway y HTTPRoute](./img/evidencia1.png)

![Petición al Gateway](./img/evidencia15.png)
---

## 4. Comunicación REST y gRPC

### 4.1 Contrato gRPC (`proto/report.proto`)

```proto
syntax = "proto3";
package wartweets;
option go_package = "./proto";

message WarReportRequest {
    Countries country = 1;
    int32 warplanes_in_air = 2;
    int32 warships_in_water = 3;
    string timestamp = 4;
}

enum Countries {
    countries_unknown = 0;
    usa = 1;
    rus = 2;
    chn = 3;
    esp = 4;
    gmt = 5;
}

message WarReportResponse { string status = 1; }

service WarReportService {
    rpc SendReport (WarReportRequest) returns (WarReportResponse);
}
```

El código Go se genera con `protoc --go_out=. --go-grpc_out=. report.proto`.

### 4.2 Servicios y puertos

| Servicio / contenedor | Protocolo | Escucha | Llama a |
|---|---|---|---|
| API Rust | REST | `:8080`, ruta `/grpc-202307716` | `http://go-api-svc:8080/report` |
| Go `api-rest` | REST | `:8080`, `/report` | `http://127.0.0.1:9090/send` |
| Go `grpc-client` | REST local + cliente gRPC | `127.0.0.1:9090`, `/send` | `go-server-svc:50051` |
| Go `grpc-server` | gRPC | `:50051` | `http://127.0.0.1:9091/publish` |
| Go `writer` | REST local | `127.0.0.1:9091`, `/publish` | RabbitMQ (`rabbitmq:5672`) |

Los contenedores de un mismo pod comparten la red, por eso `grpc-client` y `writer` solo escuchan en `127.0.0.1`: no son accesibles desde fuera del pod.

### 4.3 Códigos de respuesta

| Situación | Respuesta |
|---|---|
| Reporte correcto | `200` con `{"status":"Reporte de esp procesado"}` |
| JSON inválido | `400 json invalido` |
| País desconocido | `400 pais invalido` |
| gRPC o writer no disponibles | `502` |
| Cola interna del writer llena | `503` |

### 4.4 Concurrencia

- **Rust:** tokio atiende cada petición como una tarea asíncrona, con un único cliente HTTP compartido y reutilización de conexiones.
- **Go REST y cliente gRPC:** `net/http` lanza una goroutine por petición. Todas comparten **una sola conexión gRPC** (HTTP/2 multiplexa muchas llamadas sobre ella), y el pool de conexiones HTTP tiene hasta 100 conexiones ociosas por destino.
- **Writer:** un **pool de 4 goroutines** (`WORKERS`) consume de un canal con buffer de 1024 trabajos. Cada goroutine tiene su propio canal AMQP (no son seguros entre hilos) y activa confirmaciones de publicación: cada reporte se confirma antes de responder.
- **Timeouts:** 10 s en los clientes REST, 5 s en la llamada gRPC y 5 s en el writer.

### 4.5 Balanceo gRPC entre réplicas

gRPC mantiene conexiones largas, así que un Service ClusterIP normal enviaría toda la carga a un solo pod. Se resolvió con dos medidas:

1. `go-server-svc` es un Service **headless** (`clusterIP: None`): el DNS devuelve la IP de cada pod.
2. El cliente usa `dns:///go-server-svc:50051` con la política `round_robin`, y el servidor cierra las conexiones cada 30 s (`MaxConnectionAge`) para que los clientes vuelvan a resolver el DNS y descubran réplicas nuevas.

```bash
kubectl -n mumnk8s logs -l app=go-server -c grpc-server --tail=3 --prefix   # muestra el reparto entre pods
```

### 4.6 Variables de entorno

| Servicio | Variable | Por defecto |
|---|---|---|
| Rust | `CARNET`, `GO_API_URL`, `LOG_REQUESTS` | `202307716`, `http://go-api-svc:8080/report`, `true` |
| Go api | `MODE` (`rest`/`client`), `CLIENT_URL`, `GRPC_SERVER_ADDR`, `LOG_REQUESTS` | `rest`, `http://127.0.0.1:9090/send`, `go-server-svc:50051` |
| Go server | `MODE` (`server`/`writer`), `WRITER_URL`, `RABBITMQ_URL`, `QUEUE`, `WORKERS` | `server`, `http://127.0.0.1:9091/publish`, —, `reports`, `4` |
| Consumer | `RABBITMQ_URL`, `QUEUE`, `VALKEY_ADDR`, `ASSIGNED_COUNTRY`, `WORKERS`, `PREFETCH`, `TTL_SECONDS`, `STREAM_MAXLEN` | —, `reports`, `valkey-svc:6379`, `esp`, `8`, `50`, `86400`, `20000` |

`LOG_REQUESTS=false` desactiva el log por petición (se usa en las pruebas de rendimiento para no sesgar la medición).

---

## 5. Uso de RabbitMQ

RabbitMQ es el único broker de mensajería del flujo base.

- **Despliegue:** `k8s/10-rabbitmq.yaml` (Deployment + Service), imagen `rabbitmq:3-management` copiada a Zot. Puertos 5672 (AMQP) y 15672 (consola de administración). Las credenciales están en el Secret `rabbitmq-creds`, creado con `kubectl`: no están en el repositorio.
- **Cola:** `reports`, durable. Los mensajes se publican con `DeliveryMode: Persistent`.
- **Productor (writer):** cuatro workers, cada uno con su canal y con confirmaciones de publicación. Si RabbitMQ no está disponible al arrancar, reintenta cada 3 s. Si pierde la conexión, el proceso termina y Kubernetes lo reinicia.
- **Consumidor:**
  - Ocho workers (goroutines) leen del mismo canal con `prefetch` de 50 mensajes.
  - **Ack manual:** el mensaje se confirma solo después de escribir en Valkey.
  - Si Valkey falla, `Nack` con reencolado y una pausa de 1 s.
  - Un mensaje con JSON inválido se descarta (`Reject` sin reencolar), porque nunca será válido.

```bash
kubectl exec -n mumnk8s deploy/rabbitmq -- rabbitmqctl list_queues name messages consumers
kubectl get pods -n mumnk8s
```

**Prueba de resiliencia (Valkey apagado).** Se detuvo Valkey y se enviaron 5 reportes: los 5 respondieron `200`, la cola mostró 5 mensajes sin confirmar y el consumer registró "error escribiendo en Valkey, se reencola". Al encender Valkey, el contador pasó de 1 a 6 y la cola volvió a 0: no se perdió ningún reporte.

![Cola de RabbitMQ](./img/evidencia3.png)
![RabbitMQ](./img/evidencia4.png)
![Recuperación con Valkey caído](./img/evidencia10.png)

---

## 6. Diseño y funcionamiento del Operator

### 6.1 Qué es y cómo se creó

Es un Operator **basado en Helm** generado con `operator-sdk`. En lugar de escribir la lógica de reconciliación en Go, cada Custom Resource se asocia a un **Helm chart**: el Operator instala o actualiza el chart usando el `spec` del recurso como valores.

```bash
operator-sdk init --plugins helm --domain mumnk8s.io --project-name mumn-operator
operator-sdk create api --group cache --version v1alpha1 --kind ValkeyInstance
operator-sdk create api --group cache --version v1alpha1 --kind GrafanaInstance
```

### 6.2 Estructura

```text
operator/
├── Dockerfile                       # FROM helm-operator + watches.yaml + helm-charts
├── watches.yaml                     # asocia cada CRD con su chart
├── helm-charts/
│   ├── valkeyinstance/              # chart encapsulado de Valkey
│   └── grafanainstance/             # chart encapsulado de Grafana
└── config/
    ├── crd/                         # CRDs generados
    ├── manager/manager.yaml         # Deployment del Operator
    ├── rbac/role.yaml               # permisos
    └── samples/                     # CRs: ValkeyInstance y GrafanaInstance
```

`watches.yaml` (generado por operator-sdk):

```yaml
- group: cache.mumnk8s.io
  version: v1alpha1
  kind: ValkeyInstance
  chart: helm-charts/valkeyinstance
- group: cache.mumnk8s.io
  version: v1alpha1
  kind: GrafanaInstance
  chart: helm-charts/grafanainstance
```

La imagen del Operator (`mumn-operator:v2`) contiene los charts y se publica en Zot. Se construye con `make docker-build docker-push IMG=$ZOT/mumn-operator:v2` y se despliega con `make deploy IMG=...`.

### 6.3 Lógica de reconciliación

1. Se crea un CR (por ejemplo `ValkeyInstance valkey`).
2. El Operator instala el chart como una *release* de Helm con el mismo nombre y el `spec` del CR como valores.
3. Si el CR cambia (por ejemplo `replicas: 2`), ejecuta un `helm upgrade`.
4. Periódicamente vuelve a reconciliar, y recrea lo que falte comparando con el chart.
5. Si se borra el CR, desinstala la release y elimina sus recursos.

**Prueba de reconciliación:** se borró el Service `valkey-svc` y el Operator lo **recreó en 4 segundos**, sin intervención manual.

![Reconciliación](./img/evidencia10.png)

### 6.4 ValkeyInstance

CRD `valkeyinstances.cache.mumnk8s.io`. Crea un **StatefulSet**, un **PVC** y un **Service**.

```yaml
apiVersion: cache.mumnk8s.io/v1alpha1
kind: ValkeyInstance
metadata:
  name: valkey
  namespace: mumnk8s
spec:
  image:
    repository: <ZOT>/valkey
    tag: "8"
  imagePullSecrets:
    - name: zot-creds
  persistence:
    enabled: true
    size: 1Gi
  replicas: 1
```

| Parámetro (`spec`) | Valor por defecto | Efecto |
|---|---|---|
| `image.repository`, `image.tag` | `valkey/valkey`, `8` | Imagen (en producción, desde Zot) |
| `persistence.enabled`, `persistence.size` | `true`, `1Gi` | PVC `data-valkey-N` montado en `/data` |
| `config.appendonly` | `yes` | Persistencia AOF |
| `config.maxmemory` | `256mb` | Tope de memoria |
| `config.maxmemoryPolicy` | `volatile-lru` | Al llegar al tope, descarta primero llaves con TTL |
| `replicas` | `1` | Número de instancias |
| `resources` | 100m/128Mi a 500m/512Mi | Requests y límites |

- **Persistencia:** `volumeClaimTemplates` crea un PVC por pod. Tras borrar `valkey-0`, el `DBSIZE` siguió en 12 y el TTL se conservó.
- **Réplicas:** el pod `valkey-0` es el primario. Los demás arrancan con `--replicaof valkey-svc 6379`. El Service `valkey-svc` apunta **solo** al pod 0, así que el consumer y Grafana siempre hablan con el primario. No hay conmutación automática por error (no se usa Sentinel).

![CRD y recursos del Operator](./img/evidencia8.png)
![Persistencia de Valkey](./img/evidencia9.png)

### 6.5 GrafanaInstance

CRD `grafanainstances.cache.mumnk8s.io`. Crea un Deployment, un Service de tipo `LoadBalancer`, un Secret de administración y dos ConfigMaps.

- **Plugin:** Grafana no lee Valkey de forma nativa, así que instala `redis-datasource` al arrancar (`GF_INSTALL_PLUGINS`). Es compatible con Valkey y necesita salida a internet desde el pod.
- **Provisioning del datasource** (ConfigMap `grafana-provisioning`): `uid: valkey`, URL `redis://valkey-svc:6379`.
- **Provisioning del dashboard** (ConfigMap `grafana-dashboards`): el JSON `mumn.json` va **dentro del chart**, así que el dashboard aparece solo al desplegar.
- **Contraseña de administración:** `spec.adminPassword`, que el chart escribe en el Secret `grafana-admin`. Grafana solo la lee al arrancar, por lo que un cambio exige reiniciar el pod.

| Panel | Consulta en Valkey |
|---|---|
| Máx. / Mín. aviones y barcos | `GET max:planes`, `min:planes`, `max:ships`, `min:ships` |
| Moda de aviones y barcos | `GET mode:planes`, `mode:ships` |
| Top 5 países (aviones y barcos) | `HGETALL top:planes` / `top:ships` + transformaciones: *Reduce (Series to rows)* y *Sort by* descendente |
| País asignado | Panel de texto: ESPAÑA (ESP) |
| Reportes recibidos | `HGET reports:count ESP` |
| Evolución temporal | `XRANGE ts:esp - +` (campos `planes` y `ships`) |
| # CARNET | Panel de texto: 202307716 |

Los totales del Top 5 son **sumas acumuladas** por país.

![Dashboard de Grafana](./img/evidencia7.png)

### 6.6 Permisos (RBAC)

El `ClusterRole` del Operator incluye, además de las reglas generadas, permisos de creación, lectura, actualización y borrado sobre `configmaps`, `secrets`, `services`, `persistentvolumeclaims`, `serviceaccounts`, `deployments` y `statefulsets`. El Deployment del Operator usa `imagePullSecrets: zot-creds` y un límite de memoria de 512Mi (el de 128Mi por defecto se queda corto para Helm).

---

## 7. Configuración de HPA

`k8s/31-hpa.yaml`

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-rust-hpa
  namespace: mumnk8s
spec:
  minReplicas: 1
  maxReplicas: 3
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-rust-deploy
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 30
```

- El HPA escala `api-rust-deploy` entre 1 y 3 réplicas cuando el uso promedio de CPU supera el **30 % del `request`**.
- El Deployment de Rust pide `cpu: 100m` y `memory: 64Mi`, con límites de `500m` y `256Mi`. Con ese `request`, el umbral equivale a **30 milicores por pod**. Los `requests` son la base del porcentaje; sin ellos el HPA no puede calcular el uso.
- GKE incluye `metrics-server`.
- Al subir la carga, el HPA añade réplicas con rapidez. Al bajarla, espera una ventana de estabilización por defecto de **5 minutos** antes de reducir.

**Resultado:** con 200 usuarios concurrentes el HPA pasó de 1 a **3 réplicas**, y unos 5 minutos después de terminar la carga volvió a 1.

> **Ajuste detectado en las pruebas:** con el límite inicial de memoria (128Mi), un pod de Rust fue terminado por `OOMKilled` bajo carga (usaba ~109Mi). Se subió el límite a 256Mi (ver sección 10.2).

![HPA al subir](./img/evidencia11.png)

---

## 8. Publicación/consumo de imágenes desde Zot

### 8.1 Registry

- **VM:** `zot-vm-diego` (GCP, `e2-small`, Ubuntu 24.04, disco de 30 GB), con IP reservada y fuera del clúster.
- **Zot:** binario completo (`zot-linux-amd64`, con búsqueda e interfaz web), como servicio de `systemd`, escuchando solo en `127.0.0.1:5000`. Autenticación con `htpasswd` (usuario `mumn`).
- **HTTPS:** **Caddy** hace de proxy inverso en los puertos 80 y 443, y obtiene un certificado válido de Let's Encrypt para `<IP>.sslip.io`. `sslip.io` resuelve ese nombre a la IP de la VM, así que no se necesita un dominio propio.
- **Firewall:** `allow-web-zot-diego` abre solo TCP 80 y 443.

### 8.2 Imágenes publicadas

| Repositorio | Origen |
|---|---|
| `api-rust`, `go-api`, `go-grpc-server`, `go-consumer` | Construidas con `scripts/build-push.sh` |
| `mumn-operator` | `make docker-build docker-push` |
| `rabbitmq`, `valkey`, `grafana` | Copiadas con `scripts/mirror-images.sh` |
| `locust-config` | **OCI Artifact** (8.4) |

> El `docker push` directo de la imagen oficial de Grafana fue rechazado por Zot con `manifest invalid`. Se solucionó publicándola de otra forma: **`docker build` con `FROM grafana/grafana:11.4.0`**.

![Catálogo de Zot](./img/evidencia6.png)

### 8.3 Descarga desde GKE

Los nodos descargan las imágenes con el Secret `zot-creds` (tipo `docker-registry`), declarado como `imagePullSecrets` en cada Deployment. Se crea una copia del secreto en cada namespace que lo necesita (`mumnk8s` y `mumn-operator-system`). Todos los YAML apuntan a `<IP>.sslip.io/<imagen>:<tag>`.

Comprobación de que las imágenes de los pods salen de Zot:

```bash
kubectl get pods -A -o custom-columns=NS:.metadata.namespace,POD:.metadata.name,IMAGENES:.spec.containers[*].image | grep -E "NS|mumnk8s|mumn-operator"
```

![Imágenes de los pods](./img/evidencia12.png)

Falco se instala con Helm desde su repositorio oficial, el requisito de Zot se cumple para todos los microservicios y componentes del proyecto.

### 8.4 OCI Artifact

El archivo de entrada de Locust se publica en Zot como artefacto OCI con **ORAS**, y Locust lo descarga antes de cada prueba:

```bash
# Publicar
oras login $ZOT -u mumn
oras push $ZOT/locust-config:v1 locust/config.json:application/json

# Usar (scripts/loadtest.sh lo hace en cada corrida)
oras pull $ZOT/locust-config:v1 -o /tmp/cfg
CONFIG_FILE=/tmp/cfg/config.json locust -f locust/locustfile.py --host http://$GW
```

`config.json` define la ruta, los países y los rangos (aviones 0 a 50, barcos 0 a 30). Modificarlo y publicar un nuevo tag cambia el comportamiento de la prueba sin tocar el código.

---

## 9. Falco

Falco se instaló con Helm en el namespace `falco`, como DaemonSet (un pod por nodo). En los nodos de GKE (Container-Optimized OS) se usa el driver **`modern_ebpf`**, que no necesita cabeceras del kernel.

```bash
helm repo add falcosecurity https://falcosecurity.github.io/charts
helm upgrade --install falco falcosecurity/falco -n falco --create-namespace \
  --version 9.2.1 -f falco/falco-values.yaml --wait --timeout 10m
```

`falco/falco-values.yaml`

```yaml
driver:
  kind: modern_ebpf
tty: true
falco:
  rule_matching: all
customRules:
  mumn-rules.yaml: |-
    # (contenido de la regla de abajo)
```

### 9.1 Regla personalizada

```yaml
- rule: Lectura de archivo sensible en contenedor (Proyecto3)
  desc: >
    Detecta la lectura de /etc/shadow o /app/secretos desde cualquier contenedor
    del cluster. Regla personalizada del Proyecto 3 (carnet 202307716).
  condition: >
    evt.type in (open, openat, openat2) and evt.is_open_read=true and fd.num>=0
    and fd.name in (/etc/shadow, /app/secretos)
    and container.id != host
  output: >
    ALERTA-MUMN archivo sensible leido (archivo=%fd.name proceso=%proc.name
    cmd=%proc.cmdline usuario=%user.name contenedor=%container.name
    id=%container.id imagen=%container.image.repository)
  priority: CRITICAL
  tags: [mumnk8s, proyecto3, filesystem]
```

- Vigila las aperturas de archivo para **lectura** (`evt.is_open_read`) cuyo nombre sea `/etc/shadow` o `/app/secretos`.
- `container.id != host` limita la regla a procesos dentro de contenedores.
- Prioridad `CRITICAL`. La salida lleva un marcador (`ALERTA-MUMN`) para localizarla con `grep`.

### 9.2 Orden de evaluación de las reglas: un problema detectado

Falco evalúa las reglas con el principio **"el primer match gana"**: si dos reglas coinciden con el mismo evento, solo se dispara la primera. Las reglas estándar se cargan antes que las de `rules.d`. En la primera prueba, `cat /etc/shadow` activó la regla estándar *Read sensitive file untrusted* y **tapó** la regla propia; solo `/app/secretos`, que la regla estándar no cubre, activó `ALERTA-MUMN`.

Se resolvió con `falco.rule_matching: all` (disponible desde Falco 0.36), que evalúa todas las reglas para cada evento. Con ello, un mismo `cat /etc/shadow` produce **dos alertas**: la estándar y la propia.

### 9.3 Simulación en vivo

Terminal 1, log de Falco en tiempo real (los dos pods, porque el contenedor atacado puede estar en cualquier nodo):

```bash
kubectl -n falco logs -f -l app.kubernetes.io/name=falco -c falco --max-log-requests 5 --prefix
```

Terminal 2, la intrusión simulada:

```bash
kubectl exec -n mumnk8s deploy/go-api-deploy -c api-rest -- cat /etc/shadow
kubectl exec -n mumnk8s deploy/go-api-deploy -c api-rest -- sh -c 'mkdir -p /app && echo clave > /app/secretos && cat /app/secretos'
```

Alerta obtenida para `/app/secretos` (solo salta con el `cat`, porque `echo >` escribe y la regla vigila lecturas):

```text
[pod/falco-tgtkz/falco] 05:47:50.682971411: Critical ALERTA-MUMN archivo sensible leido (archivo=/app/secretos proceso=cat cmd=cat /app/secretos usuario=root contenedor=api-rest id=1a735c06fbf4 imagen=35.239.210.169.sslip.io/go-api) container_id=1a735c06fbf4 container_name=api-rest container_image_repository=35.239.210.169.sslip.io/go-api container_image_tag=v1 k8s_pod_name=go-api-deploy-688984ffd6-kdd92 k8s_ns_name=mumnk8s
```

![Alertas de Falco en tiempo real](./img/evidencia13.png)

---

## 10. Pruebas realizadas y conclusiones

### 10.1 Metodología

- **Generador:** Locust 2.47 desde WSL (Ubuntu), contra la IP pública del Gateway, con la configuración del OCI Artifact.
- **Carga:** 200 usuarios, rampa de 20 usuarios/s, 4 minutos. Cada usuario espera entre 0,05 y 0,2 s entre peticiones.
- **Monitoreo** (`scripts/loadtest.sh`): una muestra cada 10 s del HPA, los pods, `kubectl top` y la cola de RabbitMQ.
- **Orden:** una corrida de calentamiento de 90 s para que Rust llegue a 3 réplicas, y después cada escenario. Antes de cada corrida se espera a que la cola esté vacía.
- **Escenarios:** A (1 réplica de Go, 1 de Valkey), B (2 de Go), C (2 de Valkey) y B2 (B repetida tras corregir la memoria de Rust).

### 10.2 Resultados

| Escenario | Peticiones | Fallos | Media (ms) | Mediana (ms) | p95 (ms) | p99 (ms) | Req/s |
|---|---:|---:|---:|---:|---:|---:|---:|
| Calentamiento | 12 033 | 0 | 1276 | 1400 | 2000 | 2300 | 142,7 |
| **A** · Go 1, Valkey 1 | 38 296 | 0 | 1097 | 1200 | 1800 | 2000 | 170,2 |
| **B** · Go 2, Valkey 1 | 54 045 | 696 (1,29 %) | 738 | 720 | 1300 | 1500 | 241,5 |
| **C** · Go 1, Valkey 2 | 39 728 | 1 | 1052 | 1000 | 1800 | 1900 | 177,7 |
| **B2** · Go 2, Valkey 1 | 60 051 | 0 | 649 | 620 | 1100 | 1400 | 250,4 |

Variación respecto a A:

| | Req/s | Mediana | p95 |
|---|---:|---:|---:|
| B | +42 % | −40 % | −28 % |
| C | +4 % | −17 % | 0 % |
| B2 | +47 % | −48 % | −39 % |

**Condiciones de cada corrida:**

- A, B y C: Falco sin instalar y límite de memoria de Rust de 128Mi.
- B2: Falco ya en ejecución (instalado un minuto antes) y límite de Rust de 256Mi. La sobrecarga de Falco perjudica a B2, y aun así superó a A.
- En B2, Locust avisó de que la CPU de la máquina generadora superó el 90 %. El techo de ~250 req/s podría deberse al generador y no al sistema.

![Resultados A](./img/10-locust-a.png)
![Resultados B2](./img/10-locust-b2.png)
![Resultados C](./img/10-locust-c.png)


Pruebas que fueron realizadas.

```bash
cd ~/202307716_LAB_SO1_2S2026/Proyecto3
./scripts/loadtest.sh calentamiento 200 20 60s

# A: 1 réplica Go, 1 de Valkey
./scripts/loadtest.sh a-go1-valkey1 200 20 4m

# B: 2 réplicas Go, 1 de Valkey
kubectl -n mumnk8s scale deploy go-server-deploy --replicas=2
kubectl -n mumnk8s wait --for=condition=Ready pod -l app=go-server --timeout=120s
sleep 90
kubectl -n mumnk8s logs -l app=go-server -c grpc-server --tail=1 --prefix     # debe haber líneas de ambos pods
./scripts/loadtest.sh b2-go2-valkey1 200 20 4m
kubectl -n mumnk8s scale deploy go-server-deploy --replicas=1

# C: 1 réplica Go, 2 de Valkey
kubectl -n mumnk8s scale deploy go-server-deploy --replicas=1
kubectl -n mumnk8s patch valkeyinstance valkey --type merge -p '{"spec":{"replicas":2}}'
kubectl -n mumnk8s wait --for=condition=Ready pod/valkey-1 --timeout=180s
./scripts/loadtest.sh c-go1-valkey2 200 20 4m

kubectl -n mumnk8s scale deploy go-server-deploy --replicas=1
kubectl -n mumnk8s patch valkeyinstance valkey --type merge -p '{"spec":{"replicas":1}}'
kubectl -n mumnk8s get pods -l app=valkey        # espera a que valkey-1 desaparezca
kubectl -n mumnk8s delete pvc data-valkey-1      # su disco quedó huérfano
```

**Incidente en B.** La prueba B tuvo 696 errores, todos `503 upstream connect error` (341 "connection termination" y 355 "remote connection failure"). Ese texto lo genera el balanceador de Google cuando un backend corta la conexión, no el código propio. En el monitoreo, Rust estuvo en 3 réplicas durante toda la corrida (no hubo escalado a mitad de la prueba), y al revisar los pods apareció uno con un reinicio por `OOMKilled`: bajo carga, Rust usaba ~109Mi de un límite de 128Mi. Se subió el límite a 256Mi y se repitió la prueba (B2): **0 errores**. Los 503 son consistentes con ese reinicio, y la repetición lo respalda; no se verificó contando eventos por pod.

### 10.3 HPA

Con 200 usuarios el HPA llevó a Rust de 1 a 3 réplicas (CPU ~32 % del `request` por pod, frente al objetivo del 30 %) y volvió a 1 unos 5 minutos después de terminar la carga. Los consumos observados durante la carga fueron ~30 m de CPU por pod de Rust, ~250 m en `go-api`, ~230 m en `go-server` y ~417 m en RabbitMQ (82 % de su límite de 500 m).

### 10.4 Resiliencia

- **Valkey caído:** sin pérdida de reportes (sección 5).
- **Pod de Valkey borrado:** el PVC conservó los datos (`DBSIZE` 12 antes y después) y el TTL (sección 6.4).
- **Recurso borrado:** el Operator recreó `valkey-svc` en 52 s (sección 6.3).
- **Réplica de Valkey:** con `replicas: 2`, `valkey-0` figura como `master` con un esclavo conectado, `valkey-1` como `slave` con el enlace activo (`master_link_status:up`), y `valkey-svc` mantiene un solo endpoint.

### 10.5 Falco

La regla propia detecta lecturas de `/etc/shadow` y `/app/secretos` en tiempo real (sección 9). Se descubrió que, por defecto, una regla estándar tapaba la propia, y se corrigió con `rule_matching: all`.

### 10.6 Conclusiones

1. **El flujo completo funciona de extremo a extremo en GKE.** Los reportes recorren Locust, Gateway, Rust, Go (REST, cliente gRPC, servidor y writer), RabbitMQ, consumer, Valkey y Grafana. En todas las corridas sin incidentes hubo 0 fallos, con más de 38 000 peticiones por prueba.
2. **Escalar Go de 1 a 2 réplicas mejora de forma clara el rendimiento.** B2 redujo la mediana un 48 % y el p95 un 39 % respecto a A, y subió el rendimiento un 47 %. Como cada usuario de Locust espera la respuesta antes de enviar otra petición, el rendimiento se deduce de la latencia: `usuarios / (latencia media + espera media)`. Para A da `200 / (1,097 + 0,125) ≈ 164` req/s (medido: 170) y para B2 `200 / (0,649 + 0,125) ≈ 258` (medido: 250). La mejora real es la **reducción de latencia**, y el aumento de req/s es consecuencia.
3. **El cuello de botella de A no estaba en la CPU de Rust.** Los pods de Rust consumían ~30 m, muy por debajo de su límite. Una hipótesis consistente con los datos, que no se midió directamente, es el límite de concurrencia del writer: cada uno de sus 4 workers espera la confirmación del broker antes de tomar otro trabajo, y duplicar los pods duplica esa capacidad. Una prueba pendiente sería subir `WORKERS` y repetir A.
4. **La segunda réplica de Valkey no aporta rendimiento de escritura.** C quedó dentro del ruido frente a A (+4 % en req/s, mismo p95): las escrituras van siempre al primario. Aporta una copia de los datos y capacidad de lectura, pero **no hay conmutación automática por error**.
5. **RabbitMQ cumple su función de desacople.** Con Valkey caído no se perdió ningún reporte, y la cola se vació sola al recuperarse.
6. **El HPA funciona y depende de los `requests`.** Un `request` de 100 m hace que el 30 % equivalga a solo 30 m, lo que permite escalar con una carga moderada. El monitoreo de las pruebas permitió descubrir el `OOMKilled` y corregirlo.
7. **El Operator hace reproducible el despliegue.** Valkey y Grafana se crean con un `kubectl apply` de un recurso, el dashboard queda aprovisionado desde el chart y el estado se repara solo (52 s en la prueba). El PVC conserva los datos.
8. **Falco detecta la intrusión simulada en tiempo real**, pero hay que revisar la precedencia entre reglas: con el comportamiento por defecto, una regla estándar silenció la regla propia.

### 10.7 Limitaciones y mejoras posibles

- **Una sola corrida por escenario**: no se midió la variabilidad entre repeticiones, así que diferencias pequeñas (como la de C) no son concluyentes.
- **El generador de carga fue un portátil.** La latencia incluye la red hasta GCP, y en B2 la CPU del portátil superó el 90 %. Lo ideal sería Locust en modo distribuido o desde una VM en GCP.
- **B2 y A no son idénticas en condiciones** (Falco y límite de memoria de Rust, ver 10.2).
- **RabbitMQ sin disco persistente**: los mensajes se marcan como persistentes, pero el pod usa almacenamiento efímero, así que se perderían si el pod se reprograma.
- **Credenciales:** la contraseña de Grafana viaja en el `spec` del CR. Lo adecuado sería leerla de un Secret existente. Grafana tampoco tiene volumen persistente.
- **Gateway solo HTTP**: sin TLS.
- **Valkey sin conmutación automática por error** (haría falta Sentinel o un operador de alta disponibilidad).
- **La API no autentica a los clientes.**

---

## 11. Guía de instalación paso a paso

Esta guía reproduce todo el entorno desde cero. Sustituye `<PROJECT_ID>`, `<ZOT_PASSWORD>` y las demás claves por valores propios, y **no las guardes en el repositorio**.

### 11.1 Requisitos locales (Windows con WSL2)

- WSL2 con Ubuntu 24.04 y Docker Desktop con la integración WSL activada. Trabaja dentro de `~/`, no en `/mnt/c`.
- Herramientas instaladas en Ubuntu: `git`, `make`, `jq`, **Go**, **Rust** (rustup), `protoc` con `protoc-gen-go` y `protoc-gen-go-grpc`, **gcloud** con `gke-gcloud-auth-plugin`, `kubectl`, **Helm**, **operator-sdk**, **ORAS** y **Locust** (en un entorno virtual de Python).
- Comprobación:

```bash
for c in go cargo protoc gcloud kubectl helm operator-sdk oras docker make jq git; do
  command -v $c >/dev/null && echo "OK $c" || echo "FALTA $c"
done
```

### 11.2 Proyecto de GCP

```bash
gcloud auth login --no-launch-browser
gcloud config set project <PROJECT_ID>
gcloud config set compute/region us-central1
gcloud config set compute/zone us-central1-a
gcloud services enable compute.googleapis.com container.googleapis.com
```

En un proyecto compartido, revisa primero qué recursos existen y usa nombres propios.

### 11.3 Registry Zot

```bash
gcloud compute addresses create zot-ip-diego --region us-central1
gcloud compute firewall-rules create allow-web-zot-diego \
  --allow tcp:80,tcp:443 --direction INGRESS --target-tags zot-diego
gcloud compute instances create zot-vm-diego \
  --machine-type e2-small --image-family ubuntu-2404-lts-amd64 --image-project ubuntu-os-cloud \
  --boot-disk-size 30GB --boot-disk-type pd-standard --address zot-ip-diego --tags zot-diego

ZOT_IP=$(gcloud compute addresses describe zot-ip-diego --region us-central1 --format='value(address)')
echo "export ZOT=${ZOT_IP}.sslip.io" >> ~/.bashrc && source ~/.bashrc
gcloud compute ssh zot-vm-diego --zone us-central1-a
```

Dentro de la VM:

```bash
sudo apt update && sudo apt install -y jq apache2-utils curl
URL=$(curl -s https://api.github.com/repos/project-zot/zot/releases/latest | jq -r '.assets[] | select(.name=="zot-linux-amd64") | .browser_download_url')
sudo curl -L -o /usr/local/bin/zot "$URL" && sudo chmod +x /usr/local/bin/zot
sudo useradd --system --no-create-home --shell /usr/sbin/nologin zot
sudo mkdir -p /etc/zot /var/lib/zot
sudo htpasswd -B -c /etc/zot/htpasswd mumn          # pide la contraseña, no se escribe en ningún archivo

sudo tee /etc/zot/config.json > /dev/null <<'EOF'
{
  "distSpecVersion": "1.1.0",
  "storage": { "rootDirectory": "/var/lib/zot" },
  "http": { "address": "127.0.0.1", "port": "5000",
            "auth": { "htpasswd": { "path": "/etc/zot/htpasswd" } } },
  "log": { "level": "info" },
  "extensions": { "search": { "enable": true }, "ui": { "enable": true } }
}
EOF
sudo chown -R zot:zot /etc/zot /var/lib/zot && sudo chmod 640 /etc/zot/htpasswd

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
sudo systemctl daemon-reload && sudo systemctl enable --now zot

# HTTPS con Caddy (certificado de Let's Encrypt para <IP>.sslip.io)
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --yes --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install -y caddy
IP=$(curl -s -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip)
printf '%s.sslip.io {\n    reverse_proxy 127.0.0.1:5000\n}\n' "$IP" | sudo tee /etc/caddy/Caddyfile
sudo systemctl reload caddy
curl -i https://$IP.sslip.io/v2/          # debe responder 401 sobre HTTPS válido
```

### 11.4 Imágenes y OCI Artifact

```bash
docker login $ZOT -u mumn
./scripts/build-push.sh api-rust go-api go-grpc-server go-consumer
./scripts/mirror-images.sh                 # rabbitmq, valkey y grafana
oras login $ZOT -u mumn
oras push $ZOT/locust-config:v1 locust/config.json:application/json
oras repo ls $ZOT
```

### 11.5 Clúster GKE

```bash
gcloud container clusters create mumn-diego \
  --zone us-central1-a --num-nodes 2 --machine-type e2-standard-2 \
  --disk-type pd-standard --disk-size 35 --gateway-api=standard
gcloud container clusters get-credentials mumn-diego --zone us-central1-a
kubectl config current-context            # debe terminar en _mumn-diego
kubectl get nodes -o wide
```

### 11.6 Namespace y secretos

```bash
kubectl apply -f k8s/01-namespace.yaml
RMQ_PASS=$(openssl rand -hex 16)
kubectl -n mumnk8s create secret generic rabbitmq-creds \
  --from-literal=user=mumn --from-literal=pass=$RMQ_PASS \
  --from-literal=url="amqp://mumn:$RMQ_PASS@rabbitmq:5672/"
unset RMQ_PASS
read -rsp "Contraseña de Zot: " ZOT_PASS; echo
kubectl -n mumnk8s create secret docker-registry zot-creds \
  --docker-server=$ZOT --docker-username=mumn --docker-password="$ZOT_PASS"
```

### 11.7 Aplicación

```bash
sed -i "s/ZOT_HOST/$ZOT/g" k8s/*.yaml operator/config/samples/*.yaml
kubectl apply -f k8s/
kubectl -n mumnk8s get pods
kubectl get gateway,httproute -n mumnk8s          # el Gateway tarda unos minutos en obtener IP
```

### 11.8 Operator, Valkey y Grafana

```bash
cd operator
export IMG=$ZOT/mumn-operator:v2
make docker-build docker-push IMG=$IMG

kubectl create namespace mumn-operator-system
kubectl -n mumn-operator-system create secret docker-registry zot-creds \
  --docker-server=$ZOT --docker-username=mumn --docker-password="$ZOT_PASS"
make deploy IMG=$IMG

kubectl apply -f config/samples/cache_v1alpha1_valkeyinstance.yaml
GF_PASS=$(openssl rand -base64 12 | tr -d '/+=')       # guárdala fuera del repositorio
sed "s/CAMBIA-ESTA-CLAVE/$GF_PASS/" config/samples/cache_v1alpha1_grafanainstance.yaml | kubectl apply -f -
kubectl -n mumnk8s get valkeyinstances,grafanainstances,pods,pvc,svc
kubectl -n mumnk8s get svc grafana                      # EXTERNAL-IP de Grafana
```

### 11.9 Falco

```bash
helm repo add falcosecurity https://falcosecurity.github.io/charts && helm repo update
helm upgrade --install falco falcosecurity/falco -n falco --create-namespace \
  --version 9.2.1 -f falco/falco-values.yaml --wait --timeout 10m
kubectl -n falco get pods
```

### 11.10 Pruebas de carga

```bash
python3 -m venv ~/locust-env && source ~/locust-env/bin/activate && pip install locust && deactivate
./scripts/loadtest.sh calentamiento 200 20 90s
./scripts/loadtest.sh a-go1-valkey1 200 20 4m
```

### 11.11 Verificación

```bash
./scripts/preflight.sh          # contexto, nodos, pods, Gateway, Grafana, Valkey, cola, Zot, imágenes y Falco
./scripts/evidencias.sh         # guarda la salida de los comandos en docs/evidencias/
```

### 11.12 Eliminación de recursos

En un proyecto compartido, borra **solo** lo propio (nombres con `-diego`):

```bash
gcloud container clusters delete mumn-diego --zone us-central1-a
gcloud compute instances delete zot-vm-diego --zone us-central1-a
gcloud compute addresses delete zot-ip-diego --region us-central1
gcloud compute firewall-rules delete allow-web-zot-diego
```

Para pausar sin borrar, reduce los nodos a cero (se conservan el Gateway, su IP y los discos, y el clúster sigue generando su tarifa de gestión):

```bash
gcloud container clusters resize mumn-diego --node-pool default-pool --num-nodes 0 --zone us-central1-a
gcloud container clusters resize mumn-diego --node-pool default-pool --num-nodes 2 --zone us-central1-a
```
