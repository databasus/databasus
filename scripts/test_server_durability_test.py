import re
import unittest
from pathlib import Path

from pattern_match import get_required_match

ROOT = Path(__file__).resolve().parent.parent


def get_compose_service_block(name: str) -> str:
    compose = (ROOT / "docker-compose.yml").read_text()

    return get_required_match(rf"^  {name}:\n(.*?)(?=^  \S|^\S|\Z)", compose, re.M | re.S).group(1)


class SharedTestServerDurabilityTest(unittest.TestCase):
    def test_test_server_turns_durability_off(self) -> None:
        service = get_compose_service_block("backend-db-test")

        for flag in ("fsync=off", "synchronous_commit=off", "full_page_writes=off"):
            self.assertIn(f"-c {flag}", service)

    def test_development_database_keeps_durability(self) -> None:
        service = get_compose_service_block("backend-db")

        for setting in ("fsync", "synchronous_commit", "full_page_writes"):
            self.assertNotIn(setting, service)


if __name__ == "__main__":
    unittest.main()
