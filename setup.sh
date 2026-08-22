#!/bin/bash
# ============================================================
# Setup Script — LGTM Monitoring Stack
# Run this script BEFORE docker compose up -d
# ============================================================

set -e

NETWORK_NAME="monitoring"
SUBNET="172.20.0.0/28"
GATEWAY="172.20.0.1"

echo "🔧 LGTM Monitoring Stack — Setup"
echo "================================="
echo ""

# 1. Create Docker Network (if not exists)
if docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
  echo "⚠️  Network '$NETWORK_NAME' already exists."

  # Check if subnet matches
  EXISTING_SUBNET=$(docker network inspect "$NETWORK_NAME" --format '{{range .IPAM.Config}}{{.Subnet}}{{end}}')
  if [ "$EXISTING_SUBNET" != "$SUBNET" ]; then
    echo "❌ Subnet mismatch! (current: $EXISTING_SUBNET, required: $SUBNET)"
    echo "   Removing old network and recreating..."

    # Stop all connected containers
    CONTAINERS=$(docker network inspect "$NETWORK_NAME" --format '{{range .Containers}}{{.Name}} {{end}}')
    if [ -n "$CONTAINERS" ]; then
      echo "   Stopping containers: $CONTAINERS"
      docker compose down 2>/dev/null || true
    fi

    docker network rm "$NETWORK_NAME"
    docker network create \
      --driver bridge \
      --subnet "$SUBNET" \
      --gateway "$GATEWAY" \
      "$NETWORK_NAME"
    echo "✅ Network '$NETWORK_NAME' recreated with subnet $SUBNET"
  else
    echo "✅ Subnet already matches ($SUBNET). No changes needed."
  fi
else
  docker network create \
    --driver bridge \
    --subnet "$SUBNET" \
    --gateway "$GATEWAY" \
    "$NETWORK_NAME"
  echo "✅ Network '$NETWORK_NAME' created with subnet $SUBNET"
fi

echo ""
echo "📋 IP Address Map:"
echo "   ┌─────────────────────┬──────────────────┐"
echo "   │ Container           │ IP Address       │"
echo "   ├─────────────────────┼──────────────────┤"
echo "   │ Gateway             │ 172.20.0.1       │"
echo "   │ MinIO               │ 172.20.0.2       │"
echo "   │ Loki                │ 172.20.0.3       │"
echo "   │ Tempo               │ 172.20.0.4       │"
echo "   │ Mimir               │ 172.20.0.5       │"
echo "   │ Alertmanager        │ 172.20.0.6       │"
echo "   │ Alloy               │ 172.20.0.7       │"
echo "   │ Node Exporter       │ 172.20.0.8       │"
echo "   │ cAdvisor            │ 172.20.0.9       │"
echo "   │ Blackbox Exporter   │ 172.20.0.10      │"
echo "   │ Grafana             │ 172.20.0.11      │"
echo "   └─────────────────────┴──────────────────┘"
echo ""
echo "🚀 Network ready! Run: docker compose up -d"
