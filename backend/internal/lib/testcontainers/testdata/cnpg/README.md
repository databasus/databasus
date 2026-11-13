# CloudNativePG fixture dependencies

The k3s/CloudNativePG (CNPG) fixture applies the operator release manifest in
this directory. It publishes no images. The k3s, k3s system, CNPG operator and
CNPG PostgreSQL images are selected through
`internal/lib/testimages/images.json` and imported into k3s from the host's
Docker; k3s never pulls from a registry.

| Component | Version | Official publication evidence checked on 2026-11-12 |
| --- | --- | --- |
| CNPG release manifest | 1.30.1 | [GitHub release](https://github.com/cloudnative-pg/cloudnative-pg/releases/tag/v1.30.1), 2026-09-23 13:35:42 UTC; [`cnpg-1.30.1.yaml`](https://github.com/cloudnative-pg/cloudnative-pg/releases/download/v1.30.1/cnpg-1.30.1.yaml), SHA-256 `37237f145d8138256ea25ae830f87759255665ff08f8d552fdd8224a5ec032fb` |
| CNPG operator image | 1.30.1 | Named by the release manifest above; index `sha256:923c267ec29636db3bee20f993d0ec4973fa22998e1adad37da79e4d32b5bc07` |
| CNPG PostgreSQL 17 image | 17.11-202610260819-standard-trixie | [CNPG image catalog](https://github.com/cloudnative-pg/artifacts/blob/e4b09dbc70/image-catalogs/catalog-standard-trixie.yaml), commit of 2026-10-26 08:35:11 UTC; index `sha256:ed6923f9ba2399cf9070379a353c710c29ced8a294be73986a2c3dae7e599ce3` |
| CNPG PostgreSQL 18 image | 18.6-202610260820-standard-trixie | The same catalog commit; index `sha256:39add80f5b1e573e2f6b641c66fd1a2f8dede67ca0330e870285bc1078551390` |
| k3s | v1.36.5+k3s1 | [GitHub release](https://github.com/k3s-io/k3s/releases/tag/v1.36.5%2Bk3s1), 2026-09-30 16:10:03 UTC; [Docker Hub tag](https://hub.docker.com/v2/repositories/rancher/k3s/tags/v1.36.5-k3s1), 2026-09-30 16:10:13 UTC |
| pause | 3.10.2 | [Docker Hub tag](https://hub.docker.com/v2/repositories/rancher/mirrored-pause/tags/3.10.2), 2026-07-09 18:04:44 UTC |
| CoreDNS | 1.14.7 | [Docker Hub tag](https://hub.docker.com/v2/repositories/rancher/mirrored-coredns-coredns/tags/1.14.7), 2026-08-20 00:00:56 UTC |
| local-path provisioner | v0.0.37 | [Docker Hub tag](https://hub.docker.com/v2/repositories/rancher/local-path-provisioner/tags/v0.0.37), 2026-08-05 03:06:19 UTC |
| busybox, the local-path provisioner's volume helper | 1.37.0 | [Docker Hub tag](https://hub.docker.com/v2/repositories/rancher/mirrored-library-busybox/tags/1.37.0), 2025-03-21 18:58:15 UTC |

The release's checksum file does not list the manifest, so its SHA-256 above was
computed from the downloaded asset. The k3s system image versions are the ones
the k3s release lists in its
[`k3s-images.txt`](https://github.com/k3s-io/k3s/releases/download/v1.36.5%2Bk3s1/k3s-images.txt);
the fixture disables Traefik, the service load balancer, metrics-server and the
Helm controller, so their images are not recorded.

CNPG 1.30 supports Kubernetes 1.34 to 1.36 according to its
[supported releases](https://github.com/cloudnative-pg/cloudnative-pg/blob/v1.30.1/docs/src/supported_releases.md).
The CNPG PostgreSQL images are built on Debian trixie, like the official
`postgres:17` (17.11) and `postgres:18` (18.6) images that boot recovered
clusters, so both use the same C library collations.

Every selected release was published at least fourteen complete days before the
check date. The fixture changes one line of the manifest when it applies it: the
operator's `imagePullPolicy: Always` becomes `IfNotPresent`, because k3s has no
registry to pull from and uses the imported image.
