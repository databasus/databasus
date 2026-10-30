"""Local checks for the frontend workflow's toolchain and release prerequisites."""

import json
from pathlib import Path
import re
import tomllib
import unittest

ROOT = Path(__file__).resolve().parents[1]


def get_job(workflow, name):
    match = re.search(rf"^  {re.escape(name)}:\n(.*?)(?=^  [a-zA-Z0-9_-]+:|\Z)", workflow, re.M | re.S)
    if match is None:
        raise AssertionError(f"Missing workflow job: {name}")
    return match.group(1)


def check_frontend_toolchain(workflow):
    actions = []
    for name in ("lint-frontend", "test-frontend"):
        job = get_job(workflow, name)
        action = re.search(r"uses: (jdx/mise-action@[a-f0-9]{40})", job)
        tools = re.search(r"install_args:\s*['\"]?([^'\"\n]+)", job)
        if action is None or tools is None or set(tools.group(1).split()) != {"node", "pnpm"}:
            raise AssertionError(f"{name} must install the pinned node and pnpm tools with mise")
        if "actions/setup-node@" in job or "pnpm/action-setup@" in job or "node-version:" in job:
            raise AssertionError(f"{name} must not select a separate frontend toolchain")
        if "pnpm install --frozen-lockfile" not in job:
            raise AssertionError(f"{name} must preserve the lockfile")
        actions.append(action.group(1))
    if len(set(actions)) != 1:
        raise AssertionError("Frontend jobs must use the same pinned mise action")
    if "mise run frontend:lint" not in get_job(workflow, "lint-frontend") or "pnpm build" not in get_job(workflow, "lint-frontend"):
        raise AssertionError("Frontend lint and build must stay in the toolchain job")
    if "pnpm test" not in get_job(workflow, "test-frontend"):
        raise AssertionError("Frontend tests must run in their toolchain job")


class FrontendCiTest(unittest.TestCase):
    def setUp(self):
        self.workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()

    def test_frontend_jobs_use_one_pinned_toolchain_and_frozen_installs(self):
        check_frontend_toolchain(self.workflow)
        configuration = tomllib.loads((ROOT / "mise.toml").read_text())
        self.assertIn("node", configuration["tools"])
        self.assertIn("pnpm", configuration["tools"])
        package = json.loads((ROOT / "frontend/package.json").read_text())
        self.assertEqual(package["packageManager"], f'pnpm@{configuration["tools"]["pnpm"]}')

    def test_architecture_verification_is_part_of_the_release_lint_task(self):
        configuration = tomllib.loads((ROOT / "mise.toml").read_text())
        commands = configuration["tasks"]["frontend:lint"]["run"]
        for command in ("pnpm format:check", "pnpm lint", "pnpm lint:fsd", "pnpm typecheck", "pnpm --silent check:api"):
            with self.subTest(command=command):
                self.assertIn(command, commands)
        self.assertIn("needs: [lint-frontend]", get_job(self.workflow, "test-frontend"))
        release = get_job(self.workflow, "determine-version")
        prerequisites = re.findall(r"[a-zA-Z0-9_-]+", re.search(r"needs:\s*\[([^]]+)\]", release).group(1))
        self.assertEqual(set(prerequisites), {
            "lint-backend", "test-backend", "test-frontend", "test-verification-agent",
            "e2e-verification-agent-podman", "test-filesystems", "test-upgrade",
        })

    def test_detects_runtime_drift_and_mutable_dependency_installs(self):
        for changed in (
            re.sub(r"install_args: ['\"]node pnpm['\"]", 'node-version: \"20\"', self.workflow, count=1),
            self.workflow.replace("pnpm install --frozen-lockfile", "pnpm install", 1),
            self.workflow.replace("pnpm build", "pnpm unrelated-script", 1),
        ):
            with self.subTest(workflow=changed):
                with self.assertRaises(AssertionError):
                    check_frontend_toolchain(changed)


if __name__ == "__main__":
    unittest.main()
