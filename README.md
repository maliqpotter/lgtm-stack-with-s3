# LGTM Monitoring Stack

Full observability stack powered by Grafana's **LGTM** ecosystem with S3-compatible object storage.

| Component | Role | Port |
|-----------|------|------|
| **Loki** | Log aggregation | `3100` |
| **Grafana** | Dashboards & visualization | `3000` |
| **Tempo** | Distributed tracing | `3200` |
| **Mimir** | Metrics storage (Prometheus-compatible) | `9009` |
| **Alloy** | Unified telemetry collector (OTLP) | `12345` (UI) |
| **MinIO** | S3 object storage backend | `9000` (API) / `9001` (Console) |

## Architecture

```
┌──────────────────────┐
│     Your App(s)      │
│  (OTLP SDK enabled)  │
└──────┬───────────────┘
       │ OTLP gRPC/HTTP
       ▼
┌──────────────┐
│    Alloy     │──── traces ────▶ Tempo  ──┐
│  (Collector) │──── metrics ───▶ Mimir  ──┤──▶ MinIO (S3)
│              │──── logs ──────▶ Loki   ──┘
└──────────────┘
       │ Docker logs
       └────────────────────────▶ Loki
                                    │
                                    ▼
                              ┌──────────┐
                              │ Grafana  │
                              └──────────┘
```

## Getting Started

### Prerequisites

- **Docker Engine** ≥ 20.10
- **Docker Compose** ≥ 2.0 (plugin) or `docker-compose` v2+
- Minimum **4 GB RAM** available for Docker

### Step 1 — Clone & Navigate

```bash
git clone <repository-url>
cd LGTM-Monitoring
```

### Step 2 — Start the Stack

```bash
docker compose up -d
```

### Step 3 — Verify All Services Are Healthy

Wait a few seconds for services to initialize, then check:

```bash
# Check all container statuses
docker compose ps

# Individual health checks
curl -s http://localhost:9000/minio/health/live && echo " ✅ MinIO"
curl -s http://localhost:3100/ready && echo " ✅ Loki"
curl -s http://localhost:3200/ready && echo " ✅ Tempo"
curl -s http://localhost:9009/ready && echo " ✅ Mimir"
curl -s http://localhost:3000/api/health && echo " ✅ Grafana"
```

### Step 4 — Verify MinIO Buckets

Open [http://localhost:9001](http://localhost:9001) (login: `minioadmin` / `minioadmin123`).
Verify these buckets exist: `loki-data`, `tempo-data`, `mimir-data`.

### Step 5 — Open Grafana

Open [http://localhost:3000](http://localhost:3000) (login: `admin` / `admin`).
Verify datasources are provisioned under **Connections → Data sources**.

### Step 6 — Verification with Test App

A sample Go application is provided in the `test-app/` directory to verify the stack.

1.  **Start the Test App**:
    ```bash
    docker compose -f test-app/docker-compose.yaml up -d
    ```

2.  **Generate Telemetry**:
    ```bash
    # Health check
    curl http://localhost:8080/health

    # Create a document (generates a trace with child spans and logs)
    curl -X POST http://localhost:8080/api/documents \
         -H "Content-Type: application/json" \
         -d '{"title": "Test Observability", "content": "LGTM stack is working!"}'

    # List documents
    curl http://localhost:8080/api/documents
    ```

3.  **Check in Grafana**:
    - **Traces**: Explore -> Tempo -> Search for `test-api`.
    - **Logs**: Explore -> Loki -> `{service_name="test-api"}`.
    - **Metrics**: Explore -> Mimir -> `http_requests_total`.

## Access Points

| Service | URL | Credentials |
|---------|-----|-------------|
| Grafana | http://localhost:3000 | `admin` / `admin` |
| MinIO Console | http://localhost:9001 | `minioadmin` / `minioadmin123` |
| Alloy UI | http://localhost:12345 | — |
| Test API | http://localhost:8080 | — |

## Cleanup

```bash
# Stop monitoring stack
docker compose down

# Stop test app
docker compose -f test-app/docker-compose.yaml down

# Remove all volumes (caution: deletes all data)
docker compose down -v
```
