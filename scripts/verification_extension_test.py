import hashlib
import os
import shlex
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


class VerificationExtensionTest(unittest.TestCase):
    def test_verification_image_checksum_step_rejects_changed_bytes_and_missing_records(
        self,
    ) -> None:
        dockerfile = (
            ROOT / "backend/cmd/verification-agent/images/postgres/Dockerfile"
        ).read_text()
        checksum_step = (
            "vchord_file="
            + dockerfile.split("vchord_file=", 1)[1].split(
                "apt-get install -y --no-install-recommends /tmp/vchord.deb;", 1
            )[0]
        )
        with tempfile.TemporaryDirectory(prefix="extension-checksum-") as workspace:
            root = Path(workspace)
            archive = root / "extension.deb"
            archive.write_bytes(b"reviewed extension")
            checksum = hashlib.sha256(archive.read_bytes()).hexdigest()
            records = root / "checksums"
            records.write_text(f"{checksum}  postgresql-16-vchord_1.1.1-1_amd64.deb\n")
            marker = root / "installed"
            script = (
                checksum_step.replace("\\\n", "\n")
                .replace("/opt/vectorchord.sha256", str(records))
                .replace("/tmp/vchord.deb", str(archive))
                + "\ntouch "
                + shlex.quote(str(marker))
            )
            environment = {
                **os.environ,
                "PG_MAJOR": "16",
                "TARGETARCH": "amd64",
                "VCHORD_VERSION": "1.1.1",
            }

            accepted = subprocess.run(
                ["bash", "-euo", "pipefail", "-c", script],
                env=environment,
                capture_output=True,
                text=True,
                check=False,
            )

            self.assertEqual(accepted.returncode, 0, accepted.stderr)
            self.assertTrue(marker.exists())

            marker.unlink()
            archive.write_bytes(b"replaced extension")

            rejected = subprocess.run(
                ["bash", "-euo", "pipefail", "-c", script],
                env=environment,
                capture_output=True,
                text=True,
                check=False,
            )

            self.assertNotEqual(rejected.returncode, 0)
            self.assertFalse(marker.exists())

            records.write_text("")

            missing = subprocess.run(
                ["bash", "-euo", "pipefail", "-c", script],
                env=environment,
                capture_output=True,
                text=True,
                check=False,
            )

            self.assertNotEqual(missing.returncode, 0)
            self.assertFalse(marker.exists())


if __name__ == "__main__":
    unittest.main()
