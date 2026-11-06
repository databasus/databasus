#!/bin/sh
set -eu

printf '%s\n%s\n' "$FTP_USER_PASS" "$FTP_USER_PASS" |
    pure-pw useradd testuser -u testuser -d /home/ftpusers/testuser
pure-pw mkdb

busybox syslogd -n -O /dev/stdout &
exec /usr/sbin/pure-ftpd -A -j -l puredb:/etc/pure-ftpd/pureftpd.pdb \
    -E -H -R -p "$FTP_PASSIVE_PORTS" -P 127.0.0.1
