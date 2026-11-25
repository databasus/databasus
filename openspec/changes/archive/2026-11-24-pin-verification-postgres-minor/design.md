## Context

See proposal.md for the motivation. The current state:

- The agent builds the image reference as `<repo>:<major>` (`backend/internal/agent/container/manager.go:303-305`). TimescaleDB jobs use `timescale/timescaledb:<extension version>-pg<major>` (`manager.go:296-298`).
- When the configured image cannot be obtained, the agent falls back to the official `postgres:<major>` image (`manager.go:129-151`, `manager.go:310-321`).
- `DockerEngine.EnsureImage` pulls only when no image with that reference exists locally (`backend/internal/agent/container/dockerengine.go:93-109`). A moving tag is therefore pulled once and never refreshed.
- The verification image pins one `postgres:<major>` base digest per major (`backend/cmd/verification-agent/images/postgres/Dockerfile:11-17`). It then installs extension packages from the PostgreSQL apt repository (`Dockerfile:79-92`). Those packages can upgrade the server: the published `databasus/verification-postgres:18` of 2026-11-22 reports `PostgreSQL 18.7 (Debian 18.7-1.pgdg13+5)`, while its base's `PG_VERSION` is `18.6-1.pgdg13+2`.
- The release publishes `:<major>` and `:<major>-v<Databasus version>` (`.github/workflows/ci-release.yml:1490-1493`), and only on a release.
- `scripts/image_pins.py:13-18` requires each `FROM` digest of the verification Dockerfile to equal the `postgres:<major>` record of `backend/internal/lib/testimages/images.json`.
- Agent releases follow the server: a running agent upgrades itself to the server's version (`backend/internal/agent/start/start.go:99-110`). The agent's minors therefore move with Databasus releases.
- The agent removes no image anywhere in its code. At startup it checks Docker (`start.go:72`) and then removes the containers and networks it owns (`start.go:80`, `manager.go:120-122`).

## Goals / Non-Goals

**Goals:**

- One file in the repository is the authoritative list of the minor for each major. The agent, the image build and the release tags read it, and the Dockerfile base digests and the test catalog are checked against it.
- An image tag never claims a minor it does not contain.
- Images of superseded minors in the verification repository do not pile up on the agent's host.

**Non-Goals:**

- Detecting at runtime which minors a registry offers.
- Changing how the server tells the agent the backup's major version.
- Removing fallback `postgres:<version>` images and TimescaleDB images. The agent cannot tell them from images the host's owner pulled for other work; that needs a record of what the agent pulled and is a separate change.

## Decisions

### The agent embeds the pinned PostgreSQL versions, one per major

`backend/internal/agent/container/postgres_versions.json` maps each major to its pinned server version, for example `"18": "18.7"`. The agent embeds the file at build time. It decodes and validates it when the container manager is constructed at agent startup, never at import time; `NewManager` (`manager.go:76`) gains an error return, and its caller in `start.go` stops the agent on that error, and resolves `<repo>:<version>` through `GetPinnedPostgresVersion(pgMajor)`. The release workflow reads the same file with `jq` to pass the version to the image build. A unit test fails if the file lacks a major from 12 to 18 or holds a minor of another major, so a broken entry never reaches a release.

Rejected alternatives:

- **Ask the registry for the newest minor at job time.** Every verification would then depend on registry availability and on the day it ran, and two agents on one release could again disagree.
- **Derive the minor from the Dockerfile.** The Go binary cannot read the Dockerfile, and parsing `FROM` lines in CI duplicates the source.
- **Let the server send the minor.** The server would have to know image contents, and an old agent would request a tag its release never tested.

The repository rule "never add a version to a lookup table" in `.agents/skills/client-tool-bundles/SKILL.md` is about recognizing a server's version from its identity. This list pins a dependency, like a digest. It is the authoritative source for the minor, but not the only file a person edits to raise one: the Dockerfile base digest and the catalog records change with it, and the build check and `scripts/image_pins.py` reject them when they disagree with the list.

### Image tags name the PostgreSQL minor

The agent requests `<repo>:<major>.<minor>` instead of today's `<repo>:<major>`. A rejected alternative is the existing release tag `<repo>:<major>-v<Databasus version>`: it would also defeat the cache, but no other repository publishes it. The official `postgres` repository publishes `18.7`, so the fallback and the `verificationPgImageRepo: postgres` setting resolve to the same exact minor with no special case. A tag named after the Databasus version also says nothing about the server inside it.

The release keeps publishing `:<major>` and `:<major>-v<version>` beside the new tag. Nothing in the agent reads them; they serve people who start the image by hand.

### The build fails when the server is not the pinned minor

The image build takes the pinned version as the `PG_SERVER_VERSION` build argument, for example `18.7`; the name says it is a whole version, unlike `PG_MAJOR`. After installing the extensions, it checks two things: the base image's `PG_VERSION` starts with that minor, and `postgres --version` reports it. If either differs, the build fails. The first check keeps the pinned base honest; without it, a stale base upgraded by apt would pass. The second catches apt upgrading the server.

Rejected alternative: **pin the server package version in `apt-get install`.** The PostgreSQL apt repository keeps only current builds in its main pool, so the pin breaks as soon as a newer build appears, and the extension packages may require it anyway. A failing check states the cause; a failing install does not.

A rebuilt image for an unchanged minor replaces the old one under the same tag. Agents that already hold the tag keep their copy; that is acceptable because the server minor is the same. Extension updates reach an agent with the next minor.

### Tests pin the verification images through the catalog

The agent end-to-end tests currently let the agent pull the moving `:<major>` tag from Docker Hub. Under the new scheme, the `:<minor>` tags do not exist before the first release that builds them, so the agent would silently fall back to the official image and the tests would stop covering the Databasus image. The catalog records are therefore renamed to `databasus/verification-postgres:<minor>` and pinned to digests whose server is that minor. The test setup tags the pinned digest locally under that name before the agent starts. The agent then finds the image locally and the tests run on pinned content. The tags go under a verification repository unique to the test run, configured for the test agent. Several worktrees share one Docker daemon, and the startup cleanup of an agent on the default repository would remove tags that a run on another branch, with other pinned versions, still needs.

`scripts/image_pins.py` additionally checks that every major in the agent's list has a catalog record under its minor name.

### The minors of 2026-11-12 are taken before the waiting period ends

PostgreSQL 14.25, 15.20, 16.16, 17.12 and 18.7 were released on 2026-11-12, and their fourteen-day waiting period ends on 2026-11-27. They fix CVE-2026-6460, and the agent's current behavior breaks verification for users. Pinning an older minor is not possible, because the extension packages pull the newer server into the image regardless. The existing guide already allows an exception with a stated reason. The user asked to make the conditions explicit, so `docs/how-to-upgrade-deps.md` gains a short rule naming when a shorter period is allowed and what the report records. This change records the exception under that rule.

### The agent removes superseded images of its verification repository at startup

Pinning by minor turns every minor raise into a new image of 1.1-1.4 GB on the agent's host, measured on the published images. With a minor release about every quarter and a few majors in use, that is several gigabytes every half year. The agent therefore removes them itself.

At startup, right after removing its leftover containers (`start.go:80`), the agent lists the local images of its verification repository and removes each tag that is not a pinned version: the old `:<major>` tags and earlier minors. The repository is the default `databasus/verification-postgres` or the configured `verificationPgImageRepo`. When the configured repository is the official `postgres`, the agent removes nothing. It removes by tag, without force, so an image that also carries another tag only loses this one, and an image a container uses stays; the agent logs a warning and continues. A failed cleanup never stops the agent or a job.

Startup is enough: the pinned versions change only with an agent upgrade, and the upgrade restarts the agent.

Rejected alternatives:

- **Leave cleanup to the host's owner.** The agent created the growth, nothing tells the owner about it, and a full disk fails verification and every other service on the host.
- **Remove with force.** It would delete an image a container still uses, including a container of another agent or of the owner.
- **Remove after each pull instead of at startup.** It adds work to every job for an event that happens only on upgrade.
- **Also remove `postgres:<version>` and TimescaleDB images.** The official repository is shared with whatever else runs on the host, and Docker cannot mark an image as pulled by the agent. Telling them apart needs the agent to record its own pulls; that is out of scope.

### Terms

The change uses three terms that `docs/glossary.md` gains: the **verification image**, the Databasus image a verification restores into, published in the **verification image repository** (`databasus/verification-postgres` by default, or the repository set by `verificationPgImageRepo`); and the **pinned PostgreSQL version**, the exact server version an agent release names for a major. A verification image tag is **superseded** when it is not a pinned PostgreSQL version of the running agent. Code names follow the same words: `postgres_versions.json` holds the pinned PostgreSQL versions and `GetPinnedPostgresVersion` reads one.

## Risks / Trade-offs

- [A custom `verificationPgImageRepo` publishes only `:<major>` tags] → The pull fails and the agent falls back to the official image of the same minor. The startup log already prints the configured repository; the proposal marks the break.
- [Fallback `postgres:<version>` images and TimescaleDB images still build up on the host, and a superseded verification image stays while a container uses it] → Accepted for this change; cleaning the first two needs a record of the agent's own pulls and is a separate change, and the third is removed on a later start once the container is gone.
- [The PostgreSQL apt repository publishes a newer server between two releases] → The next release build fails with a clear message until someone raises the minor in the list. This is the intended signal, but it blocks the release.
- [Two agents on one host run different releases during an upgrade] → The newer agent may remove the older agent's image when no container uses it; the older agent pulls it again on its next job, and both converge once the older agent upgrades.
- [The container engine does not support the image list filter by repository] → The agent logs a warning and skips cleanup; verification is unaffected. Podman serves the same Docker Engine API, but this change runs the cleanup end-to-end test only on Docker; the Podman subset is not extended.
- [The official image of a minor disappears from Docker Hub] → Official tags are not removed in practice; the fallback would fail the job as a setup failure, which is retried.

## Migration Plan

The release that ships this change publishes the `:<minor>` images in the same workflow, before the `release` job. Agents upgrade to that release and pull the new tags on their next verification. Rolling back to an earlier release restores the `:<major>` behavior, with the cache problem it had.
