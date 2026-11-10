import hashlib
import json
import os
import shutil
import subprocess
import tempfile
import tomllib
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


class DependencyVerificationTest(unittest.TestCase):
    def setUp(self) -> None:
        workspace = tempfile.TemporaryDirectory(prefix="dependency-check-")
        self.addCleanup(workspace.cleanup)
        self.root = Path(workspace.name)

        paths = (
            "mise.toml",
            "mise.lock",
            ".mise/locks",
            ".github/workflows",
            "Dockerfile",
            "docker-compose.yml",
            "frontend/package.json",
            "frontend/pnpm-lock.yaml",
            "frontend/browser",
            "website/package.json",
            "website/pnpm-lock.yaml",
            "e2e/browser/compose.yaml",
            "e2e/browser/pyproject.toml",
            "e2e/browser/uv.lock",
            "backend/go.mod",
            "backend/go.sum",
            "tools/go",
            "tools/pnpm",
            "tools/dependencies/mise-bootstrap.json",
            "backend/internal/lib/testimages/images.json",
            "backend/cmd/verification-agent/images/postgres",
            "backend/internal/lib/testcontainers/testdata",
            "backend/internal/modules/backups/e2e/postgresql/testdata",
            "backend/internal/modules/backups/physicalapitests/testdata",
            "e2e/docker-storage/nfs/Dockerfile",
        )
        for relative in paths:
            source, target = ROOT / relative, self.root / relative
            target.parent.mkdir(parents=True, exist_ok=True)
            if source.is_dir():
                shutil.copytree(source, target)
            else:
                shutil.copyfile(source, target)

        scripts = self.root / "scripts"
        scripts.mkdir()
        for name in (
            "verify_dependencies.py",
            "verify-deps.sh",
            "tool_pins.py",
            "runtime_pins.py",
            "action_pins.py",
            "image_pins.py",
            "verification_extension_pins.py",
            "client_bundles.py",
            "read_mise_bootstrap.py",
            "workflow_tool_pins.py",
        ):
            shutil.copy2(ROOT / "scripts" / name, scripts / name)

        archives = []
        for architecture in ("x64", "arm"):
            path = self.root / f"assets/tools/{architecture}/mongodb/bin/mongodump"
            path.parent.mkdir(parents=True)
            path.write_bytes(b"fixture client")
            path.chmod(0o755)
            archives.append(
                {
                    "bundle": "mongodb",
                    "version": "100.19.1",
                    "build_family": "ubuntu2204",
                    "architecture": architecture,
                    "url": f"https://example.com/{architecture}.tgz",
                    "sha256": "a" * 64,
                    "files": [
                        {
                            "path": str(path.relative_to(self.root)),
                            "member": "bin/mongodump",
                            "sha256": hashlib.sha256(path.read_bytes()).hexdigest(),
                            "mode": 0o755,
                        }
                    ],
                }
            )

        self.manifest = self.root / "tools/dependencies/clients.json"
        self.manifest.write_text(json.dumps({"schema": 1, "archives": archives}))

        subprocess.run(["git", "init", "--quiet", str(self.root)], check=True)
        subprocess.run(["git", "add", "."], cwd=self.root, check=True)

    def run_check(self, directory: str = ".") -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["mise", "run", "verify-deps"],
            cwd=self.root / directory,
            capture_output=True,
            text=True,
            check=False,
            timeout=30,
            env={
                **os.environ,
                "http_proxy": "http://127.0.0.1:1",
                "https_proxy": "http://127.0.0.1:1",
                "MISE_TRUSTED_CONFIG_PATHS": str(self.root),
            },
        )

    def test_checks_the_same_local_inputs_offline_from_root_and_frontend_without_writing(
        self,
    ) -> None:
        index = (self.root / ".git/index").read_bytes()
        files = {
            path: hashlib.sha256(path.read_bytes()).hexdigest()
            for path in self.root.rglob("*")
            if path.is_file()
        }

        execution = self.run_check()
        frontend_execution = self.run_check("frontend")

        self.assertEqual(execution.returncode, 0, execution.stdout + execution.stderr)
        self.assertEqual(frontend_execution.returncode, 0, frontend_execution.stdout)
        self.assertEqual(execution.stdout, frontend_execution.stdout)
        self.assertIn("updater's responsibility", execution.stdout)
        self.assertEqual((self.root / ".git/index").read_bytes(), index)
        self.assertEqual(
            {
                path: hashlib.sha256(path.read_bytes()).hexdigest()
                for path in self.root.rglob("*")
                if path.is_file()
            },
            files,
        )

    def test_reports_independent_pin_and_checksum_failures(self) -> None:
        configuration = self.root / "mise.toml"
        configuration.write_text(
            configuration.read_text().replace('node = "24.21.0"', 'node = "latest"')
        )
        (self.root / "assets/tools/x64/mongodb/bin/mongodump").write_bytes(b"replaced")

        execution = self.run_check()

        self.assertNotEqual(execution.returncode, 0)
        self.assertIn("requires an exact version", execution.stdout)
        self.assertIn("SHA256 mismatch", execution.stdout)

    def test_rejects_a_missing_native_sidecar(self) -> None:
        (self.root / ".mise/locks/npm-semver/7.8.6/aube-lock.yaml").unlink()

        execution = self.run_check()

        self.assertNotEqual(execution.returncode, 0)
        self.assertIn("missing native lock sidecar", execution.stdout)

    def test_rejects_a_workflow_tool_without_a_declaration(self) -> None:
        workflow = self.root / ".github/workflows/unknown-tool.yaml"
        workflow.write_text("install_args: 'unknown-tool'\n")

        execution = self.run_check()

        self.assertNotEqual(execution.returncode, 0)
        self.assertIn("unknown-tool.yaml:1", execution.stdout)
        self.assertIn("unknown-tool", execution.stdout)

    def test_accepts_static_workflow_tools_with_exact_global_and_task_versions(self) -> None:
        workflow = self.root / ".github/workflows/tools.yaml"
        for arguments in (
            "'node pnpm go@1.27.3 lefthook@2.2.1 npm:semver'",
            '"node pnpm" # locked tools',
            "node pnpm # locked tools",
        ):
            with self.subTest(arguments=arguments):
                workflow.write_text(f"install_args: {arguments}\n")

                execution = self.run_check()

                self.assertEqual(execution.returncode, 0, execution.stdout + execution.stderr)

    def test_rejects_workflow_versions_without_matching_declarations(self) -> None:
        workflow = self.root / ".github/workflows/tools.yaml"
        for tool in ("node@latest", "node@0.0.1", "lefthook"):
            with self.subTest(tool=tool):
                workflow.write_text(f"install_args: '{tool}'\n")

                execution = self.run_check()

                self.assertNotEqual(execution.returncode, 0)
                self.assertIn(f"tools.yaml:1: {tool}", execution.stdout)
                self.assertIn("missing exact mise tool declaration", execution.stdout)

    def test_rejects_a_workflow_version_without_a_native_lock(self) -> None:
        lockfile = self.root / "mise.lock"
        lockfile.write_text(lockfile.read_text().replace('version = "2.2.1"', 'version = "0.0.1"'))
        workflow = self.root / ".github/workflows/tools.yaml"
        workflow.write_text("install_args: 'lefthook@2.2.1'\n")

        execution = self.run_check()

        self.assertNotEqual(execution.returncode, 0)
        self.assertIn("tools.yaml:1: lefthook@2.2.1: missing native tool lock", execution.stdout)

    def test_rejects_dynamic_multiline_and_empty_workflow_tool_lists(self) -> None:
        workflow = self.root / ".github/workflows/tools.yaml"
        for arguments in (
            "'node ${{ matrix.tool }}'",
            ">-\n  node pnpm",
            "|\n  node pnpm",
            "node\n  pnpm",
            "''",
            "--force node",
        ):
            with self.subTest(arguments=arguments):
                workflow.write_text(f"install_args: {arguments}\n")

                execution = self.run_check()

                self.assertNotEqual(execution.returncode, 0)
                self.assertIn("tools.yaml:1: unsupported install_args", execution.stdout)

    def test_rejects_pnpm_without_the_github_backend_alias(self) -> None:
        configuration = self.root / "mise.toml"
        pinned_configuration = configuration.read_text()
        for alias in ("", 'pnpm = "aqua:pnpm/pnpm"'):
            with self.subTest(alias=alias):
                configuration.write_text(
                    pinned_configuration.replace('pnpm = "github:pnpm/pnpm"', alias)
                )

                execution = self.run_check()

                self.assertNotEqual(execution.returncode, 0)
                self.assertIn("pnpm requires a short tool declaration", execution.stdout)

    def test_rejects_a_backend_alias_that_disagrees_with_the_native_lock(self) -> None:
        configuration = self.root / "mise.toml"
        configuration.write_text(
            configuration.read_text().replace(
                "[tool_alias]", '[tool_alias]\nnode = "github:nodejs/node"'
            )
        )

        execution = self.run_check()

        self.assertNotEqual(execution.returncode, 0)
        self.assertIn(
            "node@24.21.0: configured alias and locked backend disagree", execution.stdout
        )

    def test_rejects_a_missing_download_url_on_either_architecture(self) -> None:
        lockfile = self.root / "mise.lock"
        pinned_lockfile = lockfile.read_text()
        node_lock = tomllib.loads(pinned_lockfile)["tools"]["node"][0]
        for platform in ("linux-x64", "linux-arm64"):
            with self.subTest(platform=platform):
                url = node_lock[f"platforms.{platform}"]["url"]
                lockfile.write_text(pinned_lockfile.replace(f'url = "{url}"', 'url = ""'))

                execution = self.run_check()

                self.assertNotEqual(execution.returncode, 0)
                self.assertIn(f"missing {platform} download URL", execution.stdout)

    def test_rejects_an_inconsistent_browser_runtime(self) -> None:
        environment = self.root / "frontend/browser/environment.json"
        document = json.loads(environment.read_text())
        document["nodeVersion"] = "0.0.1"
        environment.write_text(json.dumps(document))

        execution = self.run_check()

        self.assertNotEqual(execution.returncode, 0)
        self.assertIn("Node/pnpm versions disagree", execution.stdout)

    def test_rejects_an_inconsistent_website_package_manager(self) -> None:
        manifest = self.root / "website/package.json"
        website = json.loads(manifest.read_text())
        website["packageManager"] = "pnpm@0.0.1"
        manifest.write_text(json.dumps(website))

        execution = self.run_check()

        self.assertNotEqual(execution.returncode, 0)
        self.assertIn("website and mise versions disagree", execution.stdout)

    def test_rejects_unpinned_actions_and_images(self) -> None:
        workflow = self.root / ".github/workflows/unpinned.yml"
        workflow.write_text("jobs:\n  check:\n    steps:\n      - uses: actions/checkout@main\n")
        dockerfile = self.root / "Dockerfile"
        dockerfile.write_text(dockerfile.read_text() + "\nFROM alpine:latest\n")

        execution = self.run_check()

        self.assertNotEqual(execution.returncode, 0)
        self.assertIn("full commit pin", execution.stdout)
        self.assertIn("image requires a digest pin", execution.stdout)

    def test_rejects_unpinned_images_with_instruction_case_or_indentation(self) -> None:
        dockerfile = self.root / "Dockerfile"
        pinned = dockerfile.read_text()
        for instruction in ("from", "FrOm", "  FROM", "\tfrom"):
            with self.subTest(instruction=instruction):
                dockerfile.write_text(pinned + f"\n{instruction} alpine:latest\n")

                execution = self.run_check()

                self.assertNotEqual(execution.returncode, 0)
                self.assertIn("image requires a digest pin", execution.stdout)

    def test_accepts_pinned_images_and_stage_aliases_with_mixed_case_and_indentation(
        self,
    ) -> None:
        dockerfile = self.root / "Dockerfile"
        dockerfile.write_text(dockerfile.read_text().replace("FROM ", "  fRoM "))

        execution = self.run_check()

        self.assertEqual(execution.returncode, 0, execution.stdout + execution.stderr)

    def test_rejects_partial_and_mixed_client_refreshes(self) -> None:
        document = json.loads(self.manifest.read_text())
        document["archives"][1]["version"] = "100.20.0"
        self.manifest.write_text(json.dumps(document))

        execution = self.run_check()

        self.assertNotEqual(execution.returncode, 0)
        self.assertIn("mixed client builds", execution.stdout)

        document["archives"].pop()
        self.manifest.write_text(json.dumps(document))

        partial_execution = self.run_check()

        self.assertNotEqual(partial_execution.returncode, 0)
        self.assertIn("partial architecture refresh", partial_execution.stdout)

    def test_rejects_different_vendor_build_families_with_the_same_client_version(self) -> None:
        document = json.loads(self.manifest.read_text())
        document["archives"][1]["build_family"] = "ubuntu2404"
        self.manifest.write_text(json.dumps(document))

        execution = self.run_check()

        self.assertNotEqual(execution.returncode, 0)
        self.assertIn("mixed client builds", execution.stdout)

    def test_rejects_lost_executable_permission_in_working_copy_and_index(self) -> None:
        relative = "assets/tools/x64/mongodb/bin/mongodump"
        client = self.root / relative
        client.chmod(0o644)

        execution = self.run_check()

        self.assertNotEqual(execution.returncode, 0)
        self.assertIn("permissions differ", execution.stdout)

        client.chmod(0o755)
        subprocess.run(["git", "update-index", "--chmod=-x", relative], cwd=self.root, check=True)

        index_execution = self.run_check()

        self.assertNotEqual(index_execution.returncode, 0)
        self.assertIn("Git index requires executable permission", index_execution.stdout)

    def test_rejects_malformed_client_metadata_without_a_traceback(self) -> None:
        self.manifest.write_text('{"schema":1,"archives":[{"files":[]}]}')

        execution = self.run_check()

        self.assertNotEqual(execution.returncode, 0)
        self.assertIn("client archive requires recorded files", execution.stdout)
        self.assertNotIn("Traceback", execution.stderr)


if __name__ == "__main__":
    unittest.main()
