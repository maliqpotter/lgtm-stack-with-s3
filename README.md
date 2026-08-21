# LGTM Monitoring Stack (Air-Gapped Kubernetes Edition)

Full observability stack powered by Grafana's **LGTM** ecosystem with S3-compatible object storage, specifically designed to monitor an **Offline/Air-Gapped Kubernetes Cluster**.

| Component | Role |
|-----------|------|
| **Loki** | Log aggregation |
| **Grafana** | Dashboards & visualization |
| **Tempo** | Distributed tracing |
| **Mimir** | Metrics storage (Prometheus-compatible) |
| **Alloy** | Unified telemetry collector (OTLP) |
| **MinIO** | S3 object storage backend |

## Architecture

This setup separates the Monitoring Server (running Docker Compose) from your Application Environment (Kubernetes).

```text
[ KUBERNETES CLUSTER (Offline) ]                    [ MONITORING SERVER (Docker Compose) ]
                                                            
+------------------+                                        +--------------------------------+
| - CPU/RAM Node   |  Di-scrape   +---------------+         |        +--> Mimir (Metrics)    |
| - Kube Metrics   | -----------> | Grafana Alloy |  Push   | +-------+                      |
| - Pod Logs       |              | (DaemonSet)   | ------> | | Alloy |-> Loki  (Logs)       |
| - App Traces     | -----------> +---------------+  OTLP   | +-------+                      |
+------------------+                                        |        +--> Tempo (Traces)     |
                                                            |           ⬇️                   |
                                                            |        Grafana (Dashboard)     |
                                                            +--------------------------------+
```

## Getting Started

### Phase 1: Start the Monitoring Server (Docker Compose)
This server acts as the central storage and visualization hub.

1. **Configure Environment Variables**:
   Copy the example environment file and adjust the credentials if necessary:
   ```bash
   cp .env.example .env
   ```

2. **Setup Network**:
   Jalankan script setup untuk membuat Docker network dengan static IP sebelum menjalankan compose:
   ```bash
   bash setup.sh
   ```

3. **Start the Stack**:
   ```bash
   docker compose up -d
   ```

Access endpoints:
- Grafana: http://\<IP_SERVER\>:3000 (admin / [Your GF_SECURITY_ADMIN_PASSWORD])

## Network Architecture

Semua container menggunakan static IP pada subnet `177.20.0.0/28` untuk keamanan dan kemudahan akses antar service di server yang sama.

| Container | IP Address | Port |
|---|---|---|
| Gateway | `177.20.0.1` | — |
| MinIO | `177.20.0.2` | `9000` (S3 API), `9001` (Console) |
| Loki | `177.20.0.3` | `3100` |
| Tempo | `177.20.0.4` | `3200` (HTTP), `4317` (OTLP gRPC) |
| Mimir | `177.20.0.5` | `9009` |
| Alertmanager | `177.20.0.6` | `9093` |
| Alloy | `177.20.0.7` | `12345` (UI), `4317` (OTLP gRPC), `4318` (OTLP HTTP) |
| Node Exporter | `177.20.0.8` | `9100` |
| cAdvisor | `177.20.0.9` | `8080` |
| Blackbox Exporter | `177.20.0.10` | `9115` |
| Grafana | `177.20.0.11` | `3000` |

> **Note**: Hanya port Grafana (`3000`) yang di-expose ke publik. Service lain hanya bisa diakses via IP internal dari server yang sama.

### Menghubungkan Aplikasi di Server yang Sama

Jika aplikasi Anda berjalan di Docker pada server yang sama, tambahkan network `monitoring` ke compose aplikasi Anda:

```yaml
services:
  my-app:
    image: my-app:latest
    environment:
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://177.20.0.7:4317
    networks:
      - monitoring

networks:
  monitoring:
    external: true
```

Jika aplikasi berjalan langsung di host (non-Docker), cukup arahkan ke `177.20.0.7:4317`.

## HTTPS (Self-Signed Certificate)

Grafana mendukung HTTPS menggunakan self-signed certificate. Secara default fitur ini **dinonaktifkan**.

### Cara Mengaktifkan

1. **Generate certificate**:
   ```bash
   bash config/grafana/generate-cert.sh <IP_SERVER>
   # Contoh: bash config/grafana/generate-cert.sh 192.168.1.100
   ```

2. **Uncomment konfigurasi HTTPS** di `docker-compose.yaml` pada bagian service `grafana`:
   ```yaml
   environment:
     - GF_SERVER_PROTOCOL=https
     - GF_SERVER_CERT_FILE=/etc/grafana/certs/grafana.crt
     - GF_SERVER_CERT_KEY=/etc/grafana/certs/grafana.key
   volumes:
     - ./config/grafana/certs:/etc/grafana/certs:ro
   ```

3. **Restart Grafana**:
   ```bash
   docker compose restart grafana
   ```

4. Akses via `https://<IP_SERVER>:3000` (browser akan menampilkan warning self-signed, klik "Proceed").

> **Note**: File certificate (`config/grafana/certs/`) sudah masuk `.gitignore` dan tidak akan ter-push ke repository.

### Phase 2: Sync Images for Offline Kubernetes
Since your Kubernetes cluster is air-gapped, you must sync the required images to your private container registry (e.g., GHCR, Harbor, Docker Hub).

1. Ensure you are logged in to your registry: `docker login ghcr.io`
2. Pull the required images and push them to your private registry. (You may create a sync script for automation)


### Phase 3: Deploy to Kubernetes
Move the generated YAML files to your Kubernetes cluster and apply them.

**1. Kube State Metrics** (Collects K8s object states):
```bash
kubectl apply -f kubernetes-config/kube-state-metrics.yaml
```

**2. Node Exporter** (Collects physical Node/OS hardware metrics like CPU, RAM, Disk):
```bash
kubectl apply -f kubernetes-config/node-exporter.yaml
```

**3. Grafana Alloy** (The Agent/Collector):
*Before applying, ensure you update `IP_SERVER_MONITORING` in `kubernetes-config/alloy-k8s.yaml` to point to the IP of your Docker Compose server.*
```bash
kubectl apply -f kubernetes-config/alloy-k8s.yaml
```

## Test Application

A sample Go application is provided in the `test-app/` directory to verify the full stack inside Kubernetes. This app automatically generates background jobs, errors, and traces every 3 seconds.

1. **Build and push to your registry**:
```bash
cd test-app
docker build -t ghcr.io/your-username/test-api:latest .
docker push ghcr.io/your-username/test-api:latest
```

2. **Deploy to K8s**:
*Update `IP_SERVER_MONITORING` in `test-app-k8s.yaml` before applying.*
```bash
kubectl apply -f test-app-k8s.yaml
```

3. **Check in Grafana**:
   - **Traces**: Explore -> Tempo -> Search for `test-api`.
   - **Logs**: Explore -> Loki -> `{app="test-app"}` or `{service_name="test-api"}`.
   - **Metrics**: Explore -> Mimir -> `http_requests_total`.
   - **Node Dashboard**: Import Dashboard ID `1860` to view hardware metrics from Node Exporter.

## Recommended Grafana Dashboards

Untuk mempermudah monitoring, berikut adalah list Grafana Dashboards yang direkomendasikan dan kompatibel dengan setup ini:

- **Domain Monitoring**: [Symphony Domain (ID: 23131)](https://grafana.com/grafana/dashboards/23131-symphony-domain/)
- **Docker cAdvisor**: [cAdvisor Exporter Docker Containers Overview (ID: 21743)](https://grafana.com/grafana/dashboards/21743-cadvisor-exporter-docker-containers-overview/)
- **Node Exporter**: [Node Exporter Dashboard (ID: 24784)](https://grafana.com/grafana/dashboards/24784-node-exporter-dashboard-20240520/)
- **Kubernetes Dashboard**: [dotdc/grafana-dashboards-kubernetes](https://github.com/dotdc/grafana-dashboards-kubernetes)

