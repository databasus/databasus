import json
from pathlib import Path
import re
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parent))

from workflow_jobs import get_jobs  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]
CI_RELEASE = (ROOT / ".github/workflows/ci-release.yml").read_text()
CODE_SCANNING = (ROOT / ".github/workflows/codeql.yml").read_text()

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
    "lint-backend",
    "lint-frontend",
    "lint-shell",
    "lint-workflows",
    "test-backend",
    "test-frontend",
    "test-frontend-browser-ui",
    "test-frontend-browser-e2e",
    "test-verification-agent",
    "e2e-verification-agent-podman",
    "test-filesystems",
    "test-upgrade",
}


def get_needs(job):
    listed_needs = re.search(r"^    needs:\s*\[([^]]+)\]", job, re.M)
    if listed_needs:
        return set(re.findall(r"[a-zA-Z0-9_-]+", listed_needs.group(1)))

    single_need = re.search(r"^    needs: ([a-zA-Z0-9_-]+)$", job, re.M)
    return {single_need.group(1)} if single_need else set()


def get_transitive_needs(jobs, job_name):
    reached_jobs = set()
    jobs_to_visit = [job_name]
    while jobs_to_visit:
        for needed_job in get_needs(jobs[jobs_to_visit.pop()]):
            if needed_job not in reached_jobs:
                reached_jobs.add(needed_job)
                jobs_to_visit.append(needed_job)

    return reached_jobs


class ReleaseGateTest(unittest.TestCase):
    def test_publishing_jobs_reach_every_gate_job(self):
        jobs = get_jobs(CI_RELEASE)

        for publishing_job in PUBLISHING_JOBS:
            with self.subTest(publishing_job=publishing_job):
                reached_jobs = get_transitive_needs(jobs, publishing_job)
                self.assertEqual(RELEASE_GATE_JOBS - reached_jobs, set())

    def test_version_selection_waits_for_the_whole_gate(self):
        jobs = get_jobs(CI_RELEASE)

        self.assertEqual(RELEASE_GATE_JOBS - get_transitive_needs(jobs, "determine-version"), set())

    def test_jobs_after_the_gate_run_only_when_it_passed(self):
        jobs = get_jobs(CI_RELEASE)

        for job_name in ("determine-version", *PUBLISHING_JOBS):
            with self.subTest(job_name=job_name):
                job_condition = re.search(r"^    if: (.*)$", jobs[job_name], re.M).group(1)
                self.assertNotRegex(job_condition, r"always\(\)|cancelled\(\)|failure\(\)|success\(\)")
                self.assertNotRegex(jobs[job_name], r"(?m)^    continue-on-error:")

    def test_build_image_checks_both_architectures(self):
        build_image = get_jobs(CI_RELEASE)["build-image"]

        self.assertIn("${{ matrix.runner }}", build_image)
        self.assertIn("platforms: ${{ matrix.platform }}", build_image)
        self.assertNotIn("platforms: linux/amd64", build_image)

        matrix_expression = re.search(r"include: \$\{\{ fromJSON\((.*)\) \}\}", build_image).group(1)
        main_entries, other_entries = re.fullmatch(
            r"github\.ref == 'refs/heads/main' && '(.*)' \|\| '(.*)'", matrix_expression
        ).groups()
        arm64_runner = "namespace-profile-2vcpu-4ram-ubuntu-26-arm"
        amd64_runner = "namespace-profile-4vcpu-8ram-ubuntu-26-x64"

        self.assertEqual(
            {entry["platform"]: entry["runner"] for entry in json.loads(main_entries)},
            {"linux/amd64": amd64_runner, "linux/arm64": arm64_runner},
        )
        self.assertEqual(
            {entry["platform"]: entry["runner"] for entry in json.loads(other_entries)},
            {"linux/amd64": amd64_runner},
        )

    def test_image_jobs_run_the_image_check(self):
        jobs = get_jobs(CI_RELEASE)

        for job_name in ("build-image", "build-dev-image"):
            with self.subTest(job_name=job_name):
                self.assertIn("scripts/check_image.sh", jobs[job_name])
                self.assertNotIn("Run smoke pings", jobs[job_name])

    def test_code_scanning_runs_as_part_of_the_release_on_main(self):
        code_scanning = get_jobs(CI_RELEASE)["code-scanning"]

        self.assertIn("if: github.ref == 'refs/heads/main'", code_scanning)
        self.assertIn("uses: ./.github/workflows/codeql.yml", code_scanning)
        self.assertIn("is-release-gate: true", code_scanning)
        self.assertIn("security-events: write", code_scanning)
        self.assertIn("actions: read", code_scanning)

    def test_codeql_is_callable_and_not_triggered_by_push(self):
        triggers = CODE_SCANNING[CODE_SCANNING.index("on:\n") : CODE_SCANNING.index("\npermissions:")]

        self.assertIn("workflow_call:", triggers)
        self.assertIn("is-release-gate:", triggers)
        self.assertIn("pull_request:", triggers)
        self.assertIn("schedule:", triggers)
        self.assertNotIn("push:", triggers)

    def test_only_pull_requests_cancel_scans(self):
        self.assertIn("cancel-in-progress: ${{ github.event_name == 'pull_request' }}", CODE_SCANNING)

    def test_release_runs_on_main_are_not_cancelled(self):
        self.assertIn("group: ci-release-${{ github.ref }}", CI_RELEASE)
        self.assertRegex(CI_RELEASE, r"cancel-in-progress: \$\{\{ github\.ref != 'refs/heads/main'")

    def test_gating_secret_scan_reads_released_history(self):
        secret_scan = get_jobs(CODE_SCANNING)["secret-scan"]

        self.assertIn("${{ inputs.is-release-gate && 'HEAD' || '--all' }}", secret_scan)
        self.assertIn('--log-opts="$SCANNED_HISTORY"', secret_scan)
        self.assertIn("fetch-depth: 0", secret_scan)

    def test_scans_fail_on_what_they_find(self):
        scan_jobs = get_jobs(CODE_SCANNING)

        for job_name in ("analyze", "secret-scan", "semgrep", "codeql-severity-gate"):
            with self.subTest(job_name=job_name):
                self.assertIn(job_name, scan_jobs)
                self.assertNotIn("continue-on-error", scan_jobs[job_name])

        self.assertIn("--error", scan_jobs["semgrep"])
        self.assertIn("--exit-code 1", scan_jobs["secret-scan"])

    def test_severity_gate_runs_after_analysis(self):
        severity_gate = get_jobs(CODE_SCANNING)["codeql-severity-gate"]

        self.assertEqual(get_needs(severity_gate), {"analyze"})
        self.assertIn("if: inputs.is-release-gate", severity_gate)
        self.assertIn("scripts/check_codeql_alerts.sh", severity_gate)
        self.assertIn("security-events: read", severity_gate)
        self.assertIn("GH_TOKEN: ${{ github.token }}", severity_gate)


if __name__ == "__main__":
    unittest.main()
