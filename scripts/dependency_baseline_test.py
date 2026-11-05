import os
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path

from pattern_match import get_required_match

WORKFLOW = Path(__file__).resolve().parents[1] / ".github/workflows/dependency-review.yml"


class DependencyBaselineTest(unittest.TestCase):
    def setUp(self) -> None:
        workspace = tempfile.TemporaryDirectory()
        self.addCleanup(workspace.cleanup)
        self.workspace = Path(workspace.name)
        self.output_path = self.workspace / "output"
        self.environment = {
            **os.environ,
            "GITHUB_OUTPUT": str(self.output_path),
            "GIT_AUTHOR_NAME": "Dependency review test",
            "GIT_AUTHOR_EMAIL": "test@example.test",
            "GIT_COMMITTER_NAME": "Dependency review test",
            "GIT_COMMITTER_EMAIL": "test@example.test",
        }
        self.git("init", "--initial-branch=main")
        self.git("commit", "--allow-empty", "-m", "Initial release")

    def git(self, *arguments: str) -> str:
        return subprocess.check_output(
            ["git", *arguments],
            cwd=self.workspace,
            env=self.environment,
            text=True,
            stderr=subprocess.PIPE,
        ).strip()

    def run_baseline_selection(self) -> subprocess.CompletedProcess[str]:
        script = get_required_match(
            r"        run: \|\n((?:          .*\n)+)", WORKFLOW.read_text()
        ).group(1)

        return subprocess.run(
            ["bash", "-e", "-o", "pipefail", "-c", textwrap.dedent(script)],
            cwd=self.workspace,
            env=self.environment,
            capture_output=True,
            text=True,
        )

    def test_uses_a_reachable_release_without_following_unrelated_tags(self) -> None:
        release_commit = self.git("rev-parse", "HEAD")
        self.git("tag", "v1.0.0")
        self.git("commit", "--allow-empty", "-m", "Development changes")
        checked_commit = self.git("rev-parse", "HEAD")
        self.git("tag", "unrelated-tag")
        self.git("checkout", "-b", "unmerged-release")
        self.git("commit", "--allow-empty", "-m", "Unmerged changes")
        self.git("tag", "v9.0.0")
        self.git("checkout", "--detach", checked_commit)

        selection = self.run_baseline_selection()

        self.assertEqual(selection.returncode, 0, selection.stderr)
        self.assertEqual(self.output_path.read_text(), f"commit={release_commit}\n")

    def test_fails_without_a_release_instead_of_silently_skipping_dependency_review(self) -> None:
        selection = self.run_baseline_selection()

        self.assertNotEqual(selection.returncode, 0)
        self.assertFalse(self.output_path.exists())


if __name__ == "__main__":
    unittest.main()
