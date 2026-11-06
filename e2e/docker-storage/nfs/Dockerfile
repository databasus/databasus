FROM debian:bookworm-slim@sha256:99c0b25c46d0214d5a2091092e7908907dcaecd6a962cde0ac700acee222b6f3
RUN apt-get update \
    && apt-get install -y --no-install-recommends nfs-kernel-server rpcbind \
    && rm -rf /var/lib/apt/lists/*
COPY exports /etc/exports
COPY --chmod=0755 start.sh /usr/local/bin/start-nfs
EXPOSE 2049
# Kernel NFS configuration requires a privileged root process for the fixture's lifetime.
# nosemgrep: dockerfile.security.missing-user-entrypoint.missing-user-entrypoint
ENTRYPOINT ["/usr/local/bin/start-nfs"]
