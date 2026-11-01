import os
from pathlib import Path
import subprocess
import tempfile
import unittest


REPOSITORY_ROOT = Path(__file__).resolve().parent.parent


class LibeditShimTestCase(unittest.TestCase):
    def test_keeps_loader_path_when_native_libedit_soname_is_available(self):
        with tempfile.TemporaryDirectory(prefix="libedit-shim-") as directory:
            workspace = Path(directory)
            commands = workspace / "bin"
            commands.mkdir()
            ldconfig = commands / "ldconfig"
            ldconfig.write_text(
                "#!/bin/sh\nprintf '%s\\n' 'libedit.so.2 (libc6,x86-64) => /lib/x86_64-linux-gnu/libedit.so.2'\n"
            )
            ldconfig.chmod(0o755)
            shim_directory = workspace / "shim"

            checked = subprocess.run(
                [
                    "bash", "-euo", "pipefail", "-c",
                    'source "$1"; apply_libedit_shim; printf "%s" "$LD_LIBRARY_PATH"',
                    "bash", str(REPOSITORY_ROOT / "scripts" / "fedora.sh"),
                ],
                env={
                    **os.environ,
                    "PATH": f"{commands}:{os.environ['PATH']}",
                    "LIBEDIT_SHIM_DIR": str(shim_directory),
                    "LD_LIBRARY_PATH": "/existing/native/libraries",
                },
                capture_output=True,
                text=True,
                check=False,
            )

            self.assertEqual(checked.returncode, 0, checked.stderr)
            self.assertEqual(checked.stdout, "/existing/native/libraries")
            self.assertFalse(shim_directory.exists())
