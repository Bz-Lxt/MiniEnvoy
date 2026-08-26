#!/bin/sh
# MVP 10k/50k harness. Records environment; does not invent results.
set -eu
OUT="${1:-bench/results/mvp-$(date +%Y%m%d-%H%M%S).txt}"
mkdir -p "$(dirname "$OUT")"
{
  echo "time=$(date '+%Y-%m-%d %H:%M:%S %z')"
  echo "uname=$(uname -a)"
  echo "go=$(go version 2>/dev/null || true)"
  echo "ulimit_n=$(ulimit -n)"
  echo "target=${TARGET:-127.0.0.1:31880}"
  echo "NOTE: Docker Desktop / macOS results are functional only."
} > "$OUT"
echo "wrote $OUT"
