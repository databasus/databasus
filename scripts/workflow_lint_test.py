from pathlib import Path
import sys
import tomllib
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parent))

from workflow_jobs import get_jobs  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]


class WorkflowLintTest(unittest.TestCase):
    def test_workflow_lint_runs_locally_and_in_ci(self):
        tasks = tomllib.loads((ROOT / "mise.toml").read_text())["tasks"]
        workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()

        self.assertIn("lint-workflows", tasks["lint"]["run"])
        self.assertIn("actionlint -shellcheck=", tasks["lint-workflows"]["run"])
        self.assertTrue((ROOT / ".github/actionlint.yaml").is_file())

        lint_workflows = get_jobs(workflow)["lint-workflows"]
        self.assertIn("mise run lint-workflows", lint_workflows)
        self.assertNotRegex(lint_workflows, r"(?m)^    if:")


if __name__ == "__main__":
    unittest.main()
