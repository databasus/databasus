import json
import re
import tomllib
import unittest
from pathlib import Path

from pattern_match import get_required_match
from workflow_jobs import get_jobs

ROOT = Path(__file__).resolve().parents[1]
DOWNLOAD_ATTEMPTS_ALLOWED_TO_FAIL = [
    "Download artifact",
    "Download artifact (attempt 2)",
]
E2E_TASKS = ("e2e:browser", "e2e:unit", "e2e:lifecycle")


def get_step_names_allowed_to_fail(job: str) -> list[str]:
    head_of_job, *steps = re.split(r"(?m)^      - name: ", job)
    if "continue-on-error" in head_of_job:
        raise AssertionError("the whole job must not be allowed to fail")
    return [step.splitlines()[0] for step in steps if "continue-on-error" in step]


def check_frontend_toolchain(workflow: str) -> None:
    jobs = get_jobs(workflow)
    actions = []
    for name in ("lint-frontend", "test-frontend", "test-frontend-browser-ui"):
        job = jobs[name]
        action = re.search(r"uses: (jdx/mise-action@[a-f0-9]{40})", job)
        tools = re.search(r"install_args:\s*['\"]?([^'\"\n]+)", job)

        if action is None or tools is None or set(tools.group(1).split()) != {"node", "pnpm"}:
            raise AssertionError(f"{name} must install the pinned node and pnpm tools with mise")
        if "actions/setup-node@" in job or "pnpm/action-setup@" in job or "node-version:" in job:
            raise AssertionError(f"{name} must not select a separate frontend toolchain")
        if not re.search(r"pnpm(?: --dir frontend)? install --frozen-lockfile", job):
            raise AssertionError(f"{name} must preserve the lockfile")

        actions.append(action.group(1))

    if len(set(actions)) != 1:
        raise AssertionError("Frontend jobs must use the same pinned mise action")
    if (
        "mise run frontend:lint" not in jobs["lint-frontend"]
        or "pnpm build" not in jobs["lint-frontend"]
    ):
        raise AssertionError("Frontend lint and build must stay in the toolchain job")
    if "pnpm test" not in jobs["test-frontend"]:
        raise AssertionError("Frontend tests must run in their toolchain job")


class FrontendCiTest(unittest.TestCase):
    def setUp(self) -> None:
        self.workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()

    def test_frontend_jobs_use_one_pinned_toolchain_and_frozen_installs(self) -> None:
        check_frontend_toolchain(self.workflow)

        configuration = tomllib.loads((ROOT / "mise.toml").read_text())
        self.assertIn("node", configuration["tools"])
        self.assertIn("pnpm", configuration["tools"])
        self.assertEqual(configuration["tool_alias"]["pnpm"], "github:pnpm/pnpm")

        package = json.loads((ROOT / "frontend/package.json").read_text())
        self.assertEqual(package["packageManager"], f"pnpm@{configuration['tools']['pnpm']}")

    def test_script_suite_installs_its_frontend_tools_before_discovery(self) -> None:
        job = get_jobs(self.workflow)["test-scripts"]
        tools = re.search(r"install_args:\s*['\"]?([^'\"\n]+)", job)

        assert tools is not None
        self.assertTrue({"node", "pnpm"}.issubset(tools.group(1).split()))
        self.assertRegex(job, r"uses: jdx/mise-action@[a-f0-9]{40}")
        self.assertLess(job.index("install_args:"), job.index("python3 -m unittest discover"))

    def test_architecture_verification_is_part_of_the_release_lint_task(self) -> None:
        configuration = tomllib.loads((ROOT / "mise.toml").read_text())
        self.assertEqual(configuration["tasks"]["frontend:lint"]["run"], "pnpm lint")
        package = json.loads((ROOT / "frontend/package.json").read_text())
        self.assertEqual(package["scripts"]["lint"], "bash lint/check.sh")
        commands = (ROOT / "frontend/lint/check.sh").read_text()

        for command in (
            "pnpm exec oxfmt --check",
            "pnpm exec oxlint",
            "node lint/fsd.ts",
            "node lint/styles.ts",
            "pnpm typecheck",
            "pnpm --silent generate:api --check",
        ):
            with self.subTest(command=command):
                self.assertIn(command, commands)
        self.assertNotRegex(get_jobs(self.workflow)["test-frontend"], r"(?m)^    needs:")

        release = get_jobs(self.workflow)["determine-version"]
        prerequisites = re.findall(
            r"[a-zA-Z0-9_-]+", get_required_match(r"needs:\s*\[([^]]+)\]", release).group(1)
        )
        self.assertEqual(
            set(prerequisites),
            {
                "lint-backend",
                "test-backend",
                "test-frontend",
                "test-frontend-browser-ui",
                "test-e2e-browser",
                "test-e2e-browser-lifecycle",
                "test-verification-agent",
                "e2e-verification-agent-podman",
                "test-filesystems",
                "test-upgrade",
                "lint-workflows",
                "build-image",
                "build-verification-image",
                "code-scanning",
                "lint-frontend",
                "lint-python",
                "lint-shell",
                "test-scripts",
                "dockerfile-scan",
                "dependency-review",
            },
        )

    def test_browser_job_uses_the_local_command_and_keeps_failure_artifacts(self) -> None:
        job = get_jobs(self.workflow)["test-frontend-browser-ui"]
        self.assertIn(
            "pnpm --dir frontend test:browser --project=ui --shard=${{ matrix.shard }}/3", job
        )
        self.assertIn("name: frontend-browser-ui-${{ matrix.arch }}-${{ matrix.shard }}-of-3", job)
        self.assertIn("if: failure()", job)
        self.assertIn("path: frontend/browser-results/", job)

        for action in re.findall(r"uses: ([^\s]+)", job):
            if action.startswith("./"):
                self.assertEqual(action, "./.github/actions/read-mise-bootstrap")
                self.assertTrue((ROOT / action / "action.yml").is_file())
            else:
                self.assertRegex(action, r"@[a-f0-9]{40}$")

        environment = json.loads((ROOT / "frontend/browser/environment.json").read_text())
        self.assertRegex(environment["playwrightImage"], r"@sha256:[a-f0-9]{64}$")
        configuration = tomllib.loads((ROOT / "mise.toml").read_text())
        self.assertEqual(environment["nodeVersion"], configuration["tools"]["node"])
        self.assertEqual(environment["pnpmVersion"], configuration["tools"]["pnpm"])

        package = json.loads((ROOT / "frontend/package.json").read_text())
        self.assertEqual(
            package["devDependencies"]["@playwright/test"], environment["playwrightVersion"]
        )
        self.assertIn("lint", package["scripts"])

    def test_browser_end_to_end_suite_runs_on_the_candidate_image_and_is_required_before_release(
        self,
    ) -> None:
        job = get_jobs(self.workflow)["test-e2e-browser"]
        self.assertIn("needs: [build-storage-image]", job)
        self.assertNotRegex(job, r"(?m)^    if:")
        self.assertIn("install_args: 'python uv'", job)
        self.assertIn("databasus-storage-test-${{ matrix.arch }}.tar.zst | docker load", job)
        self.assertEqual(
            job.count(
                "DATABASUS_IMAGE: databasus-storage-test:${{ github.sha }}-${{ matrix.arch }}"
            ),
            1,
        )
        self.assertLess(
            job.index("run: mise run e2e:unit\n"), job.index("run: mise run e2e:browser\n")
        )
        self.assertNotIn("run: mise run e2e:lifecycle", job)
        lifecycle = get_jobs(self.workflow)["test-e2e-browser-lifecycle"]
        self.assertIn("needs: [build-storage-image]", lifecycle)
        self.assertNotRegex(lifecycle, r"(?m)^    if:")
        self.assertIn("install_args: 'python uv'", lifecycle)
        self.assertIn("run: mise run e2e:lifecycle ${{ matrix.group }}", lifecycle)
        self.assertNotIn("run: mise run e2e:browser", lifecycle)
        self.assertNotIn("run: mise run e2e:unit", lifecycle)
        self.assertEqual(
            lifecycle.count(
                "DATABASUS_IMAGE: databasus-storage-test:${{ github.sha }}-${{ matrix.arch }}"
            ),
            1,
        )
        self.assertEqual(
            get_step_names_allowed_to_fail(lifecycle), DOWNLOAD_ATTEMPTS_ALLOWED_TO_FAIL
        )
        self.assertNotIn("upload-artifact", job)
        self.assertNotIn("backend-db-test", job)
        self.assertNotIn("--project=e2e", job)
        self.assertEqual(get_step_names_allowed_to_fail(job), DOWNLOAD_ATTEMPTS_ALLOWED_TO_FAIL)
        self.assertNotIn("lint-backend", job)
        self.assertNotIn("test-backend", job)

        for action in re.findall(r"uses: ([^\s]+)", job):
            if action.startswith("./"):
                self.assertEqual(action, "./.github/actions/read-mise-bootstrap")
                self.assertTrue((ROOT / action / "action.yml").is_file())
            else:
                self.assertRegex(action, r"@[a-f0-9]{40}$")

        release = get_jobs(self.workflow)["determine-version"]
        self.assertIn("test-e2e-browser,", release)
        self.assertIn("test-e2e-browser-lifecycle,", release)
        self.assertNotIn("always()", release)
        self.assertNotIn("failure()", release)
        self.assertNotIn("continue-on-error", release)

        self.assertNotIn("test-frontend-browser-e2e", self.workflow)
        self.assertNotIn("mise run test-full", self.workflow)
        self.assertEqual(self.workflow.count("mise run e2e:browser"), 1)

    def test_browser_end_to_end_suite_installs_only_through_the_pinned_uv_and_its_lockfile(
        self,
    ) -> None:
        configuration = tomllib.loads((ROOT / "mise.toml").read_text())

        self.assertRegex(configuration["tools"]["uv"], r"^\d+\.\d+\.\d+$")

        for task_name in E2E_TASKS:
            with self.subTest(task_name=task_name):
                self.assertTrue(
                    configuration["tasks"][task_name]["run"].startswith(
                        "uv run --locked --project e2e/browser "
                    )
                )

    def test_full_run_is_the_visible_command_of_the_browser_end_to_end_suite(self) -> None:
        tasks = tomllib.loads((ROOT / "mise.toml").read_text())["tasks"]

        self.assertNotIn("e2e", tasks)
        self.assertTrue(tasks["e2e:browser"]["hide"])
        self.assertEqual(tasks["test-full"]["run"], "scripts/test.sh --full")
        self.assertEqual(tasks["test"]["run"], "scripts/test.sh")

    def test_detects_runtime_drift_and_mutable_dependency_installs(self) -> None:
        for mutation_index, changed in enumerate(
            (
                re.sub(
                    r"install_args: ['\"]node pnpm['\"]",
                    'node-version: "20"',
                    self.workflow,
                    count=1,
                ),
                self.workflow.replace("pnpm install --frozen-lockfile", "pnpm install", 1),
                self.workflow.replace("pnpm build", "pnpm unrelated-script", 1),
            )
        ):
            with self.subTest(mutation=mutation_index):
                with self.assertRaises(AssertionError):
                    check_frontend_toolchain(changed)


if __name__ == "__main__":
    unittest.main()
