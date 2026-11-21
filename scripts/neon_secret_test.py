import re
import unittest
from pathlib import Path

from workflow_jobs import get_jobs, get_steps

ROOT = Path(__file__).resolve().parents[1]
SECRET_REFERENCE = "secrets.TEST_NEON_DATABASE_URL"
GROUP_CONDITIONED_VALUE = re.compile(
    r"^\$\{\{ matrix\.group == '(?P<group>[a-z0-9-]+)' && (?P<value>[^|]+?) \|\| '' \}\}$"
)


def get_step_environment(step: str) -> dict[str, str]:
    environment_block = re.search(r"^        env:\n((?:          [^\n]*\n|\s*\n)*)", step, re.M)
    if environment_block is None:
        return {}

    environment: dict[str, str] = {}
    for line in environment_block.group(1).splitlines():
        variable = re.match(r"^          ([A-Z0-9_]+): (.+)$", line)
        if variable is not None:
            environment[variable.group(1)] = variable.group(2).strip()

    return environment


def evaluate_for_group(expression: str, group: str) -> str:
    conditioned = GROUP_CONDITIONED_VALUE.match(expression)
    if conditioned is None:
        raise AssertionError(f"not a value conditioned on one matrix group: {expression}")

    return conditioned.group("value") if conditioned.group("group") == group else "''"


class NeonSecretTest(unittest.TestCase):
    def setUp(self) -> None:
        self.workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()
        self.job = get_jobs(self.workflow)["test-backend"]
        self.groups = sorted(set(re.findall(r"^          - group: (\S+)$", self.job, re.M)))
        self.go_test_step = next(
            step for step in get_steps(self.job) if step.startswith("      - name: Run Go tests\n")
        )

    def test_names_the_secret_only_in_the_go_test_step(self) -> None:
        self.assertEqual(1, self.workflow.count(SECRET_REFERENCE))
        self.assertIn(SECRET_REFERENCE, self.go_test_step)

        job_header = self.job.split("    steps:\n", 1)[0]
        self.assertNotIn("TEST_NEON_DATABASE_URL", job_header)

    def test_gives_the_secret_and_the_requirement_only_to_the_logical_group(self) -> None:
        environment = get_step_environment(self.go_test_step)

        self.assertIn("logical", self.groups)

        for group in self.groups:
            with self.subTest(group=group):
                connection_url = evaluate_for_group(environment["TEST_NEON_DATABASE_URL"], group)
                is_required = evaluate_for_group(environment["TEST_IS_NEON_REQUIRED"], group)

                if group == "logical":
                    self.assertEqual(SECRET_REFERENCE, connection_url)
                    self.assertEqual("'true'", is_required)
                else:
                    self.assertEqual("''", connection_url)
                    self.assertEqual("''", is_required)


if __name__ == "__main__":
    unittest.main()
