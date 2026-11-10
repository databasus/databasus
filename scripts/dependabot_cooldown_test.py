import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


class DependabotCooldownTest(unittest.TestCase):
    def test_cooldowns_preserve_release_delays_with_supported_fields(self) -> None:
        configuration = (ROOT / ".github/dependabot.yml").read_text()
        periods = {"npm": 7, "gomod": 7, "docker": 14, "github-actions": 7}

        for record in re.split(r"^  - package-ecosystem: ", configuration, flags=re.M)[1:]:
            ecosystem = record.splitlines()[0].strip('"')
            cooldown = record.split("    cooldown:\n", 1)[1]

            with self.subTest(ecosystem=ecosystem):
                self.assertRegex(cooldown, rf"(?m)^      default-days: {periods[ecosystem]}$")

                if ecosystem in ("npm", "gomod"):
                    for release_type in ("semver-minor-days", "semver-patch-days"):
                        self.assertRegex(
                            cooldown, rf"(?m)^      {release_type}: {periods[ecosystem]}$"
                        )
                    self.assertRegex(cooldown, r"(?m)^      semver-major-days: 30$")
                else:
                    self.assertNotRegex(cooldown, r"(?m)^      semver-")


if __name__ == "__main__":
    unittest.main()
