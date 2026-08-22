#!/bin/bash
# ============================================================
# Generate Self-Signed SSL Certificate for Grafana
# Usage: ./generate-cert.sh [IP_ADDRESS]
# Example: ./generate-cert.sh 192.172.25.102
# ============================================================

set -e

IP="${1:-192.172.25.102}"
CERT_DIR="$(dirname "$0")/certs"

mkdir -p "$CERT_DIR"

echo "🔐 Generating self-signed SSL certificate for IP: $IP"

openssl req -x509 -nodes -newkey rsa:2048 \
  -keyout "$CERT_DIR/grafana.key" \
  -out "$CERT_DIR/grafana.crt" \
  -days 365 \
  -subj "/CN=Grafana/O=Monitoring/C=ID" \
  -addext "subjectAltName=IP:${IP},IP:127.0.0.1,DNS:localhost,DNS:grafana"

chmod 644 "$CERT_DIR/grafana.crt"
chmod 644 "$CERT_DIR/grafana.key"

echo "✅ Certificate generated successfully!"
echo "   📄 Certificate: $CERT_DIR/grafana.crt"
echo "   🔑 Key:         $CERT_DIR/grafana.key"
echo ""
echo "To enable HTTPS, uncomment the HTTPS section in docker-compose.yaml"
echo "and run: docker compose up -d grafana"
