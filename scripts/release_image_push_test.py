import os
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path

from workflow_jobs import get_jobs, get_steps

ROOT = Path(__file__).resolve().parent.parent
RETRY = "scripts/retry-on-registry-rate-limit.sh"
PUSH_STEPS = {
    "push-image": "Build and push release image by digest",
    "push-verification-image": "Build and push verification image by digest",
}
MANIFEST_JOBS = ("merge-image-manifest", "merge-verification-image-manifest")


def get_step(job: str, name: str) -> str:
    for step in get_steps(job):
        if step.startswith(f"      - name: {name}\n"):
            return step
    raise AssertionError(f"no step named {name}")


def get_run_script(step: str) -> str:
    return textwrap.dedent(step.split("        run: |\n", 1)[1])


class ReleaseImagePushTest(unittest.TestCase):
    def setUp(self) -> None:
        self.jobs = get_jobs((ROOT / ".github/workflows/ci-release.yml").read_text())

    def test_runs_at_most_eight_verification_pushes_at_once(self) -> None:
        self.assertRegex(self.jobs["push-verification-image"], r"\n      max-parallel: 8\n")

    def test_pushes_through_the_rate_limit_retry(self) -> None:
        for job_name, step_name in PUSH_STEPS.items():
            with self.subTest(job=job_name):
                job = self.jobs[job_name]
                step = get_step(job, step_name)

                self.assertNotIn("docker/build-push-action", job)
                self.assertIn(f"{RETRY} docker buildx build", step)
                self.assertIn("push-by-digest=true", step)

    def test_creates_every_manifest_through_the_rate_limit_retry(self) -> None:
        calls = [
            line.strip()
            for job_name in MANIFEST_JOBS
            for line in self.jobs[job_name].splitlines()
            if "imagetools create" in line
        ]

        self.assertEqual(len(calls), 3)
        for call in calls:
            self.assertTrue(call.startswith(f'"$GITHUB_WORKSPACE/{RETRY}" '), call)
        for job_name in MANIFEST_JOBS:
            self.assertIn("uses: actions/checkout@", self.jobs[job_name])

    def run_push_step(
        self, directory: Path, job_name: str, metadata: str
    ) -> subprocess.CompletedProcess[str]:
        """Runs a push step with a stub build that writes the given metadata."""
        script = get_run_script(get_step(self.jobs[job_name], PUSH_STEPS[job_name]))
        self.assertIn("/tmp/image-metadata.json", script)
        self.assertIn("/tmp/digests", script)
        script = script.replace("/tmp/image-metadata.json", str(directory / "metadata.json"))
        script = script.replace("/tmp/digests", str(directory / "digests"))
        (directory / "metadata").write_text(metadata)
        retry = directory / RETRY
        retry.parent.mkdir(parents=True)
        retry.write_text(
            "#!/usr/bin/env bash\n"
            'metadata=$(dirname "$0")/../metadata\n'
            'while (($# > 0)); do [[ $1 == --metadata-file ]] && cp "$metadata" "$2"; shift; done\n'
        )
        retry.chmod(0o755)

        return subprocess.run(
            ["bash", "-eo", "pipefail", "-c", script],
            cwd=directory,
            env={**os.environ, "PLATFORM": "linux/amd64"},
            capture_output=True,
            text=True,
            timeout=30,
        )

    def test_push_records_the_digest_the_build_reports(self) -> None:
        for job_name in PUSH_STEPS:
            with self.subTest(job=job_name), tempfile.TemporaryDirectory() as workspace:
                directory = Path(workspace)

                execution = self.run_push_step(
                    directory, job_name, '{"containerimage.digest": "sha256:abc"}'
                )

                self.assertEqual(execution.returncode, 0, execution.stderr)
                self.assertTrue((directory / "digests/abc").exists())

    def test_push_fails_when_the_build_reports_no_digest(self) -> None:
        for job_name in PUSH_STEPS:
            with self.subTest(job=job_name), tempfile.TemporaryDirectory() as workspace:
                directory = Path(workspace)

                execution = self.run_push_step(directory, job_name, "{}")

                self.assertNotEqual(execution.returncode, 0)
                self.assertFalse((directory / "digests").exists())


if __name__ == "__main__":
    unittest.main()
