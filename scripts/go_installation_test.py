import json
import os
import re
import subprocess
import tempfile
import textwrap
import unittest
import zipfile
from pathlib import Path

from workflow_jobs import get_jobs, get_steps

ROOT = Path(__file__).resolve().parent.parent


class GoInstallationTest(unittest.TestCase):
    def setUp(self) -> None:
        workspace = tempfile.TemporaryDirectory(prefix="go-installation-")
        self.addCleanup(workspace.cleanup)
        self.root = Path(workspace.name)
        self.backend = self.root / "backend"
        self.backend.mkdir()
        proxy = self.root / "proxy/example.test/dependency/@v"
        proxy.mkdir(parents=True)
        module = "module example.test/dependency\n\ngo 1.26.10\n"
        (proxy / "v1.0.0.mod").write_text(module)
        (proxy / "v1.0.0.info").write_text(
            json.dumps({"Version": "v1.0.0", "Time": "2020-01-01T00:00:00Z"})
        )
        (proxy / "list").write_text("v1.0.0\n")
        with zipfile.ZipFile(proxy / "v1.0.0.zip", "w") as archive:
            prefix = "example.test/dependency@v1.0.0/"
            archive.writestr(prefix + "go.mod", module)
            archive.writestr(prefix + "dependency.go", "package dependency\nfunc Version() {}\n")

        self.environment = {
            **os.environ,
            "GOPROXY": (self.root / "proxy").as_uri(),
            # The local fixture module is unknown to the public checksum service.
            "GOSUMDB": "off",
            "GOMODCACHE": str(self.root / "modules"),
            "GOPATH": str(self.root / "gopath"),
            "GOWORK": "off",
            "GOENV": "off",
            "GOTOOLCHAIN": "local",
        }
        (self.backend / "go.mod").write_text(
            "module installation-fixture\n\ngo 1.26.10\n\nrequire example.test/dependency v1.0.0\n"
        )
        (self.backend / "main.go").write_text(
            'package main\nimport "example.test/dependency"\nfunc main() { dependency.Version() }\n'
        )
        subprocess.run(
            ["go", "mod", "tidy"],
            cwd=self.backend,
            env=self.environment,
            check=True,
            capture_output=True,
            timeout=30,
        )

    def get_installation_scripts(self) -> dict[str, str]:
        scripts: dict[str, str] = {}
        jobs = get_jobs((ROOT / ".github/workflows/ci-release.yml").read_text())
        for job in ("lint-backend", "test-verification-agent", "test-backend"):
            for step in get_steps(jobs[job]):
                if re.search(r"^          go mod (download|tidy)", step, re.M):
                    scripts[job] = textwrap.dedent(step.split("        run: |\n", 1)[1])

        dockerfile = (ROOT / "Dockerfile").read_text()
        commands = re.findall(r"^RUN (?:--mount=\S+ )?(go mod [^\n]+)$", dockerfile, re.M)
        self.assertEqual(len(commands), 2)

        for stage, command in zip(("backend", "verification-agent"), commands, strict=True):
            scripts["docker-" + stage] = "cd backend\n" + command

        self.assertEqual(len(scripts), 5)

        return scripts

    def test_native_installations_reject_missing_checksums_and_stale_requirements(self) -> None:
        manifest = self.backend / "go.mod"
        checksums = self.backend / "go.sum"
        clean_manifest, clean_checksums = manifest.read_text(), checksums.read_text()
        for consumer, script in self.get_installation_scripts().items():
            manifest.write_text(clean_manifest)
            checksums.write_text(clean_checksums)

            accepted = subprocess.run(
                ["bash", "-euo", "pipefail", "-c", script],
                cwd=self.root,
                env=self.environment,
                capture_output=True,
                text=True,
                check=False,
                timeout=30,
            )

            self.assertEqual(accepted.returncode, 0, accepted.stdout + accepted.stderr)
            self.assertEqual(manifest.read_text(), clean_manifest)
            self.assertEqual(checksums.read_text(), clean_checksums)

            for defect in ("module checksum", "archive checksum", "missing requirement"):
                with self.subTest(consumer=consumer, defect=defect):
                    manifest.write_text(clean_manifest)
                    checksums.write_text(clean_checksums)
                    if defect == "missing requirement":
                        manifest.write_text(clean_manifest.split("require ")[0])
                    else:
                        checksums.write_text(
                            "".join(
                                line
                                for line in clean_checksums.splitlines(keepends=True)
                                if ("/go.mod " in line) != (defect == "module checksum")
                            )
                        )

                    committed_inputs = (manifest.read_bytes(), checksums.read_bytes())
                    rejected = subprocess.run(
                        ["bash", "-euo", "pipefail", "-c", script],
                        cwd=self.root,
                        env=self.environment,
                        capture_output=True,
                        text=True,
                        check=False,
                        timeout=30,
                    )

                    self.assertNotEqual(rejected.returncode, 0, rejected.stdout + rejected.stderr)
                    self.assertEqual(
                        (manifest.read_bytes(), checksums.read_bytes()), committed_inputs
                    )


if __name__ == "__main__":
    unittest.main()
