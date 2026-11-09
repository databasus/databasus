FROM docker.io/library/debian:bookworm-slim@sha256:99c0b25c46d0214d5a2091092e7908907dcaecd6a962cde0ac700acee222b6f3

RUN apt-get update \
 && apt-get install -y --no-install-recommends pure-ftpd busybox \
 && rm -rf /var/lib/apt/lists/* \
 && useradd --uid 1001 --create-home --home-dir /home/ftpusers/testuser testuser

COPY start.sh /start.sh

EXPOSE 21

# pure-ftpd requires root to authenticate and chroot sessions, then drops to the FTP user.
# nosemgrep: dockerfile.security.missing-user-entrypoint.missing-user-entrypoint
ENTRYPOINT ["sh", "/start.sh"]
