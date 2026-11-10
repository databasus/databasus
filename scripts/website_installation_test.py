import io
import json
import re
import shlex
import subprocess
import tarfile
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


class WebsiteInstallationTest(unittest.TestCase):
    def get_installation_command(self) -> list[str]:
        workflow = (ROOT / ".github/workflows/deploy-website.yml").read_text()
        installation = re.search(r"- name: Install dependencies\n\s+run: ([^\n]+)", workflow)
        self.assertIsNotNone(installation)
        assert installation is not None

        return shlex.split(installation[1]) + [
            "--offline",
            "--ignore-scripts",
        ]

    def create_lock(self, root: Path) -> None:
        subprocess.run(
            [
                "pnpm",
                "install",
                "--lockfile-only",
                "--offline",
                "--ignore-scripts",
                "--store-dir",
                str(root / "lock-store"),
            ],
            cwd=root,
            check=True,
            capture_output=True,
            timeout=60,
        )

    def test_website_installation_rejects_a_stale_lock_without_repairing_it(self) -> None:
        with tempfile.TemporaryDirectory(prefix="native-frozen-") as workspace:
            root = Path(workspace)
            for directory, version in (("first", "1.0.0"), ("second", "2.0.0")):
                package = root / directory
                package.mkdir()
                (package / "package.json").write_text(
                    json.dumps({"name": "local-package", "version": version})
                )

            manifest = {
                "name": "website-fixture",
                "version": "1.0.0",
                "dependencies": {"local-package": "file:./first"},
            }
            (root / "package.json").write_text(json.dumps(manifest))
            self.create_lock(root)
            command = self.get_installation_command()

            installed = subprocess.run(
                command, cwd=root, capture_output=True, text=True, check=False, timeout=60
            )

            self.assertEqual(installed.returncode, 0, installed.stderr)

            lock = (root / "pnpm-lock.yaml").read_bytes()
            manifest["dependencies"] = {"local-package": "file:./second"}
            (root / "package.json").write_text(json.dumps(manifest))

            rejected = subprocess.run(
                command, cwd=root, capture_output=True, text=True, check=False, timeout=60
            )

            self.assertNotEqual(rejected.returncode, 0, rejected.stdout + rejected.stderr)
            self.assertIn("ERR_PNPM_OUTDATED_LOCKFILE", rejected.stdout + rejected.stderr)
            self.assertEqual((root / "pnpm-lock.yaml").read_bytes(), lock)

    def write_package_archive(self, archive: Path, content: bytes) -> None:
        manifest = json.dumps({"name": "local-package", "version": "1.0.0"}).encode()

        with tarfile.open(archive, "w:gz") as package:
            for name, payload in (("package.json", manifest), ("index.js", content)):
                member = tarfile.TarInfo("package/" + name)
                member.size = len(payload)
                package.addfile(member, io.BytesIO(payload))

    def test_website_installation_rejects_changed_package_bytes_without_repairing_integrity(
        self,
    ) -> None:
        with tempfile.TemporaryDirectory(prefix="native-integrity-") as workspace:
            root = Path(workspace)
            archive = root / "local-package.tgz"
            self.write_package_archive(archive, b"module.exports = 'reviewed';")
            (root / "package.json").write_text(
                json.dumps(
                    {
                        "name": "website-fixture",
                        "version": "1.0.0",
                        "dependencies": {"local-package": "file:./local-package.tgz"},
                    }
                )
            )
            self.create_lock(root)
            lock = (root / "pnpm-lock.yaml").read_bytes()
            self.assertIn(b"integrity: sha512-", lock)

            command = self.get_installation_command()

            installed = subprocess.run(
                [*command, "--store-dir", str(root / "accepted-store")],
                cwd=root,
                capture_output=True,
                text=True,
                check=False,
                timeout=60,
            )

            self.assertEqual(installed.returncode, 0, installed.stderr)

            self.write_package_archive(archive, b"module.exports = 'replaced';")

            rejected = subprocess.run(
                [*command, "--store-dir", str(root / "rejected-store")],
                cwd=root,
                capture_output=True,
                text=True,
                check=False,
                timeout=60,
            )

            self.assertNotEqual(rejected.returncode, 0, rejected.stdout + rejected.stderr)
            self.assertIn("ERR_PNPM_TARBALL_INTEGRITY", rejected.stdout + rejected.stderr)
            self.assertEqual((root / "pnpm-lock.yaml").read_bytes(), lock)


if __name__ == "__main__":
    unittest.main()
