## Why

The verification agent restores a backup into a container from the image tag `<repo>:<major>`, such as `databasus/verification-postgres:18`. That tag moves between PostgreSQL minor releases, and the agent pulls it only when no image with that tag exists locally. Agents on the same Databasus release therefore verify on different minors, and an agent that pulled the tag once never gets a newer server. This is how PostgreSQL 18.7 broke verification for existing agents: a backup from an 18.7 source carries the new internal column option `pg_attver`, and an agent still holding an 18.6 image rejects it. A freshly published `:18` image already contains 18.7, so the failure depends only on when the agent first pulled.

The tag does not even describe its content. The published `databasus/verification-postgres:18` image of 2026-11-22 is built on an 18.6 base, but installing the extension packages upgraded its server to 18.7. The minor inside an image is decided by the package mirror on the day of the build.

## What Changes

- Each agent release names an exact PostgreSQL minor for every supported major and restores into the image tagged with that minor, for example `databasus/verification-postgres:18.7`. A release that raises a minor makes every upgraded agent pull the new image on its next verification.
- **BREAKING**: the agent stops requesting `<repo>:<major>`. A custom `verificationPgImageRepo` must publish `<major>.<minor>` tags. The official `postgres` repository already does, so the stock-image setting and the fallback to the official image keep working with the same exact minor. The `:<major>` tags are still published for people who run the image by hand; nothing in the agent reads them.
- The verification image build pins its base images to the minors the agent names and fails when the server installed in the built image is a different minor.
- The release publishes the verification image under `<major>.<minor>` in addition to its existing tags.
- At startup the agent removes images of its verification repository whose tag is not a pinned version, so superseded minors do not fill the host's disk. It removes nothing from the official `postgres` repository and never forces a removal.
- The pinned minors are the releases of 2026-11-12, which carry the `pg_attver` change: 14.25, 15.20, 16.16, 17.12 and 18.7. Majors 12 and 13 stay on their final releases, 12.22 and 13.23.
- These minors are taken before their fourteen-day waiting period ends, as a recorded exception: the releases fix a security issue (CVE-2026-6460) and the current state breaks backup verification for users. At the user's request, `docs/how-to-upgrade-deps.md` states explicitly when the waiting period may be shortened for critical fixes; today it only says that an exception needs a reason.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `backup-verification`: the agent restores into the exact PostgreSQL minor its release names, and an agent upgrade moves it to a newer minor without manual image removal. The agent removes superseded images of its verification repository and nothing else.

## Impact

- Verification agent: image selection, the stock fallback and startup image cleanup in `backend/internal/agent/container` and `backend/internal/agent/start`, with their tests and the agent end-to-end image list.
- Verification image: `backend/cmd/verification-agent/images/postgres/Dockerfile` base pins and a build-time version check.
- Release workflow: the verification image jobs in `.github/workflows/ci-release.yml` add the minor tag.
- Test image catalog `backend/internal/lib/testimages/images.json` and the pin checker `scripts/image_pins.py`.
- Documentation: `docs/how-to-upgrade-deps.md`.
- The change answers to the root `AGENTS.md` and `backend/AGENTS.md`.

Out of scope:

- Updating the bundled PostgreSQL client tools, the other database clients and every other dependency. That is the next, separate commit.
- TimescaleDB images, which already carry an exact extension version in their tag.
- Removing fallback `postgres:<version>` images and TimescaleDB images from the agent's host. That needs the agent to record which images it pulled itself, and is a separate change.
- Letting the server choose the minor, or verifying a backup on the source's own minor.
- Authentication for private image registries.
