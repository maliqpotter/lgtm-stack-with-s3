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
   
2. **Start the Stack**:
   ```bash
   docker-compose up -d
   ```

Access endpoints:
- Grafana: http://localhost:3000 (admin / [Your GF_SECURITY_ADMIN_PASSWORD])
- MinIO: http://localhost:9001 ([Your MINIO_ROOT_USER] / [Your MINIO_ROOT_PASSWORD])

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
