import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


class DependabotCooldownTest(unittest.TestCase):
    def test_ordinary_cooldowns_keep_the_documented_period_and_longer_major_delay(self) -> None:
        configuration = (ROOT / ".github/dependabot.yml").read_text()
        periods = {"npm": 7, "gomod": 7, "docker": 14, "github-actions": 7}

        for record in re.split(r"^  - package-ecosystem: ", configuration, flags=re.M)[1:]:
            ecosystem = record.splitlines()[0].strip('"')
            cooldown = record.split("    cooldown:\n", 1)[1]

            with self.subTest(ecosystem=ecosystem):
                for release_type in ("default-days", "semver-minor-days", "semver-patch-days"):
                    self.assertRegex(cooldown, rf"(?m)^      {release_type}: {periods[ecosystem]}$")
                self.assertRegex(cooldown, r"(?m)^      semver-major-days: 30$")


if __name__ == "__main__":
    unittest.main()
