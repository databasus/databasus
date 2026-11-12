# Patroni fixture dependencies

This fixture builds PostgreSQL/Patroni and HAProxy locally. It publishes no images.
PostgreSQL bases and the etcd index/platform digests are selected through
`internal/lib/testimages/images.json`.

| Component | Version | Official publication evidence checked on 2026-11-11 |
| --- | --- | --- |
| Patroni | 4.1.5 | [PyPI metadata](https://pypi.org/pypi/patroni/4.1.5/json), 2026-08-12 |
| etcd | 3.6.16 | [Quay tag metadata](https://quay.io/api/v1/repository/coreos/etcd/tag/?specificTag=v3.6.16), 2026-10-22 22:58:43 UTC; index `sha256:a9c7cc12156f4b26ca41f0b57ba5b76c45037ff6c1e47ea3d68cb7908d48a965` |
| HAProxy source | 3.2.26 | [Upstream directory](https://www.haproxy.org/download/3.2/src/), 2026-10-14; [SHA256](https://www.haproxy.org/download/3.2/src/haproxy-3.2.26.tar.gz.sha256) verified before extraction |

The current Docker Hub HAProxy 3.2.26 image was published on 2026-11-05,
according to its [tag metadata](https://hub.docker.com/v2/repositories/library/haproxy/tags/3.2.26).
It had not completed the fourteen-day waiting period on the verification date,
so the fixture compiles the eligible upstream source instead.

`requirements.txt` records the complete Patroni dependency selection and hashes.
`build-requirements.txt` installs the locked Python build tools before building
the python-etcd distribution, without resolving another build environment.
`publications.json` records official PyPI publication evidence for every locked
package. Every locked hash was compared with the corresponding official
distribution metadata and its publication time. The cutoff is
2026-11-04 00:00:00 UTC, allowing seven complete days before the check date.

Refresh the runtime lock deliberately, review the official dates and hashes,
and regenerate the publication evidence before adopting it:

```sh
uv pip compile --generate-hashes --exclude-newer 2026-11-04T00:00:00Z \
  --python-version 3.13 --no-header --no-annotate \
  --output-file requirements.txt requirements.in
```

The fixture uses the distribution package manager's authenticated repositories
for build prerequisites. Those distribution packages are not retained or locked;
rebuilding a fixture does not promise identical image bytes. Runtime test results
must record the versions and host architecture actually exercised.
