from pathlib import Path
import re
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parent))

from release_gate_test import PUBLISHING_JOBS, RELEASE_GATE_JOBS, get_needs, get_transitive_needs  # noqa: E402
from workflow_jobs import get_jobs  # noqa: E402

WORKFLOWS = Path(__file__).resolve().parents[1] / ".github/workflows"


class CiEventsTest(unittest.TestCase):
    def test_checks_run_on_main_pushes_and_manual_runs_only(self):
        workflow = (WORKFLOWS / "ci-release.yml").read_text()
        triggers = workflow.split("on:\n", 1)[1].split("\npermissions:", 1)[0]

        self.assertIn("branches: [main]", triggers)
        self.assertIn("workflow_dispatch:", triggers)
        self.assertNotIn("pull_request:", triggers)
        self.assertNotIn("develop", triggers)

        jobs = get_jobs(workflow)
        for job_name in RELEASE_GATE_JOBS | {"build-storage-image"}:
            with self.subTest(job_name=job_name):
                self.assertNotRegex(jobs[job_name], r"(?m)^    if:")

    def test_manual_runs_cannot_select_versions_or_publish(self):
        jobs = get_jobs((WORKFLOWS / "ci-release.yml").read_text())
        version_condition = re.search(r"(?m)^    if: (.*)$", jobs["determine-version"]).group(1)

        self.assertEqual(version_condition, "${{ github.event_name == 'push' && github.ref == 'refs/heads/main' && !contains(github.event.head_commit.message, '[skip-release]') }}")
        for job_name in PUBLISHING_JOBS:
            with self.subTest(job_name=job_name):
                self.assertIn("determine-version", get_transitive_needs(jobs, job_name))
                self.assertIn("if: ${{ needs.determine-version.outputs.should_release == 'true' }}", jobs[job_name])

        self.assertNotIn("databasus-dev", "".join(jobs.values()))

    def test_develop_only_builds_and_publishes_both_architectures(self):
        workflow = (WORKFLOWS / "deploy-dev-image.yml").read_text()
        triggers = workflow.split("on:\n", 1)[1].split("\npermissions:", 1)[0]

        self.assertIn("branches: [develop]", triggers)
        self.assertNotIn("workflow_dispatch:", triggers)
        self.assertNotIn("pull_request:", triggers)

        jobs = get_jobs(workflow)
        self.assertEqual(set(jobs), {"push-dev-image", "merge-dev-image-manifest"})
        self.assertEqual(get_needs(jobs["push-dev-image"]), set())
        self.assertEqual(get_needs(jobs["merge-dev-image-manifest"]), {"push-dev-image"})
        self.assertEqual(set(re.findall(r"platform: (\S+)", jobs["push-dev-image"])), {"linux/amd64", "linux/arm64"})
        self.assertIn("fail-fast: false", jobs["push-dev-image"])
        self.assertNotIn("check_image.sh", workflow)
        self.assertNotIn("mise run", workflow)
        self.assertIn("databasus/databasus-dev:latest", workflow)
        self.assertIn("databasus/databasus-dev:${{ github.sha }}", workflow)
        self.assertIn("cancel-in-progress: false", workflow)

    def test_no_workflow_automatically_checks_pull_requests(self):
        for workflow_path in WORKFLOWS.glob("*.yml"):
            with self.subTest(workflow=workflow_path.name):
                triggers = workflow_path.read_text().split("on:\n", 1)[1].split("\npermissions:", 1)[0]
                self.assertNotRegex(triggers, r"(?m)^  pull_request(?:_target)?:")

    def test_manual_scans_use_the_same_gate_as_release_scans(self):
        workflow = (WORKFLOWS / "ci-release.yml").read_text()
        scan_job = get_jobs(workflow)["code-scanning"]
        scan_workflow = (WORKFLOWS / "codeql.yml").read_text()

        self.assertIn("is-release-gate: true", scan_job)
        self.assertNotIn("github.head_ref", scan_workflow)
        self.assertNotIn("refs/heads/main", scan_workflow)

    def test_dependency_review_compares_the_release_baseline_with_the_checked_commit(self):
        review_job = get_jobs((WORKFLOWS / "ci-release.yml").read_text())["dependency-review"]
        review_workflow = (WORKFLOWS / "dependency-review.yml").read_text()

        self.assertIn("uses: ./.github/workflows/dependency-review.yml", review_job)
        self.assertIn("workflow_call:", review_workflow)
        self.assertIn("fetch-depth: 0", review_workflow)
        self.assertIn("git describe --tags --match 'v[0-9]*' --abbrev=0", review_workflow)
        self.assertIn("base-ref: ${{ steps.release-baseline.outputs.commit }}", review_workflow)
        self.assertIn("head-ref: ${{ github.sha }}", review_workflow)
        self.assertIn("fail-on-severity: high", review_workflow)


if __name__ == "__main__":
    unittest.main()
