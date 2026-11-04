from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parent.parent


# No test run can notice a missing race detector by failing, so these pin where the flag stays.
class RaceDetectorInCiTest(unittest.TestCase):
    def test_ci_test_groups_run_with_the_race_detector(self):
        workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()

        self.assertIn("go test -race -p=8 -count=1 -failfast -timeout 40m", workflow)

    def test_commands_shared_with_ci_add_the_race_detector_only_under_ci(self):
        shared_commands = {
            "agent:test": (ROOT / "mise-tasks/agent/test").read_text(),
            "agent:e2e-podman": (ROOT / "mise-tasks/agent/e2e-podman").read_text(),
            "agent:e2e": (ROOT / "mise-tasks/agent/e2e").read_text(),
            "scripts/test.sh": (ROOT / "scripts/test.sh").read_text(),
            "backend:test-upgrade": (ROOT / "mise-tasks/backend/test-upgrade").read_text(),
        }

        for name, command in shared_commands.items():
            with self.subTest(command=name):
                self.assertIn("go test ${CI:+-race} ", command)
                self.assertNotIn("go test -race", command)
                self.assertNotRegex(command, r"go test -(?!-)")


if __name__ == "__main__":
    unittest.main()
