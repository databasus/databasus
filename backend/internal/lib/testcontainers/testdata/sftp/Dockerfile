FROM debian:bookworm-slim@sha256:99c0b25c46d0214d5a2091092e7908907dcaecd6a962cde0ac700acee222b6f3

RUN apt-get update \
 && apt-get install -y --no-install-recommends openssh-server \
 && rm -rf /var/lib/apt/lists/* \
 && useradd --uid 1001 --create-home --shell /bin/false testuser \
 && mkdir -p /run/sshd /home/testuser/.ssh /home/testuser/upload \
 && chown root:root /home/testuser \
 && chmod 0755 /home/testuser \
 && chown testuser:testuser /home/testuser/upload

COPY sshd_config /etc/ssh/sshd_config
COPY start.sh /start.sh

EXPOSE 22

# sshd requires root to authenticate users and enter the SFTP chroot before dropping privileges.
# nosemgrep: dockerfile.security.missing-user-entrypoint.missing-user-entrypoint
ENTRYPOINT ["sh", "/start.sh"]
