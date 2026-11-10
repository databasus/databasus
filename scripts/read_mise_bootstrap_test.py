import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


class ReadMiseBootstrapTest(unittest.TestCase):
    def test_bootstrap_reader_outputs_pinned_checksums_and_rejects_a_missing_architecture(
        self,
    ) -> None:
        with tempfile.TemporaryDirectory(prefix="mise-identity-") as workspace:
            root = Path(workspace)
            (root / "scripts").mkdir()
            records = root / "tools/dependencies/mise-bootstrap.json"
            records.parent.mkdir(parents=True)
            shutil.copyfile(
                ROOT / "scripts/read_mise_bootstrap.py", root / "scripts/read_mise_bootstrap.py"
            )
            shutil.copyfile(ROOT / "tools/dependencies/mise-bootstrap.json", records)
            document = json.loads(records.read_text())
            output = root / "output"
            for architecture, platform in (("X64", "linux-x64"), ("ARM64", "linux-arm64")):
                with self.subTest(architecture=architecture):
                    output.unlink(missing_ok=True)
                    execution = subprocess.run(
                        [sys.executable, "-B", str(root / "scripts/read_mise_bootstrap.py")],
                        env={
                            **os.environ,
                            "RUNNER_ARCH": architecture,
                            "GITHUB_OUTPUT": str(output),
                        },
                        capture_output=True,
                        text=True,
                        check=False,
                    )

                    self.assertEqual(execution.returncode, 0, execution.stderr)
                    self.assertEqual(
                        output.read_text(),
                        f"version={document['version']}\nsha256={document['platforms'][platform]}\n",
                    )
            output.unlink()
            del document["platforms"]["linux-arm64"]
            records.write_text(json.dumps(document))

            rejected = subprocess.run(
                [sys.executable, "-B", str(root / "scripts/read_mise_bootstrap.py")],
                env={**os.environ, "RUNNER_ARCH": "X64", "GITHUB_OUTPUT": str(output)},
                capture_output=True,
                text=True,
                check=False,
            )

            self.assertNotEqual(rejected.returncode, 0)
            self.assertIn("both Linux architectures", rejected.stderr)
            self.assertFalse(output.exists())


if __name__ == "__main__":
    unittest.main()
