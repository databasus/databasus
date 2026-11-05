import re
import sys
import tomllib
import unittest
from pathlib import Path

from pattern_match import get_required_match

sys.path.insert(0, str(Path(__file__).resolve().parent))

from workflow_jobs import get_jobs

ROOT = Path(__file__).resolve().parents[1]
CI_RELEASE = (ROOT / ".github/workflows/ci-release.yml").read_text()
CODE_SCANNING = (ROOT / ".github/workflows/codeql.yml").read_text()
TASKS = tomllib.loads((ROOT / "mise.toml").read_text())["tasks"]

PUBLISHING_JOBS = (
    "push-image",
    "merge-image-manifest",
    "push-verification-image",
    "merge-verification-image-manifest",
    "release",
    "publish-helm-chart",
)
RELEASE_GATE_JOBS = {
    "code-scanning",
    "build-image",
    "build-verification-image",
    "dockerfile-scan",
    "dependency-review",
    "lint-backend",
    "lint-frontend",
    "lint-python",
    "lint-shell",
    "test-scripts",
    "lint-workflows",
    "test-backend",
    "test-frontend",
    "test-frontend-browser-ui",
    "test-e2e-browser",
    "test-verification-agent",
    "e2e-verification-agent-podman",
    "test-filesystems",
    "test-upgrade",
}


def get_needs(job: str) -> set[str]:
    listed_needs = re.search(r"^    needs:\s*\[([^]]+)\]", job, re.M)
    if listed_needs:
        return set(re.findall(r"[a-zA-Z0-9_-]+", listed_needs.group(1)))

    single_need = re.search(r"^    needs: ([a-zA-Z0-9_-]+)$", job, re.M)
    return {single_need.group(1)} if single_need else set()


def get_transitive_needs(jobs: dict[str, str], job_name: str) -> set[str]:
    reached_jobs = set()
    jobs_to_visit = [job_name]
    while jobs_to_visit:
        for needed_job in get_needs(jobs[jobs_to_visit.pop()]):
            if needed_job not in reached_jobs:
                reached_jobs.add(needed_job)
                jobs_to_visit.append(needed_job)

    return reached_jobs


class ReleaseGateTest(unittest.TestCase):
    def test_python_quality_is_shared_and_required_before_every_publication(self) -> None:
        python_workflow = (ROOT / ".github/workflows/lint-python.yml").read_text()
        lint_job = get_jobs(python_workflow)["lint-python"]
        self.assertIn("workflow_call:", python_workflow)
        self.assertIn("permissions:\n  contents: read\n", python_workflow)
        self.assertIn("MISE_TASK_RUN_AUTO_INSTALL: 'false'", python_workflow)
        self.assertIn("timeout-minutes:", lint_job)
        self.assertIn("install_args: 'python uv ruff ty'", lint_job)
        self.assertIn("run: mise run python:lint", lint_job)
        self.assertNotIn("continue-on-error", lint_job)
        self.assertNotRegex(lint_job, r"(?m)^\s+if:")
        self.assertNotIn("secrets.", python_workflow)
        for action in re.findall(r"uses: (\S+)", lint_job):
            self.assertRegex(action, r"@[a-f0-9]{40}$")

        for workflow_name, publishing_jobs in (
            ("ci-release.yml", PUBLISHING_JOBS),
            ("deploy-dev-image.yml", ("push-dev-image", "merge-dev-image-manifest")),
        ):
            jobs = get_jobs((ROOT / ".github/workflows" / workflow_name).read_text())
            self.assertIn("uses: ./.github/workflows/lint-python.yml", jobs["lint-python"])
            self.assertNotRegex(jobs["lint-python"], r"(?m)^    (if|continue-on-error):")
            for job_name in publishing_jobs:
                with self.subTest(workflow=workflow_name, job=job_name):
                    self.assertIn("lint-python", get_transitive_needs(jobs, job_name))
                    self.assertNotRegex(jobs[job_name], r"always\(\)|failure\(\)|cancelled\(\)")
                    self.assertNotRegex(jobs[job_name], r"(?m)^    continue-on-error:")

    def test_script_test_job_installs_the_real_python_and_hook_tools(self) -> None:
        test_job = get_jobs(CI_RELEASE)["test-scripts"]
        installed_tools = get_required_match(r"install_args: '([^']+)'", test_job).group(1).split()
        self.assertTrue({"python", "uv", "ruff", "ty", "hk"}.issubset(installed_tools))

    def test_publishing_jobs_reach_every_gate_job(self) -> None:
        jobs = get_jobs(CI_RELEASE)

        for publishing_job in PUBLISHING_JOBS:
            with self.subTest(publishing_job=publishing_job):
                reached_jobs = get_transitive_needs(jobs, publishing_job)
                self.assertEqual(RELEASE_GATE_JOBS - reached_jobs, set())

    def test_version_selection_waits_for_the_whole_gate(self) -> None:
        jobs = get_jobs(CI_RELEASE)

        self.assertEqual(RELEASE_GATE_JOBS - get_transitive_needs(jobs, "determine-version"), set())

    def test_browser_end_to_end_check_waits_only_for_its_image(self) -> None:
        jobs = get_jobs(CI_RELEASE)

        self.assertEqual(get_transitive_needs(jobs, "test-e2e-browser"), {"build-storage-image"})

    def test_upgrade_check_waits_only_for_its_image_and_tests_it(self) -> None:
        jobs = get_jobs(CI_RELEASE)

        self.assertEqual(get_transitive_needs(jobs, "test-upgrade"), {"build-storage-image"})
        self.assertRegex(
            jobs["test-upgrade"],
            r"DATABASUS_IMAGE: databasus-storage-test:\$\{\{ github\.sha \}\}-\$\{\{ matrix\.arch \}\}\n\s+run: mise run backend:test-upgrade\n",
        )

    def test_backend_matrix_reports_every_group_and_every_shard(self) -> None:
        test_backend = get_jobs(CI_RELEASE)["test-backend"]

        self.assertEqual(
            set(re.findall(r"- group: (\S+)", test_backend)),
            {
                "other",
                "agent-e2e",
                "logical",
                "physical-1-of-3",
                "physical-2-of-3",
                "physical-3-of-3",
                "physical-api-1-of-3",
                "physical-api-2-of-3",
                "physical-api-3-of-3",
            },
        )

    def test_jobs_after_the_gate_run_only_when_it_passed(self) -> None:
        jobs = get_jobs(CI_RELEASE)

        for job_name in ("determine-version", *PUBLISHING_JOBS):
            with self.subTest(job_name=job_name):
                job_condition = get_required_match(r"^    if: (.*)$", jobs[job_name], re.M).group(1)
                self.assertNotRegex(
                    job_condition, r"always\(\)|cancelled\(\)|failure\(\)|success\(\)"
                )
                self.assertNotRegex(jobs[job_name], r"(?m)^    continue-on-error:")

    def test_build_image_checks_both_architectures(self) -> None:
        build_image = get_jobs(CI_RELEASE)["build-image"]

        self.assertIn("${{ matrix.runner }}", build_image)
        self.assertIn("platforms: ${{ matrix.platform }}", build_image)
        self.assertNotIn("platforms: linux/amd64", build_image)

        arm64_runner = "namespace-profile-2vcpu-4ram-ubuntu-26-arm"
        amd64_runner = "namespace-profile-4vcpu-8ram-ubuntu-26-x64"
        self.assertEqual(
            dict(re.findall(r"platform: (\S+)\n\s+runner: (\S+)", build_image)),
            {"linux/amd64": amd64_runner, "linux/arm64": arm64_runner},
        )

    def test_application_image_runs_the_image_check(self) -> None:
        build_image = get_jobs(CI_RELEASE)["build-image"]

        self.assertIn("scripts/check_image.sh", build_image)
        self.assertNotIn("Run smoke pings", build_image)

    def test_code_scanning_runs_as_part_of_the_release_on_main(self) -> None:
        code_scanning = get_jobs(CI_RELEASE)["code-scanning"]

        self.assertNotRegex(code_scanning, r"(?m)^    if:")
        self.assertIn("uses: ./.github/workflows/codeql.yml", code_scanning)
        self.assertIn("is-release-gate: true", code_scanning)
        self.assertIn("security-events: write", code_scanning)
        self.assertIn("actions: read", code_scanning)

    def test_codeql_is_callable_and_not_triggered_by_push(self) -> None:
        triggers = CODE_SCANNING[
            CODE_SCANNING.index("on:\n") : CODE_SCANNING.index("\npermissions:")
        ]

        self.assertIn("workflow_call:", triggers)
        self.assertIn("is-release-gate:", triggers)
        self.assertNotIn("pull_request:", triggers)
        self.assertIn("schedule:", triggers)
        self.assertNotIn("push:", triggers)

    def test_manual_scans_do_not_share_concurrency_with_releases_or_schedules(self) -> None:
        self.assertIn("group: codeql-${{ github.event_name }}-${{ github.ref }}", CODE_SCANNING)
        self.assertIn(
            "cancel-in-progress: ${{ github.event_name == 'workflow_dispatch' }}", CODE_SCANNING
        )

    def test_release_runs_on_main_are_not_cancelled(self) -> None:
        self.assertIn("group: ci-release-${{ github.event_name }}-${{ github.ref }}", CI_RELEASE)
        self.assertIn(
            "cancel-in-progress: ${{ github.event_name == 'workflow_dispatch' }}", CI_RELEASE
        )

    def test_independent_checks_start_without_waiting_for_lint(self) -> None:
        jobs = get_jobs(CI_RELEASE)

        for job_name in RELEASE_GATE_JOBS - {
            "test-e2e-browser",
            "test-filesystems",
            "test-upgrade",
        }:
            with self.subTest(job_name=job_name):
                self.assertEqual(get_needs(jobs[job_name]), set())

    def test_release_waits_for_both_image_manifests(self) -> None:
        jobs = get_jobs(CI_RELEASE)

        self.assertTrue(
            {"merge-image-manifest", "merge-verification-image-manifest"}.issubset(
                get_needs(jobs["release"])
            )
        )

    def test_gating_secret_scan_reads_released_history(self) -> None:
        secret_scan = get_jobs(CODE_SCANNING)["secret-scan"]

        self.assertIn("${{ inputs.is-release-gate && 'HEAD' || '--all' }}", secret_scan)
        self.assertIn("mise run security:secrets", secret_scan)
        self.assertIn('--log-opts="${SCANNED_HISTORY:---all}"', TASKS["security:secrets"]["run"])
        self.assertIn("fetch-depth: 0", secret_scan)

    def test_scans_fail_on_what_they_find(self) -> None:
        scan_jobs = get_jobs(CODE_SCANNING)

        for job_name in ("analyze", "secret-scan", "semgrep", "codeql-severity-gate"):
            with self.subTest(job_name=job_name):
                self.assertIn(job_name, scan_jobs)
                self.assertNotIn("continue-on-error", scan_jobs[job_name])

        self.assertIn("mise run security:semgrep", scan_jobs["semgrep"])
        self.assertIn("--error", TASKS["security:semgrep"]["run"])
        self.assertIn("--exit-code 1", TASKS["security:secrets"]["run"])

    def test_severity_gate_runs_after_analysis(self) -> None:
        severity_gate = get_jobs(CODE_SCANNING)["codeql-severity-gate"]

        self.assertEqual(get_needs(severity_gate), {"analyze"})
        self.assertIn("if: inputs.is-release-gate", severity_gate)
        self.assertIn("scripts/check_codeql_alerts.sh", severity_gate)
        self.assertIn("security-events: read", severity_gate)
        self.assertIn("GH_TOKEN: ${{ github.token }}", severity_gate)


if __name__ == "__main__":
    unittest.main()
