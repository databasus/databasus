---
name: client-tool-bundles
description: Add a supported version of a database engine or refresh a vendored client tool bundle under assets/tools (pg_dump, mysqldump, mariadb-dump, mongodump and their siblings). Covers deciding whether a new bundle is needed, finding binaries for both architectures, checking that they start in the runtime image, registering the bundle with its version identity, and refreshing a patch level. Load when a task adds an engine version, supports a new release line, or replaces binaries under assets/tools.
---

# Client tool bundles

Every file under `assets/tools/` is a vendored third-party binary that runs in production. "System" in [`docs/architecture.md`](../../../docs/architecture.md#system) explains why they are committed instead of installed at image build time. Two rules follow from that:

- The bytes in the repository are the bytes that ship, so verify a binary in the runtime image, not on a development host.
- A bundle directory name is a label we choose. Nothing holds you to it while you assemble the bundle, so confirm what each binary is yourself.

[`assets/tools/README.md`](../../../assets/tools/README.md) lists the bundles, their patch levels and download sources, the runtime image and the shared libraries it provides. Read it before changing anything here.

## Before adding a version

Adding an engine version is mostly about the binary, not about the code that reads the version. Work through this list before committing anything; the first two items often end the job early.

- [ ] **A new upstream release line needs a client here before it is supported.** A server on a line without a bundle is refused at the connection test, because a server is never dumped with a client from an older line.
- [ ] **Decide whether this is a new line or a new release inside a line we already serve.** Only a new line needs a bundle. A newer release inside a served line keeps working, and its bundle is refreshed when there is a reason to.
- [ ] **Prefer a line upstream still patches.** Short-term and innovation releases live about a quarter. Bundle the long-term release of that era, and a short-term release only when users run it. When a line we bundle later gets a long-term release, refresh the bundle to it instead of adding an identity.
- [ ] **Find binaries for both architectures.** Check, in order: the vendor's architecture-independent tarball, the vendor's Debian repository, the vendor's Ubuntu repository, then RPMs for an older enterprise distribution. Do not conclude a build is unavailable after checking one of them.
- [ ] **Check that the binary starts in the runtime image.** Compare its newest glibc symbol with the image's glibc using `objdump -T <binary> | grep -oE 'GLIBC_2\.[0-9]+' | sort -uV | tail -1`, and list its shared libraries with `objdump -p <binary> | grep NEEDED`. Every library must already be in the image, per the table in the README. A matching soname is not enough: check the OpenSSL symbol versions the same way with `OPENSSL_3\.[0-9]+`, because a newer requirement fails only at run time. A development host with a newer OpenSSL hides the problem, so run the binary in the runtime image as the refresh list below shows.
- [ ] **Follow the identity shape the engine already uses.** MySQL and MariaDB record a server's version differently; copy the existing shape instead of introducing the other one, and never record a version the server did not report.
- [ ] **Add the bundle and the identity it serves in one change**, in the engine's file under `backend/internal/util/tools/`. A version without a bundle stays unsupported.
- [ ] **Check both architectures before claiming the version is supported.** A version whose client exists for one architecture only is supported only there and refused on the other; record that in the README.
- [ ] **Never add a version to a lookup table.** If adding a version seems to need a table entry, the table is the bug: derive the behavior from the numbers in the identity.
- [ ] **An identity names a line, so its digits do not name the patch in the bundle.** A bundle holding a release from a different line is a mistake.
- [ ] **Add the identity constant on both sides**: in the backend and in `frontend/src/entity/databases/model/<engine>/`.
- [ ] **Add the version to the test matrices** that boot a real server of each version, such as `backend/internal/features/databases/databases/<engine>/model_test.go` and `backend/internal/features/tests/logical/<engine>/backup_restore_test.go`. A unit test of the version mapping proves nothing about a dump.
- [ ] **Update the supported-version lists** in the root `README.md`, in all five copies under `assets/readme/`, and the tables in `assets/tools/README.md`.

## Refreshing a patch level

A refresh changes the client for every server already using that bundle, so it is never a silent operation. For MariaDB, `tools/refresh-mariadb-bundle.sh <bundle> <arch>` does the work; edit the pinned package version in it instead of passing one in.

- [ ] Keep every binary within one bundle on the same upstream build. A tool older than its sibling can emit wrong data unnoticed.
- [ ] Pin the exact upstream package version in the refresh script. "The newest patch of this line" makes the result depend on the day it ran, and two architectures refreshed a week apart then disagree.
- [ ] Refresh both architectures from the same upstream source in the same change. Otherwise one directory name comes to mean two different builds.
- [ ] Before committing, run the refreshed binary's `--version` for each architecture in a container of the runtime base image the `Dockerfile` names, with `assets/tools` mounted read-only and the shared libraries from the README table installed.

- [ ] Confirm the executable bit is recorded in the index: `git ls-files -s` must show `100755`. A binary committed as `100644` fails at run time, not at build time.
- [ ] Run the engine's whole test matrix, not only the refreshed version.
- [ ] Record the new patch and its download source in the tables in `assets/tools/README.md` in the same change.

## What not to do

- Do not install client binaries in the `Dockerfile`.
- Do not add a shared library to the runtime image only to satisfy one client when a build against the libraries already present exists.
- Do not trust a directory name while building the bundle. Confirm what a binary is by running `--version`, or by reading its version string when the architecture cannot run locally. `mariadb-dump --version` prints two numbers, and only the one before `-MariaDB` is the server line.
