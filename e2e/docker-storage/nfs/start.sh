#!/usr/bin/env bash
set -euo pipefail

mount -t nfsd nfsd /proc/fs/nfsd
rpcbind -w
rpc.nfsd --no-nfs-version 3 --nfs-version 4 --grace-time 10
exportfs -rav
rpc.mountd --foreground --no-nfs-version 2 --no-nfs-version 3 &
mountd_pid=$!

cleanup() {
    kill "$mountd_pid" 2>/dev/null || true
    rpc.nfsd 0
    exportfs -au
}
trap cleanup EXIT
trap 'exit 0' TERM INT

rpcinfo -t localhost nfs 4
echo 'Startup successful.'
wait "$mountd_pid"
