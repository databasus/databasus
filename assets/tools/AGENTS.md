# Client binaries — rules for changing this tree

Every file under `assets/tools/` is a vendored third-party binary that runs in
production. `docs/architecture.md` explains why they are committed instead of installed at
image build time. Two rules follow from that and apply to every change here:
the bytes in the repository are the bytes that ship, so a binary must be
verified in the runtime environment rather than on a developer machine, and a
directory name is a label chosen by us, which the startup check holds you to
but which nothing enforces while you are assembling the bundle.

The backend resolves a binary as
`assets/tools/<arch>/<engine>/<engine>-<bundle>/bin/<command>`, where `<arch>`
is `x64` for `linux/amd64` and `arm` for `linux/arm64`. The mapping lives in
`backend/internal/util/tools/paths.go`.

## Before adding a version

Adding an engine version is mostly about the binary, not about the code that
reads the version. Work through this list before committing anything; the
first two items often end the job early.

- [ ] **A new upstream release line needs a client here before it is
      supported.** Until the bundle lands, a server on that line is refused at
      the connection test, by design: we never dump a server with a client from
      an older line. Both engines open a line roughly once a quarter, so this
      is the recurring task that keeps that promise.
- [ ] **Is this a new line, or a new release inside a line we already serve?**
      Only a new line needs a bundle. A newer release inside a served line —
      MariaDB 13.2 against our 13.0 client, MySQL 26.10 against our 26.7 one —
      keeps working, and the bundle is refreshed when there is a reason to.
      MariaDB's modern client additionally serves every line from 10.2 up by
      designation, which is recorded in `README.md` in this directory.
- [ ] **Is the release line one upstream still patches?** Short-term and
      innovation releases live about a quarter. Prefer the long-term release of
      that era; bundle a short-term release only when users run it. When a line
      we bundle later gets a long-term release, refresh the bundle to it rather
      than adding an identity.
- [ ] **Do binaries exist for both architectures?** Check, in order: the
      vendor's architecture-independent tarball, the vendor's Debian
      repository, the vendor's Ubuntu repository, then RPMs for an older
      enterprise distribution. Do not conclude a build is unavailable after
      checking one of them.
- [ ] **Will the binary start in the runtime image?** The image is
      `debian:bookworm-slim` with glibc 2.36. Compare with
      `objdump -T <binary> | grep -oE 'GLIBC_2\.[0-9]+' | sort -uV | tail -1`
      and list the shared libraries with `objdump -p <binary> | grep NEEDED`.
      Every library must already be in the image; see the table in
      `README.md` in this directory. A matching soname is not enough: check
      the OpenSSL symbol versions the same way with `OPENSSL_3\.[0-9]+`,
      because bookworm ships OpenSSL 3.0 and a newer requirement fails only at
      run time. A development host with a newer OpenSSL hides the problem, so
      run the binary in the runtime image as the refresh checklist shows.
- [ ] **Know which of the two identity shapes the engine uses.** MySQL's
      identity doubles as the name of its bundle directory, because each MySQL
      line gets a client of its own. MariaDB keeps two values, a server
      identity and a client version, because one client serves many server
      lines. Follow the engine's existing shape rather than introducing the
      other one, and never record a version the server did not report.
- [ ] **Add the bundle and the identity together.** The new directory goes into
      the bundle set the startup check walks, in
      `backend/internal/util/tools/mysql.go` or `mariadb.go`, and the identity
      it serves is mapped to it. Detection refuses any version with no bundle,
      so these two land in one change or the version stays unsupported.
- [ ] **Check both architectures before claiming the version is supported.**
      A version whose client exists for only one architecture is supported only
      there, and is refused on the other. `assets/tools/arm/mysql/` has no 5.7
      bundle for that reason.
- [ ] **Do not look for version tables to extend; there are none left.** The
      restore downgrade order is computed from the numbers in the identity, the
      MySQL compression choice is derived from the identity rather than matched
      against a list of known ones, and the interface renders the identity
      string itself. All three used to be lookup tables whose missing entry
      changed behavior quietly. If you find yourself adding a version to a
      table, the table is the bug.
- [ ] **An identity names a line, so its digits are not a promise about the
      binary.** Identity `26` covers every MySQL 26.x and `13.0` every MariaDB
      13.x, while the bundle holds one patch. That is intended. What is not
      intended is a bundle holding a version from a *different* line, which the
      startup check now rejects.
- [ ] **Add the identity constant on both sides.** The backend constant and the
      one in `frontend/src/entity/databases/model/<engine>/` name the same
      value, and the enumeration is the type of the field, so a missing
      constant is a build error rather than a blank field.
- [ ] **Add the version to the test matrices.** A unit test of the version
      mapping proves nothing about a dump. The matrices in
      `backend/internal/features/databases/databases/<engine>/model_test.go`
      and `backend/internal/features/tests/logical/<engine>/backup_restore_test.go`
      boot a real server of each version.
- [ ] **Update the supported-version lists** in the root `README.md` and in all
      five copies under `assets/readme/`, and the tables in `README.md` in this
      directory. `assets/readme/AGENTS.md` requires every copy to follow the
      English file.

## Refreshing a patch level

Refreshing is lower risk than adding a version, but it changes the client for
every server already using that bundle, so it is not a silent operation.
For MariaDB, `tools/refresh-mariadb-bundle.sh <bundle> <arch>` does the work;
edit the pinned package version in it rather than passing one in.

- [ ] Keep every binary within one bundle on the same upstream build. A
      `pg_dump` older than its sibling `pg_basebackup` is how issue #725
      survived unnoticed.
- [ ] Pin the exact upstream package version in the refresh script. Installing
      "the newest patch of this line" makes the result depend on the day it
      ran, and two architectures refreshed a week apart then disagree.
- [ ] Refresh both architectures from the same upstream source in the same
      change. Mixing sources is how one directory name comes to mean two
      different builds, and the difference surfaces as an engine-specific
      dump failure on one architecture only.
- [ ] Run the refreshed binary in the runtime base image before committing:

      docker run --rm --platform linux/amd64 -v "$PWD/assets/tools:/tools:ro" \
        debian:bookworm-slim bash -c 'apt-get update -qq && \
          apt-get install -y -qq libedit2 libncurses6 libtinfo6 libssl3 >/dev/null && \
          /tools/x64/<engine>/<bundle>/bin/<command> --version'

- [ ] Confirm the executable bit is recorded in the index. `git ls-files -s`
      must show `100755`; a binary committed as `100644` fails at runtime and
      not at build time.
- [ ] Run the engine's whole matrix, not only the refreshed version. A refresh
      changes the client for every server on that bundle.
- [ ] Record the new patch and its download source in the tables in
      `README.md` in this directory, in the same change.

## What not to do here

- Do not install client binaries in the `Dockerfile`; see "System" in `docs/architecture.md`.
- Do not add a shared library to the runtime image only to satisfy one client
  when a build against the libraries already present exists.
- Do not trust a directory name while you are building the bundle. Confirm
  what a binary is by running `--version`, or by reading its version string
  when the architecture cannot be executed locally. The startup check does
  compare the two — it runs every client's version flag and rejects a bundle
  whose version belongs to another line — but it is the last net, not the
  first. Note that `mariadb-dump --version` prints two numbers and only the one
  before `-MariaDB` is the server line; `client 10.19` is the protocol
  version.
