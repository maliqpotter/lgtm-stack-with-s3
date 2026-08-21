#!/bin/bash
# ============================================================
# Setup Script — LGTM Monitoring Stack
# Jalankan script ini SEBELUM docker compose up -d
# ============================================================

set -e

NETWORK_NAME="monitoring"
SUBNET="177.20.0.0/28"
GATEWAY="177.20.0.1"

echo "🔧 LGTM Monitoring Stack — Setup"
echo "================================="
echo ""

# 1. Buat Docker Network (jika belum ada)
if docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
  echo "⚠️  Network '$NETWORK_NAME' sudah ada."

  # Cek apakah subnet-nya sudah sesuai
  EXISTING_SUBNET=$(docker network inspect "$NETWORK_NAME" --format '{{range .IPAM.Config}}{{.Subnet}}{{end}}')
  if [ "$EXISTING_SUBNET" != "$SUBNET" ]; then
    echo "❌ Subnet tidak sesuai! (sekarang: $EXISTING_SUBNET, dibutuhkan: $SUBNET)"
    echo "   Menghapus network lama dan membuat ulang..."

    # Hentikan semua container yang masih terhubung
    CONTAINERS=$(docker network inspect "$NETWORK_NAME" --format '{{range .Containers}}{{.Name}} {{end}}')
    if [ -n "$CONTAINERS" ]; then
      echo "   Menghentikan container: $CONTAINERS"
      docker compose down 2>/dev/null || true
    fi

    docker network rm "$NETWORK_NAME"
    docker network create \
      --driver bridge \
      --subnet "$SUBNET" \
      --gateway "$GATEWAY" \
      "$NETWORK_NAME"
    echo "✅ Network '$NETWORK_NAME' berhasil dibuat ulang dengan subnet $SUBNET"
  else
    echo "✅ Subnet sudah sesuai ($SUBNET). Tidak perlu perubahan."
  fi
else
  docker network create \
    --driver bridge \
    --subnet "$SUBNET" \
    --gateway "$GATEWAY" \
    "$NETWORK_NAME"
  echo "✅ Network '$NETWORK_NAME' berhasil dibuat dengan subnet $SUBNET"
fi

echo ""
echo "📋 IP Address Map:"
echo "   ┌─────────────────────┬──────────────────┐"
echo "   │ Container           │ IP Address       │"
echo "   ├─────────────────────┼──────────────────┤"
echo "   │ Gateway             │ 177.20.0.1       │"
echo "   │ MinIO               │ 177.20.0.2       │"
echo "   │ Loki                │ 177.20.0.3       │"
echo "   │ Tempo               │ 177.20.0.4       │"
echo "   │ Mimir               │ 177.20.0.5       │"
echo "   │ Alertmanager        │ 177.20.0.6       │"
echo "   │ Alloy               │ 177.20.0.7       │"
echo "   │ Node Exporter       │ 177.20.0.8       │"
echo "   │ cAdvisor            │ 177.20.0.9       │"
echo "   │ Blackbox Exporter   │ 177.20.0.10      │"
echo "   │ Grafana             │ 177.20.0.11      │"
echo "   └─────────────────────┴──────────────────┘"
echo ""
echo "🚀 Network siap! Jalankan: docker compose up -d"
