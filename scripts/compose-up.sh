#!/usr/bin/env bash
# Bring up Docker Compose with a reproducible Scythe tree: initialize the submodule so the image
# COPY includes the same commit as this repo (see Dockerfile for clone fallback when submodule absent).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
if ! git submodule update --init --recursive; then
	echo "Warning: git submodule update failed (not a git checkout?). Docker build may clone Scythe from GitHub instead." >&2
fi
exec docker compose up --build "$@"
