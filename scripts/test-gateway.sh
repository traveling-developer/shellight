#!/usr/bin/env bash
# Local OpenShell test gateway for development, running on the nested rootless
# Podman inside the devcontainer. Follows the Podman section of
# https://docs.nvidia.com/openshell/v0.1.7/how-it-works/gateways/container-deployment.md
#
# State, mTLS bundle and CLI config live in .dev/test-gateway (git ignored).

set -euo pipefail

usage() {
  cat <<EOF
Usage: $0 <command>
  up       start Podman service and gateway, register the CLI
  down     delete all sandboxes and remove the gateway container
           (keeps .dev/test-gateway; delete it for a full reset)
  cli ...  run an openshell command against the test gateway
EOF
  exit 1
}

# ghcr.io/nvidia/openshell/gateway:0.1.7, multi-arch index digest (checked 2026-11-07)
GATEWAY_IMAGE="ghcr.io/nvidia/openshell/gateway:0.1.7@sha256:9b94165a3dbcf6d454b822e5b21f371f168c44155c8fa1d6dd55ba091a041d1f"
CONTAINER_NAME="shellight-test-gateway"
GATEWAY_NAME="shellight-test"
# Same port inside and outside: sandbox supervisors call back to 127.0.0.1:<port>.
PORT="17690"

DATA_DIR="$(cd "$(dirname "$0")/.." && pwd)/.dev/test-gateway"
TLS_DIR="$DATA_DIR/tls"
CLI_CONFIG="$DATA_DIR/cli-config"

log() { printf '==> %s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

# openshell CLI with its own config dir, isolated from ~/.config/openshell.
os() {
  XDG_CONFIG_HOME="$CLI_CONFIG" \
  OPENSHELL_LOCAL_TLS_DIR="$TLS_DIR" \
  OPENSHELL_GATEWAY="$GATEWAY_NAME" \
    openshell "$@"
}

# The gateway's Podman driver needs the Podman API socket. There is no
# systemd in the devcontainer, so run the service in the background.
ensure_podman_service() {
  PODMAN_SOCK="$(podman info --format '{{.Host.RemoteSocket.Path}}')"
  podman --url "unix://$PODMAN_SOCK" info >/dev/null 2>&1 && return
  log "starting Podman API service on $PODMAN_SOCK"
  mkdir -p "$(dirname "$PODMAN_SOCK")"
  nohup podman system service --time=0 "unix://$PODMAN_SOCK" >"$DATA_DIR/podman-service.log" 2>&1 &
  for _ in $(seq 20); do
    [ -S "$PODMAN_SOCK" ] && return
    sleep 0.5
  done
  cat "$DATA_DIR/podman-service.log" >&2
  die "Podman API service did not start"
}

generate_certs() {
  [ -f "$TLS_DIR/client/tls.crt" ] && return
  log "generating mTLS bundle in $TLS_DIR"
  podman run --rm \
    --userns=keep-id:uid=1000,gid=1000 \
    -v "$DATA_DIR:$DATA_DIR" \
    -e HOME="$DATA_DIR" \
    "$GATEWAY_IMAGE" \
    generate-certs --output-dir "$TLS_DIR"
}

start_gateway() {
  if podman container exists "$CONTAINER_NAME"; then
    podman start "$CONTAINER_NAME" >/dev/null
    return
  fi
  log "starting $CONTAINER_NAME on 127.0.0.1:$PORT"
  podman run -d \
    --name "$CONTAINER_NAME" \
    --userns=keep-id:uid=1000,gid=1000 \
    -p "127.0.0.1:$PORT:$PORT" \
    -v "$DATA_DIR:$DATA_DIR" \
    -v "$PODMAN_SOCK:/var/run/podman.sock" \
    -e HOME="$DATA_DIR" \
    -e XDG_DATA_HOME="$DATA_DIR" \
    -e OPENSHELL_LOCAL_TLS_DIR="$TLS_DIR" \
    -e OPENSHELL_COMPUTE_DRIVER=podman \
    -e OPENSHELL_PODMAN_SOCKET=/var/run/podman.sock \
    -e "OPENSHELL_DB_URL=sqlite:$DATA_DIR/gateway.db" \
    "$GATEWAY_IMAGE" \
    --bind-address 0.0.0.0 --port "$PORT" >/dev/null
}

register_cli() {
  [ -f "$CLI_CONFIG/openshell/gateways/$GATEWAY_NAME/metadata.json" ] && return
  log "registering $GATEWAY_NAME in $CLI_CONFIG"
  mkdir -p "$CLI_CONFIG"
  os gateway add "https://127.0.0.1:$PORT" --local --name "$GATEWAY_NAME"
}

wait_healthy() {
  log "waiting for gateway"
  for _ in $(seq 30); do
    # `sandbox list` needs a working mTLS connection; `status` may exit 0 while disconnected.
    if os sandbox list >/dev/null 2>&1; then
      os status
      return
    fi
    sleep 2
  done
  podman logs --tail 50 "$CONTAINER_NAME" >&2 || true
  die "gateway did not become healthy within 60 s"
}

cmd_up() {
  [ -c /dev/net/tun ] || die "/dev/net/tun missing (see runArgs in devcontainer.json)"
  mkdir -p "$DATA_DIR"
  ensure_podman_service
  generate_certs
  start_gateway
  register_cli
  wait_healthy
}

cmd_down() {
  if podman container exists "$CONTAINER_NAME"; then
    # Sandbox containers outlive the gateway container, so delete them first
    # and wait until the gateway has cleaned them up.
    log "deleting all sandboxes"
    os sandbox delete --all >/dev/null || true
    for _ in $(seq 30); do
      os sandbox list -o json 2>/dev/null | jq -e '.sandboxes | length == 0' >/dev/null && break
      sleep 1
    done
    log "removing $CONTAINER_NAME"
    podman rm -f "$CONTAINER_NAME" >/dev/null
  fi
}

case "${1:-}" in
  up) cmd_up ;;
  down) cmd_down ;;
  cli) shift; os "$@" ;;
  *) usage ;;
esac
