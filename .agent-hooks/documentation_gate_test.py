import json
import unittest

import review_gate_test as fixtures


class DocumentationGateTest(unittest.TestCase):
    def test_runs_for_each_document_owner_with_selected_profile(self) -> None:
        for client in ("claude", "codex"):
            for path in (
                "docs/testing.md",
                "README.md",
                "backend/README.md",
                "assets/readme/README.ru.md",
                "backend/AGENTS.md",
                ".agents/skills/testing/SKILL.md",
                ".agent-hooks/diff-prompt.md",
            ):
                with self.subTest(client=client, path=path):
                    repository = fixtures.HookRepository(self)
                    repository.environment["AGENT_REVIEW_CLIENT"] = client
                    repository.submit_prompt("Keep documentation focused")
                    repository.write(path, "# Documentation\n")

                    response = repository.stop("documentation")

                    self.assertEqual(response, {})
                    [call] = repository.reviewer_calls()
                    self.assertEqual(call["name"], client)
                    self.assertEqual(
                        fixtures.get_effort(call), "medium" if client == "codex" else "low"
                    )
                    arguments = call["arguments"]
                    self.assertEqual(
                        arguments[arguments.index("--model") + 1],
                        "gpt-6.1-sol" if client == "codex" else "opus",
                    )
                    self.assertIn("## Documentation review", call["prompt"])

    def test_skips_code_openspec_and_generated_skills(self) -> None:
        for path in (
            "backend/internal/widgets/service.go",
            "notes.md",
            "openspec/specs/widgets/spec.md",
            "openspec/changes/archive/finished/README.md",
            ".agents/skills/openspec-explore/SKILL.md",
            "backend/api/generated/README.md",
        ):
            with self.subTest(path=path):
                repository = fixtures.HookRepository(self)
                repository.submit_prompt("Update a file")
                repository.write(path, "content\n")

                self.assertEqual(repository.stop("documentation"), {})
                self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_documentation_unchanged_since_request(self) -> None:
        repository = fixtures.HookRepository(self)
        repository.write("docs/testing.md", "# Another request\n")
        repository.submit_prompt("Change code")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        self.assertEqual(repository.stop("documentation"), {})
        self.assertEqual(repository.reviewer_calls(), [])

    def test_runs_for_documentation_edited_after_another_request(self) -> None:
        repository = fixtures.HookRepository(self)
        repository.write("docs/testing.md", "# Another request\n")
        repository.submit_prompt("Fix documentation")
        repository.write("docs/testing.md", "# This request\n")

        self.assertEqual(repository.stop("documentation"), {})
        self.assertEqual(len(repository.reviewer_calls()), 1)

    def test_lists_only_documentation_changed_by_this_request(self) -> None:
        repository = fixtures.HookRepository(self)
        repository.write("docs/architecture.md", "# Another request\n")
        repository.submit_prompt("Fix test documentation")
        repository.write("docs/testing.md", "# This request\n")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        repository.stop("documentation")

        [call] = repository.reviewer_calls()
        changed_paths = call["prompt"].split("Currently changed paths:\n", 1)[1]
        self.assertEqual(changed_paths.strip(), "- docs/testing.md")

    def test_runs_for_staged_documentation(self) -> None:
        repository = fixtures.HookRepository(self)
        repository.submit_prompt("Fix documentation")
        repository.write("docs/testing.md", "# Running tests\n")
        repository.git("add", "docs/testing.md")

        self.assertEqual(repository.stop("documentation"), {})
        self.assertEqual(len(repository.reviewer_calls()), 1)

    def test_runs_for_deleted_documentation(self) -> None:
        repository = fixtures.HookRepository(self)
        repository.submit_prompt("Remove redundant documentation")
        (repository.root / "README.md").unlink()

        self.assertEqual(repository.stop("documentation"), {})
        self.assertEqual(len(repository.reviewer_calls()), 1)

    def test_blocks_repeats_and_accepts_repaired_documentation(self) -> None:
        repository = fixtures.HookRepository(self)
        repository.submit_prompt("Fix documentation")
        repository.write("docs/testing.md", "# Implementation inventory\n")
        repository.set_reviewer_result(
            {
                "ok": False,
                "reason": "docs/testing.md: remove the helper inventory; it is clear from the code.",
            }
        )

        rejection = repository.stop("documentation")

        self.assertEqual(rejection["decision"], "block")
        self.assertIn("remove the helper inventory", rejection["reason"])
        repository.set_reviewer_result({"ok": True, "reason": ""})
        repository.write("docs/testing.md", "# Running tests\n")
        self.assertEqual(repository.stop("documentation"), {})
        self.assertEqual(repository.stop("documentation"), {})
        self.assertEqual(len(repository.reviewer_calls()), 2)

    def test_blocks_when_reviewer_cannot_run(self) -> None:
        for client in ("claude", "codex"):
            with self.subTest(client=client):
                repository = fixtures.HookRepository(self)
                repository.review_client = client
                repository.environment["AGENT_REVIEW_CLIENT"] = client
                repository.submit_prompt("Fix documentation")
                repository.write("docs/testing.md", "# Running tests\n")
                repository.set_reviewer_failure("model unavailable")

                response = repository.stop("documentation")

                self.assertEqual(response["decision"], "block")
                self.assertIn("model unavailable", response["reason"])

    def test_manual_review_runs_documentation_only_when_relevant(self) -> None:
        for has_documentation in (False, True):
            with self.subTest(has_documentation=has_documentation):
                repository = fixtures.HookRepository(self)
                repository.complete_tasks()
                if has_documentation:
                    repository.write("docs/testing.md", "# Running tests\n")

                process = fixtures.ManualReviewTest().run_review(repository)

                self.assertEqual(process.returncode, 0, process.stdout + process.stderr)
                documentation_calls = [
                    call
                    for call in repository.reviewer_calls()
                    if "## Documentation review" in call["prompt"]
                ]
                self.assertEqual(len(documentation_calls), int(has_documentation))
                self.assertIn(
                    "Quick check `documentation`: "
                    + ("accepted" if has_documentation else "skipped"),
                    process.stdout,
                )

    def test_both_clients_wire_documentation_into_stop(self) -> None:
        for relative_path in (".codex/hooks.json", ".claude/settings.json"):
            configuration = json.loads(
                (fixtures.HOOK_PATH.parent.parent / relative_path).read_text()
            )
            commands = [
                hook["command"]
                for group in configuration["hooks"]["Stop"]
                for hook in group["hooks"]
            ]

            self.assertEqual(sum(" documentation --client " in command for command in commands), 1)
