import re
import unittest
from dataclasses import dataclass
from pathlib import Path

from workflow_jobs import get_jobs, get_steps

ROOT = Path(__file__).resolve().parents[1]
UPLOAD_ACTION = "actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a"
DOWNLOAD_ACTION = "actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c"


@dataclass(frozen=True)
class ArtifactTransfer:
    direction: str
    artifact_name: str
    path: str
    timeout_minutes: int = 3
    compression_level: int = 6
    pattern: str = "''"
    is_merge_multiple: bool = False


CANDIDATE_DOWNLOAD = ArtifactTransfer(
    direction="download",
    artifact_name="databasus-storage-test-${{ matrix.arch }}",
    path="/tmp",
)
TRANSFERS = {
    "test-e2e-browser": CANDIDATE_DOWNLOAD,
    "test-e2e-browser-lifecycle": CANDIDATE_DOWNLOAD,
    "test-filesystems": CANDIDATE_DOWNLOAD,
    "test-upgrade": CANDIDATE_DOWNLOAD,
    "build-storage-image": ArtifactTransfer(
        direction="upload",
        artifact_name="databasus-storage-test-${{ matrix.arch }}",
        path="/tmp/databasus-storage-test-${{ matrix.arch }}.tar.zst",
        timeout_minutes=5,
        compression_level=0,
    ),
    "push-image": ArtifactTransfer(
        direction="upload",
        artifact_name="digests-release-${{ matrix.arch }}",
        path="/tmp/digests/*",
    ),
    "push-verification-image": ArtifactTransfer(
        direction="upload",
        artifact_name="digests-verification-${{ matrix.major }}-${{ matrix.arch }}",
        path="/tmp/digests/*",
    ),
    "merge-image-manifest": ArtifactTransfer(
        direction="download",
        artifact_name="''",
        path="/tmp/digests",
        pattern="digests-release-*",
        is_merge_multiple=True,
    ),
    "merge-verification-image-manifest": ArtifactTransfer(
        direction="download",
        artifact_name="''",
        path="/tmp/digests",
        pattern="digests-verification-${{ matrix.major }}-*",
        is_merge_multiple=True,
    ),
}


class ArtifactTransferTest(unittest.TestCase):
    def setUp(self) -> None:
        self.workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()

    def assert_transfer(self, jobs: dict[str, str], job_name: str) -> None:
        transfer = TRANSFERS[job_name]
        job = jobs[job_name]
        action = UPLOAD_ACTION if transfer.direction == "upload" else DOWNLOAD_ACTION
        steps = [
            step
            for step in get_steps(job)
            if f"uses: actions/{transfer.direction}-artifact@" in step
        ]
        self.assertEqual(len(steps), 3, job_name)

        settings = {"ARTIFACT_NAME": transfer.artifact_name, "ARTIFACT_PATH": transfer.path}
        if transfer.direction == "upload":
            settings |= {
                "ARTIFACT_TIMEOUT_MINUTES": f"'{transfer.timeout_minutes}'",
                "ARTIFACT_COMPRESSION_LEVEL": f"'{transfer.compression_level}'",
            }
        else:
            settings |= {
                "ARTIFACT_PATTERN": transfer.pattern,
                "ARTIFACT_MERGE_MULTIPLE": f"'{str(transfer.is_merge_multiple).lower()}'",
            }
        for key, value in settings.items():
            self.assertRegex(job, rf"(?m)^      {key}: {re.escape(value)}$")

        for attempt, step in enumerate(steps, 1):
            step_id = f"{transfer.direction}-artifact-{attempt}"
            self.assertRegex(step, rf"(?m)^        id: {step_id}$")
            self.assertRegex(step, rf"(?m)^        uses: {re.escape(action)} # v\d+\.\d+\.\d+$")
            self.assertNotRegex(step, r"(?m)^        env:")
            self.assertIn("          name: ${{ env.ARTIFACT_NAME }}\n", step)
            self.assertIn("          path: ${{ env.ARTIFACT_PATH }}\n", step)
            conditions = re.findall(r"^        if: (.*)$", step, re.M)
            self.assertEqual(
                conditions,
                []
                if attempt == 1
                else [f"steps.{transfer.direction}-artifact-{attempt - 1}.outcome == 'failure'"],
            )
            allowed_failures = re.findall(r"^        continue-on-error: (.*)$", step, re.M)
            self.assertEqual(allowed_failures, ["true"] if attempt < 3 else [])

            if transfer.direction == "upload":
                self.assertIn(
                    "        timeout-minutes: ${{ fromJSON(env.ARTIFACT_TIMEOUT_MINUTES) }}\n",
                    step,
                )
                for field in (
                    "compression-level: ${{ env.ARTIFACT_COMPRESSION_LEVEL }}",
                    "retention-days: 1",
                    "if-no-files-found: error",
                    "overwrite: true",
                ):
                    self.assertIn(f"          {field}\n", step)
            else:
                self.assertIn("        timeout-minutes: 3\n", step)
                self.assertIn("          pattern: ${{ env.ARTIFACT_PATTERN }}\n", step)
                self.assertIn(
                    "          merge-multiple: ${{ env.ARTIFACT_MERGE_MULTIPLE }}\n", step
                )

    def test_preserves_artifact_settings_and_three_fatal_bounded_attempts(self) -> None:
        jobs = get_jobs(self.workflow)
        for job_name in TRANSFERS:
            with self.subTest(job=job_name):
                self.assert_transfer(jobs, job_name)

    def test_rejects_unsafe_changes_in_shared_steps_and_caller_settings(self) -> None:
        for original, replacement, job_name in (
            (
                "if: steps.download-artifact-1.outcome == 'failure'",
                "if: always()",
                "test-filesystems",
            ),
            (
                "id: download-artifact-3\n",
                "id: download-artifact-3\n        continue-on-error: true\n",
                "test-upgrade",
            ),
            ("        timeout-minutes: 3\n", "", "test-filesystems"),
            (
                "      - *download-artifact-2\n",
                "      - *download-artifact-1\n",
                "test-filesystems",
            ),
            (
                "ARTIFACT_NAME: databasus-storage-test-${{ matrix.arch }}",
                "ARTIFACT_NAME: wrong",
                "test-e2e-browser",
            ),
            (
                "ARTIFACT_TIMEOUT_MINUTES: '5'",
                "ARTIFACT_TIMEOUT_MINUTES: '3'",
                "build-storage-image",
            ),
            (
                "ARTIFACT_COMPRESSION_LEVEL: '0'",
                "ARTIFACT_COMPRESSION_LEVEL: '6'",
                "build-storage-image",
            ),
            (UPLOAD_ACTION, "actions/upload-artifact@main", "build-storage-image"),
            ("ARTIFACT_TIMEOUT_MINUTES: '3'", "ARTIFACT_TIMEOUT_MINUTES: '5'", "push-image"),
            ("ARTIFACT_COMPRESSION_LEVEL: '6'", "ARTIFACT_COMPRESSION_LEVEL: '0'", "push-image"),
            (
                "digests-verification-${{ matrix.major }}-${{ matrix.arch }}",
                "digests-verification-${{ matrix.arch }}",
                "push-verification-image",
            ),
            (
                "ARTIFACT_PATTERN: digests-release-*",
                "ARTIFACT_PATTERN: digests-*",
                "merge-image-manifest",
            ),
            (
                "ARTIFACT_MERGE_MULTIPLE: 'true'",
                "ARTIFACT_MERGE_MULTIPLE: 'false'",
                "merge-image-manifest",
            ),
            (
                "digests-verification-${{ matrix.major }}-*",
                "digests-verification-*",
                "merge-verification-image-manifest",
            ),
        ):
            with self.subTest(change=original):
                self.assertIn(original, self.workflow)
                with self.assertRaises(AssertionError):
                    jobs = get_jobs(self.workflow.replace(original, replacement))
                    self.assert_transfer(jobs, job_name)
