FROM alpine:3.21

RUN apk add --no-cache openssh-server \
 && ssh-keygen -A \
 && adduser -D -s /bin/sh testuser \
 && echo 'testuser:testpassword' | chpasswd

COPY sshd_config /etc/ssh/sshd_config
COPY authorized_keys /home/testuser/.ssh/authorized_keys

RUN chown -R testuser:testuser /home/testuser/.ssh \
 && chmod 700 /home/testuser/.ssh \
 && chmod 600 /home/testuser/.ssh/authorized_keys

EXPOSE 22

# nosemgrep: dockerfile.security.missing-user.missing-user
CMD ["/usr/sbin/sshd", "-D", "-e"]
