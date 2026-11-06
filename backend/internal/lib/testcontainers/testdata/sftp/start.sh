#!/bin/sh
set -eu

if [ -n "$SFTP_PASSWORD" ]; then
    printf 'testuser:%s\n' "$SFTP_PASSWORD" | chpasswd
else
    usermod --password '*' testuser
fi

ssh-keygen -A
exec /usr/sbin/sshd -D -e
