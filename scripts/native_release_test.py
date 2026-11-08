import json
import os
import re
import subprocess
import tempfile
import unittest
from pathlib import Path

from release_gate_test import get_transitive_needs
from workflow_jobs import get_jobs

ROOT = Path(__file__).resolve().parents[1]
ARCHITECTURES = {"amd64", "arm64"}
BACKEND_GROUPS = {
    "storage-providers",
    *(f"other-{shard}-of-3" for shard in range(1, 4)),
    "logical",
    "agent-e2e",
    *(f"physical-{shard}-of-3" for shard in range(1, 4)),
    *(f"physical-api-{shard}-of-3" for shard in range(1, 4)),
}
REQUIRED_TEST_JOBS = {
    "test-scripts",
    "test-backend",
    "test-frontend",
    "test-frontend-browser-ui",
    "test-e2e-browser",
    "test-e2e-browser-lifecycle",
    "test-verification-agent",
    "e2e-verification-agent-podman",
    "test-filesystems",
    "test-upgrade",
}
CANDIDATE_JOBS = {
    "test-e2e-browser",
    "test-e2e-browser-lifecycle",
    "test-filesystems",
    "test-upgrade",
}


def get_matrix_entries(job: str) -> list[dict[str, str]]:
    entries: list[dict[str, str]] = []
    for line in job.split("    steps:")[0].splitlines():
        field = re.fullmatch(r"          (- |  )(\w+): (\S+)", line)
        if field:
            if field[1] == "- ":
                entries.append({})
            entries[-1][field[2]] = field[3]
    return entries


def check_release_coverage(workflow: str) -> None:
    jobs = get_jobs(workflow)
    gates = get_transitive_needs(jobs, "determine-version")
    if (
        'go test -race -p=8 -parallel="$TEST_PARALLELISM"' not in jobs["test-backend"]
        or "TEST_PARALLELISM: '2'" not in jobs["test-backend"]
    ):
        raise AssertionError("test-backend: container concurrency is unbounded")
    for name in REQUIRED_TEST_JOBS | {"build-storage-image", "build-verification-image"}:
        job = jobs[name]
        entries = get_matrix_entries(job)
        groups = {
            "test-backend": BACKEND_GROUPS,
            "test-filesystems": {"permissions", "upgrades", "lifecycle", "network"},
            "test-frontend-browser-ui": {"1", "2", "3"},
            "test-e2e-browser-lifecycle": {"failures", "interruption", "isolation"},
        }.get(name, {""})
        identities = [
            (entry["arch"], entry.get("shard", entry.get("group", ""))) for entry in entries
        ]
        if len(set(identities)) != len(identities) or set(identities) != {
            (architecture, group) for architecture in ARCHITECTURES for group in groups
        }:
            raise AssertionError(f"{name}: incomplete native matrix")
        if name != "build-storage-image" and name not in gates:
            raise AssertionError(f"{name}: disconnected release gate")
        if "fail-fast: false" not in job:
            raise AssertionError(f"{name}: failures cancel other legs")
        if "run: scripts/check_native_architecture.sh ${{ matrix.arch }}" not in job:
            raise AssertionError(f"{name}: native architecture is unchecked")
        for entry in entries:
            architecture, runner = entry["arch"], entry["runner"]
            is_vm = name == "e2e-verification-agent-podman" or entry.get("group") == "network"
            if is_vm:
                required_runner = "ubuntu-24.04" + ("-arm" if architecture == "arm64" else "")
            elif name == "test-backend":
                size = (
                    "8vcpu-16ram"
                    if (
                        entry["group"].startswith("other-")
                        or entry["group"] in {"storage-providers", "logical"}
                    )
                    else "4vcpu-8ram"
                )
                required_runner = f"namespace-profile-{size}-ubuntu-22-" + (
                    "arm" if architecture == "arm64" else "x64"
                )
            else:
                required_runner = runner
                suffix = "-arm" if architecture == "arm64" else "-x64"
                if not runner.endswith(suffix):
                    raise AssertionError(f"{name}: runner architecture mismatch")
            if runner != required_runner:
                raise AssertionError(f"{name}: unsuitable native runner")
        for step in re.split(r"(?m)^      - name: ", job):
            if "continue-on-error:" in step and step.splitlines()[0] not in {
                "Download artifact",
                "Download artifact (attempt 2)",
                "Upload artifact",
                "Upload artifact (attempt 2)",
            }:
                raise AssertionError(f"{name}: test failure is allowed")
        if re.search(r"(?m)^    if:", job):
            raise AssertionError(f"{name}: required suite can be skipped")
    if "needs: [build-storage-image]" not in jobs["test-e2e-browser-lifecycle"]:
        raise AssertionError("lifecycle: disconnected candidate prerequisite")
    for name in CANDIDATE_JOBS | {"build-storage-image"}:
        job = jobs[name]
        if (
            "      ARTIFACT_NAME: databasus-storage-test-${{ matrix.arch }}\n" not in job
            or job.count("          name: ${{ env.ARTIFACT_NAME }}\n") != 3
        ):
            raise AssertionError(f"{name}: artifact architecture mismatch")
        if name in CANDIDATE_JOBS and re.findall(r"DATABASUS_IMAGE: ([^\n]+)", job) != [
            "databasus-storage-test:${{ github.sha }}-${{ matrix.arch }}"
        ]:
            raise AssertionError(f"{name}: scenario candidate architecture mismatch")
        if "databasus-storage-test:${{ github.sha }}-${{ matrix.arch }}" not in job:
            raise AssertionError(f"{name}: candidate architecture mismatch")
        if "databasus-storage-test-${{ matrix.arch }}.tar.zst" not in job:
            raise AssertionError(f"{name}: archive architecture mismatch")
        if name != "build-storage-image" and (
            "scripts/check_native_architecture.sh ${{ matrix.arch }} databasus-storage-test:${{ github.sha }}-${{ matrix.arch }}"
            not in job
        ):
            raise AssertionError(f"{name}: candidate is not checked before scenarios")
    for name in ("build-storage-image", "build-verification-image"):
        if "platforms: linux/${{ matrix.arch }}" not in jobs[name]:
            raise AssertionError(f"{name}: build platform mismatch")


class NativeReleaseTest(unittest.TestCase):
    def test_covers_every_native_suite_and_matching_candidate(self) -> None:
        check_release_coverage((ROOT / ".github/workflows/ci-release.yml").read_text())

    def test_rejects_incomplete_matrices_and_bypassed_failures(self) -> None:
        workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()
        mutations = [
            (
                workflow.replace('-parallel="$TEST_PARALLELISM" ', ""),
                "container concurrency is unbounded",
            ),
            (
                workflow.replace("TEST_PARALLELISM: '2'", "TEST_PARALLELISM: '8'"),
                "container concurrency is unbounded",
            ),
            (workflow.replace("- arch: arm64", "- arch: amd64", 1), "incomplete native matrix"),
            (
                workflow.replace("- group: physical-3-of-3", "- group: physical-2-of-3", 1),
                "incomplete native matrix",
            ),
            (workflow.replace("        test-scripts,\n", ""), "disconnected release gate"),
            (
                workflow.replace(
                    "- name: Run frontend tests\n",
                    "- name: Run frontend tests\n        continue-on-error: true\n",
                ),
                "test failure is allowed",
            ),
            (
                workflow.replace(
                    "ARTIFACT_NAME: databasus-storage-test-${{ matrix.arch }}",
                    "ARTIFACT_NAME: databasus-storage-test-amd64",
                    1,
                ),
                "artifact architecture mismatch",
            ),
            (
                workflow.replace(
                    "          name: ${{ env.ARTIFACT_NAME }}", "          name: wrong"
                ),
                "artifact architecture mismatch",
            ),
            (
                workflow.replace("      - *download-artifact-2\n", "", 1),
                "artifact architecture mismatch",
            ),
        ]
        for changed, diagnostic in mutations:
            with (
                self.subTest(diagnostic=diagnostic),
                self.assertRaisesRegex(AssertionError, diagnostic),
            ):
                check_release_coverage(changed)

    def test_rejects_missing_duplicate_disconnected_and_non_native_new_selections(self) -> None:
        workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()
        mutations = [
            (
                "test-backend",
                "- group: other-3-of-3",
                "- group: other-2-of-3",
                "incomplete native matrix",
            ),
            (
                "test-backend",
                "- group: storage-providers",
                "- group: other-1-of-3",
                "incomplete native matrix",
            ),
            ("test-frontend-browser-ui", "shard: 3", "shard: 2", "incomplete native matrix"),
            (
                "test-e2e-browser-lifecycle",
                "group: isolation",
                "group: failures",
                "incomplete native matrix",
            ),
            (
                "test-e2e-browser-lifecycle",
                "needs: [build-storage-image]",
                "needs: [lint-backend]",
                "disconnected candidate prerequisite",
            ),
            (
                "determine-version",
                "        test-e2e-browser-lifecycle,\n",
                "",
                "disconnected release gate",
            ),
            (
                "test-e2e-browser-lifecycle",
                "run: mise run e2e:lifecycle",
                "continue-on-error: true\n        run: mise run e2e:lifecycle",
                "test failure is allowed",
            ),
            (
                "test-e2e-browser-lifecycle",
                "ARTIFACT_NAME: databasus-storage-test-${{ matrix.arch }}",
                "ARTIFACT_NAME: databasus-storage-test-amd64",
                "artifact architecture mismatch",
            ),
            (
                "test-e2e-browser-lifecycle",
                "DATABASUS_IMAGE: databasus-storage-test:${{ github.sha }}-${{ matrix.arch }}",
                "DATABASUS_IMAGE: databasus-storage-test:${{ github.sha }}-amd64",
                "scenario candidate architecture mismatch",
            ),
            (
                "test-e2e-browser-lifecycle",
                "scripts/check_native_architecture.sh ${{ matrix.arch }} databasus-storage-test:${{ github.sha }}-${{ matrix.arch }}",
                "scripts/check_native_architecture.sh ${{ matrix.arch }} databasus-storage-test:${{ github.sha }}-amd64",
                "candidate is not checked",
            ),
            (
                "test-e2e-browser-lifecycle",
                "    timeout-minutes: 30",
                "    if: always()\n    timeout-minutes: 30",
                "required suite can be skipped",
            ),
        ]
        for job_name, original, replacement, diagnostic in mutations:
            with self.subTest(job=job_name, diagnostic=diagnostic):
                matched_job = re.search(
                    r"^  " + job_name + r":\n.*?(?=^  [a-zA-Z0-9_-]+:|\Z)", workflow, re.M | re.S
                )
                assert matched_job is not None
                job = matched_job.group()
                self.assertIn(original, job)
                changed = (
                    workflow[: matched_job.start()]
                    + job.replace(original, replacement, 1)
                    + workflow[matched_job.end() :]
                )
                with self.assertRaisesRegex(AssertionError, diagnostic):
                    check_release_coverage(changed)

        for job_name, axis in (
            ("test-backend", "group"),
            ("test-frontend-browser-ui", "arch"),
            ("test-e2e-browser-lifecycle", "arch"),
        ):
            with self.subTest(job=job_name, mutation="missing leg"):
                matched_job = re.search(
                    r"^  " + job_name + r":\n.*?(?=^  [a-zA-Z0-9_-]+:|\Z)", workflow, re.M | re.S
                )
                assert matched_job is not None
                job = matched_job.group()
                changed_job, count = re.subn(
                    r"          - " + axis + r": [^\n]+\n(?:            [^\n]+\n)+",
                    "",
                    job,
                    count=1,
                )
                self.assertEqual(count, 1)
                changed = (
                    workflow[: matched_job.start()] + changed_job + workflow[matched_job.end() :]
                )
                with self.assertRaisesRegex(AssertionError, "incomplete native matrix"):
                    check_release_coverage(changed)


class NativeArchitectureTest(unittest.TestCase):
    def test_rejects_wrong_host_engine_and_candidate_before_scenarios(self) -> None:
        for host, engine, image, architecture, status in [
            ("x86_64", "x86_64", "linux/amd64", "amd64", 0),
            ("aarch64", "aarch64", "linux/arm64", "arm64", 0),
            ("x86_64", "aarch64", "linux/arm64", "arm64", 1),
            ("aarch64", "x86_64", "linux/arm64", "arm64", 1),
            ("aarch64", "aarch64", "linux/amd64", "arm64", 1),
            ("aarch64", "aarch64", "", "arm64", 1),
        ]:
            with (
                self.subTest(host=host, engine=engine, image=image),
                tempfile.TemporaryDirectory() as directory,
            ):
                binary = Path(directory)
                (binary / "uname").write_text(f"#!/bin/sh\necho {host}\n")
                (binary / "docker").write_text(
                    '#!/bin/sh\nif [ "$1" = info ]; then echo "$ENGINE"; '
                    'elif [ -n "$IMAGE" ]; then echo "$IMAGE"; else exit 1; fi\n'
                )
                for path in binary.iterdir():
                    path.chmod(0o755)
                invocation = subprocess.run(
                    [
                        "bash",
                        str(ROOT / "scripts/check_native_architecture.sh"),
                        architecture,
                        "candidate:commit-" + architecture,
                    ],
                    env={
                        **os.environ,
                        "PATH": f"{binary}:{os.environ['PATH']}",
                        "ENGINE": engine,
                        "IMAGE": image,
                    },
                    capture_output=True,
                    text=True,
                    check=False,
                )
                self.assertEqual(
                    invocation.returncode, status, invocation.stdout + invocation.stderr
                )


class ImagePreparationTest(unittest.TestCase):
    def test_uses_the_same_policy_and_fails_for_other_missing_images(self) -> None:
        policy = json.loads(
            (ROOT / "backend/internal/lib/testcontainers/unavailable_images.json").read_text()
        )
        for architecture in ("x86_64", "aarch64"):
            for failing_image in ("", "mysql:8.0"):
                with (
                    self.subTest(architecture=architecture, failure=failing_image),
                    tempfile.TemporaryDirectory() as directory,
                ):
                    binary = Path(directory)
                    pull_log = binary / "pulled"
                    (binary / "uname").write_text(f"#!/bin/sh\necho {architecture}\n")
                    (binary / "sleep").write_text("#!/bin/sh\nexit 0\n")
                    (binary / "docker").write_text(
                        '#!/bin/sh\necho "$2" >> "$PULL_LOG"\n[ "$2" != "$FAILING_IMAGE" ]\n'
                    )
                    for path in binary.iterdir():
                        path.chmod(0o755)
                    invocation = subprocess.run(
                        ["bash", str(ROOT / "mise-tasks/backend/pull-testcontainers")],
                        cwd=ROOT / "backend",
                        env={
                            **os.environ,
                            "PATH": f"{binary}:{os.environ['PATH']}",
                            "PULL_LOG": str(pull_log),
                            "FAILING_IMAGE": failing_image,
                        },
                        text=True,
                        capture_output=True,
                        check=False,
                    )
                    self.assertEqual(
                        invocation.returncode == 0, not failing_image, invocation.stderr
                    )
                    images = set(pull_log.read_text().splitlines())
                    self.assertIn("mysql:8.0", images)
                    self.assertIn("mariadb:10.1", images)
                    for image, reason in policy["arm64"].items():
                        if architecture == "aarch64":
                            self.assertNotIn(image, images)
                            self.assertIn(reason, invocation.stderr)
                        else:
                            self.assertIn(image, images)


class NfsFixtureTest(unittest.TestCase):
    def test_checks_host_prerequisites_without_pulling_a_run_local_image(self) -> None:
        suite = (ROOT / "e2e/docker-storage/run.sh").read_text()
        prerequisites = suite.split("run_network_prerequisites() {", 1)[1].split("\n}", 1)[0]
        with tempfile.TemporaryDirectory() as directory:
            binary = Path(directory)
            for name in ("mount.cifs", "mount.nfs"):
                executable = binary / name
                executable.write_text("#!/bin/sh\nexit 0\n")
                executable.chmod(0o755)
            docker = binary / "docker"
            docker.write_text("#!/bin/sh\necho 'unexpected fixture pull' >&2\nexit 1\n")
            docker.chmod(0o755)
            completed = subprocess.run(
                ["bash", "-eu", "-c", prerequisites],
                env={**os.environ, "PATH": f"{binary}:{os.environ['PATH']}"},
                capture_output=True,
                text=True,
            )

        self.assertEqual(completed.returncode, 0, completed.stderr)

    def test_preserves_native_exports_and_run_scoped_cleanup(self) -> None:
        fixture = ROOT / "e2e/docker-storage/nfs"
        dockerfile = (fixture / "Dockerfile").read_text()
        self.assertIn("debian:bookworm-slim@sha256:", dockerfile)
        self.assertNotIn("--platform=", dockerfile)
        self.assertIn("nfs-kernel-server", dockerfile)
        self.assertEqual(
            (fixture / "exports").read_text(),
            "/nfsshare *(rw,fsid=0,sync,no_subtree_check,root_squash)\n",
        )
        suite = (ROOT / "e2e/docker-storage/run.sh").read_text()
        self.assertIn('docker build --tag "${nfs_fixture_image}" "${suite_directory}/nfs"', suite)
        self.assertIn('docker image rm "${nfs_fixture_image}"', suite)
        self.assertIn('nfs_fixture_image="databasus-nfs-fixture:${run_id}"', suite)
        self.assertIn("root-squashed NFS accepted a client-root ownership change", suite)
        workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()
        self.assertIn(
            "trivy config --severity HIGH,CRITICAL --exit-code 1 e2e/docker-storage/nfs/Dockerfile",
            workflow,
        )

    def test_reports_readiness_only_after_export_and_propagates_startup_failure(self) -> None:
        for failing_command in ("", "mount", "rpc.nfsd", "exportfs", "rpcinfo", "rpc.mountd"):
            with self.subTest(failure=failing_command), tempfile.TemporaryDirectory() as directory:
                binary = Path(directory)
                log = binary / "commands"
                # Only external privileged commands are replaced; the fixture entry point runs unchanged.
                for command in (
                    "mount",
                    "rpcbind",
                    "rpc.nfsd",
                    "exportfs",
                    "rpcinfo",
                    "rpc.mountd",
                ):
                    (binary / command).write_text(
                        '#!/bin/bash\ncommand="${0##*/}"\necho "$command $*" >> "$COMMAND_LOG"\n'
                        'if [[ "$command" == rpc.nfsd && "$*" == *"--no-nfs-version 2"* ]]; then\n'
                        '  echo "2: Unsupported version" >&2; exit 2\n'
                        "fi\n"
                        'if [[ "$command" == "$FAILING_COMMAND" ]]; then exit 17; fi\n'
                        'if [[ "$command" == rpc.mountd ]]; then\n'
                        '  while ! grep -q "rpcinfo" "$COMMAND_LOG"; do :; done\n'
                        "fi\n"
                    )
                    (binary / command).chmod(0o755)
                invocation = subprocess.run(
                    ["bash", str(ROOT / "e2e/docker-storage/nfs/start.sh")],
                    env={
                        **os.environ,
                        "PATH": f"{binary}:{os.environ['PATH']}",
                        "COMMAND_LOG": str(log),
                        "FAILING_COMMAND": failing_command,
                    },
                    text=True,
                    capture_output=True,
                    check=False,
                    timeout=10,
                )
                self.assertEqual(
                    invocation.returncode, 17 if failing_command else 0, invocation.stderr
                )
                commands = log.read_text()
                if not failing_command:
                    self.assertIn("Startup successful.", invocation.stdout)
                    self.assertLess(commands.index("exportfs -rav"), commands.index("rpcinfo"))
                    self.assertIn("rpc.nfsd 0", commands)
                    self.assertIn("exportfs -au", commands)
                elif failing_command in {"mount", "rpc.nfsd", "exportfs", "rpcinfo"}:
                    self.assertNotIn("Startup successful.", invocation.stdout)
