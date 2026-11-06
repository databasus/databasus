import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class SecretScanAllowlistTest(unittest.TestCase):
    def test_allowlist_holds_fixture_fingerprints(self) -> None:
        allowlist = (ROOT / ".gitleaksignore").read_text()

        for file_and_line in (
            "backend/internal/lib/crypto/legacy_master_key_test.go:generic-api-key:18",
            "backend/internal/app/app_test.go:generic-api-key:213",
        ):
            with self.subTest(file_and_line=file_and_line):
                self.assertIn(
                    f"37e8dcd3b0a5a09d7eb4cec8f2225269823bfa0d:{file_and_line}", allowlist
                )


if __name__ == "__main__":
    unittest.main()
