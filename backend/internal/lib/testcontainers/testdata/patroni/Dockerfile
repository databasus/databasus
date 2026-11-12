ARG POSTGRES_IMAGE
FROM ${POSTGRES_IMAGE}

RUN apt-get update \
 && apt-get install -y --no-install-recommends python3-venv gcc make libc6-dev curl zstd \
 && rm -rf /var/lib/apt/lists/* \
 && python3 -m venv /opt/patroni

COPY build-requirements.txt /tmp/build-requirements.txt
RUN /opt/patroni/bin/pip install --require-hashes --no-deps -r /tmp/build-requirements.txt
COPY requirements.txt /tmp/requirements.txt
RUN /opt/patroni/bin/pip install --require-hashes --no-deps --no-build-isolation -r /tmp/requirements.txt

RUN curl --fail --location --output /tmp/haproxy.tar.gz https://www.haproxy.org/download/3.2/src/haproxy-3.2.26.tar.gz \
 && echo '760ef35fafc1e05844e73ad07e138e9e96fddbf5de02f606249c027f754827da  /tmp/haproxy.tar.gz' | sha256sum --check \
 && mkdir /tmp/haproxy \
 && tar -xzf /tmp/haproxy.tar.gz -C /tmp/haproxy --strip-components=1 \
 && make -C /tmp/haproxy -j2 TARGET=linux-glibc USE_THREAD=1 \
 && install /tmp/haproxy/haproxy /usr/local/sbin/haproxy \
 && rm -rf /tmp/haproxy /tmp/haproxy.tar.gz

COPY start.sh /usr/local/bin/start-patroni
ENV PATH="/opt/patroni/bin:${PATH}"

# The inherited volume needs ownership repair before Patroni can initialize PostgreSQL;
# start.sh then drops to the postgres user with gosu.
# nosemgrep: dockerfile.security.missing-user-entrypoint.missing-user-entrypoint
ENTRYPOINT ["sh", "/usr/local/bin/start-patroni"]
