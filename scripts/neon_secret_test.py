import re
import unittest
from dataclasses import dataclass
from pathlib import Path

from workflow_jobs import get_jobs, get_steps

ROOT = Path(__file__).resolve().parents[1]


@dataclass(frozen=True)
class GroupSecrets:
    group: str
    secret_variables: tuple[str, ...]
    required_variable: str


# Each Neon test receives its secrets and its required flag only in the group that runs it.
GROUP_SECRETS = (
    GroupSecrets("logical", ("TEST_NEON_DATABASE_URL",), "TEST_IS_NEON_REQUIRED"),
    GroupSecrets(
        "storage-providers",
        (
            "TEST_NEON_S3_ENDPOINT",
            "TEST_NEON_S3_REGION",
            "TEST_NEON_S3_BUCKET",
            "TEST_NEON_S3_ACCESS_KEY",
            "TEST_NEON_S3_SECRET_KEY",
        ),
        "TEST_IS_NEON_S3_REQUIRED",
    ),
)
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


class NeonSecretsTest(unittest.TestCase):
    def setUp(self) -> None:
        self.workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()
        self.job = get_jobs(self.workflow)["test-backend"]
        self.groups = sorted(set(re.findall(r"^          - group: (\S+)$", self.job, re.M)))
        self.go_test_step = next(
            step for step in get_steps(self.job) if step.startswith("      - name: Run Go tests\n")
        )

    def test_names_each_secret_only_in_the_go_test_step(self) -> None:
        job_header = self.job.split("    steps:\n", 1)[0]

        for group_secrets in GROUP_SECRETS:
            for variable in group_secrets.secret_variables:
                with self.subTest(variable=variable):
                    secret_reference = f"secrets.{variable}"

                    self.assertEqual(1, self.workflow.count(secret_reference))
                    self.assertIn(secret_reference, self.go_test_step)
                    self.assertNotIn(variable, job_header)

    def test_gives_each_secret_and_requirement_only_to_its_group(self) -> None:
        environment = get_step_environment(self.go_test_step)

        for group_secrets in GROUP_SECRETS:
            self.assertIn(group_secrets.group, self.groups)

            for group in self.groups:
                is_owning_group = group == group_secrets.group

                for variable in group_secrets.secret_variables:
                    with self.subTest(variable=variable, group=group):
                        secret_value = evaluate_for_group(environment[variable], group)

                        self.assertEqual(
                            f"secrets.{variable}" if is_owning_group else "''", secret_value
                        )

                with self.subTest(variable=group_secrets.required_variable, group=group):
                    is_required = evaluate_for_group(
                        environment[group_secrets.required_variable], group
                    )

                    self.assertEqual("'true'" if is_owning_group else "''", is_required)


if __name__ == "__main__":
    unittest.main()
