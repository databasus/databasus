from pathlib import Path
import re
import unittest

WORKFLOWS = Path(__file__).resolve().parents[1] / ".github/workflows"


class WorkflowPermissionsTest(unittest.TestCase):
    def test_workflows_grant_no_top_level_write(self):
        for workflow_path in WORKFLOWS.glob("*.yml"):
            with self.subTest(workflow=workflow_path.name):
                workflow = workflow_path.read_text()
                top_level_permissions = re.search(r"^permissions:\n((?:  .*\n)+)", workflow, re.M).group(1)
                self.assertEqual(top_level_permissions, "  contents: read\n")

    def test_dependency_review_has_read_only_permissions_without_pr_comments(self):
        workflow = (WORKFLOWS / "dependency-review.yml").read_text()
        job = workflow[workflow.index("  dependency-review:\n") :]

        self.assertNotIn("pull-requests: write", job)
        self.assertIn("comment-summary-in-pr: never", job)


if __name__ == "__main__":
    unittest.main()
