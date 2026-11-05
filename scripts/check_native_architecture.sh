#!/usr/bin/env bash
set -euo pipefail

normalize_architecture() {
    case "$1" in
        x86_64 | amd64 | x64) echo amd64 ;;
        aarch64 | arm64) echo arm64 ;;
        *) echo "Unsupported architecture: $1" >&2; return 1 ;;
    esac
}

required_architecture=$(normalize_architecture "${1:?Expected architecture is required}")
host_architecture=$(normalize_architecture "$(uname -m)")
engine_architecture=$(normalize_architecture "$(docker info --format '{{.Architecture}}')")
echo "Native verification: host=${host_architecture}, engine=${engine_architecture}, kernel=$(uname -r)"
if [[ "$host_architecture" != "$required_architecture" || "$engine_architecture" != "$required_architecture" ]]; then
    echo "Native architecture mismatch: expected ${required_architecture}" >&2
    exit 1
fi

if [[ -n "${2:-}" ]]; then
    image_platform=$(docker image inspect --format '{{.Os}}/{{.Architecture}}' "$2")
    if [[ "$image_platform" != "linux/${required_architecture}" ]]; then
        echo "Candidate architecture mismatch: $2 is ${image_platform}, expected linux/${required_architecture}" >&2
        exit 1
    fi
fi
