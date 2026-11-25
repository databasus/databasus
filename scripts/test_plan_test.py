import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

from test_plan import (
    BackendSelector,
    FrontendSelector,
    Selector,
    SuiteSelector,
    TestPlanError,
    parse_selection,
)

SCRIPT = Path(__file__).resolve().with_name("test_plan.py")


def make_plan(*selection_lines: str, fence: str = "tests") -> str:
    block = "\n".join(selection_lines)
    return (
        "# Test plan\n\n## Impact\n\n| Changed | Affects | Why |\n|---|---|---|\n\n"
        f"## Not selected\n\n- nothing\n\n## Selection\n\n```{fence}\n{block}\n```\n"
    )


class ParseSelectionTest(unittest.TestCase):
    def assert_selected(self, line: str, selector: Selector) -> None:
        self.assertEqual(parse_selection(make_plan(line)), [selector])

    def assert_refused(self, plan_text: str, reason: str) -> None:
        with self.assertRaises(TestPlanError) as raised:
            parse_selection(plan_text)
        self.assertIn(reason, str(raised.exception))

    def test_accepts_a_backend_package_with_a_version_filter(self) -> None:
        self.assert_selected(
            "backend ./internal/engines/postgresql/logical -run '^TestSource$/^postgres:1[78]$'",
            BackendSelector(
                "./internal/engines/postgresql/logical", "^TestSource$/^postgres:1[78]$"
            ),
        )

    def test_accepts_narrow_recursive_backend_patterns(self) -> None:
        for pattern in (
            "./cmd/...",
            "./internal/modules/notifiers/...",
            "./internal/agent/e2e/...",
        ):
            with self.subTest(pattern=pattern):
                self.assert_selected(f"backend {pattern}", BackendSelector(pattern, ""))

    def test_accepts_a_frontend_path_alone_and_with_a_test_name(self) -> None:
        self.assert_selected(
            "frontend src/features/edit-notifier",
            FrontendSelector("src/features/edit-notifier", ""),
        )
        self.assert_selected(
            "frontend src/features/edit-notifier/ -t 'shows the error'",
            FrontendSelector("src/features/edit-notifier", "shows the error"),
        )

    def test_accepts_every_known_suite(self) -> None:
        for name in (
            "browser-ui",
            "e2e-browser",
            "e2e-unit",
            "lifecycle",
            "lifecycle-interruption-checks",
            "upgrade",
            "scripts",
            "review-hooks",
        ):
            with self.subTest(name=name):
                self.assert_selected(f"suite {name}", SuiteSelector(name))

    def test_accepts_none_as_the_only_line(self) -> None:
        self.assertEqual(parse_selection(make_plan("none")), [])

    def test_skips_empty_and_comment_lines(self) -> None:
        self.assertEqual(
            parse_selection(make_plan("# the hook tests", "", "suite review-hooks")),
            [SuiteSelector("review-hooks")],
        )

    def test_refuses_none_beside_another_selector(self) -> None:
        self.assert_refused(make_plan("none", "suite scripts"), "`none` must be the only line")

    def test_refuses_an_empty_selection(self) -> None:
        self.assert_refused(make_plan("# nothing yet"), "holds no selector")

    def test_refuses_a_selection_fenced_otherwise(self) -> None:
        self.assert_refused(make_plan("suite scripts", fence="sh"), "the plan has no selection")

    def test_refuses_a_plan_without_a_selection_section(self) -> None:
        self.assert_refused("# Test plan\n\n## Impact\n", "no ## Selection section")

    def test_refuses_an_unclosed_block(self) -> None:
        self.assert_refused("## Selection\n\n```tests\nsuite scripts\n", "is not closed")

    def test_refuses_a_shell_operator(self) -> None:
        self.assert_refused(
            make_plan("backend ./internal/lib/... && rm -rf /tmp/x"), "does not match its form"
        )

    def test_refuses_paths_outside_their_module(self) -> None:
        for line in ("backend ../frontend", "backend internal/lib", "frontend ../backend"):
            with self.subTest(line=line):
                self.assert_refused(make_plan(line), "leaves the")

    def test_refuses_the_whole_backend_module_and_its_broad_trees(self) -> None:
        for pattern in (
            "./...",
            "./internal/...",
            "./internal/modules/...",
            "./internal/engines/...",
            "./internal/lib/...",
        ):
            with self.subTest(pattern=pattern):
                self.assert_refused(make_plan(f"backend {pattern}"), "selects the whole test set")

    def test_refuses_unclean_forms_of_refused_paths(self) -> None:
        for line in (
            "backend ./internal/./...",
            "backend ./internal//...",
            "backend ./e2e/./upgrade",
            "backend ./internal/lib/process/",
            "frontend src/features/.",
            "frontend src//features",
        ):
            with self.subTest(line=line):
                self.assert_refused(make_plan(line), "empty or `.` element")

    def test_refuses_patterns_that_reach_the_upgrade_test(self) -> None:
        for pattern in (
            "./e2e/...",
            "./e2e/upgrade",
            "./e2e/upgrade/...",
            "./e2e/upgrade/upgradecheck",
        ):
            with self.subTest(pattern=pattern):
                self.assert_refused(make_plan(f"backend {pattern}"), "runs as `suite upgrade`")

    def test_refuses_a_wildcard_inside_a_pattern(self) -> None:
        self.assert_refused(
            make_plan("backend ./internal/.../e2e"), "other than as its last element"
        )

    def test_refuses_the_frontend_tree_and_its_layers(self) -> None:
        for path in ("src", "src/features", "src/shared/"):
            with self.subTest(path=path):
                self.assert_refused(make_plan(f"frontend {path}"), "selects the whole test set")

    def test_refuses_an_unknown_suite(self) -> None:
        self.assert_refused(make_plan("suite test"), "names an unknown suite")

    def test_refuses_commands(self) -> None:
        for line in ("mise run test", "mise run backend:test-package ./internal/lib/..."):
            with self.subTest(line=line):
                self.assert_refused(make_plan(line), "never commands")

    def test_refuses_flags_other_than_the_filter(self) -> None:
        for line in (
            "backend ./internal/lib/process -parallel 1",
            "frontend src/features/edit-notifier -run x",
            "suite scripts extra",
        ):
            with self.subTest(line=line):
                self.assert_refused(make_plan(line), "does not match")


class CommandTest(unittest.TestCase):
    def run_script(self, action: str, plan_text: str | None) -> subprocess.CompletedProcess[str]:
        with tempfile.TemporaryDirectory() as directory:
            change_directory = Path(directory) / "notify-on-retry"
            change_directory.mkdir()
            if plan_text is not None:
                (change_directory / "tests.md").write_text(plan_text)
            return subprocess.run(
                [sys.executable, str(SCRIPT), action, str(change_directory)],
                capture_output=True,
                text=True,
                check=False,
            )

    def test_check_names_a_missing_plan(self) -> None:
        completed = self.run_script("check", None)

        self.assertEqual(completed.returncode, 1)
        self.assertIn("The test plan of `notify-on-retry` is not valid", completed.stderr)
        self.assertIn("has no test plan", completed.stderr)
        self.assertIn("Allowed selectors", completed.stderr)

    def test_check_accepts_a_valid_plan_silently(self) -> None:
        completed = self.run_script("check", make_plan("suite scripts"))

        self.assertEqual((completed.returncode, completed.stdout, completed.stderr), (0, "", ""))

    def test_list_prints_tab_separated_selectors(self) -> None:
        completed = self.run_script(
            "list",
            make_plan(
                "backend ./internal/modules/notifiers/...",
                "frontend src/features/edit-notifier -t 'shows the error'",
                "suite e2e-browser",
            ),
        )

        self.assertEqual(completed.returncode, 0)
        self.assertEqual(
            completed.stdout.splitlines(),
            [
                "backend\t./internal/modules/notifiers/...\t",
                "frontend\tsrc/features/edit-notifier\tshows the error",
                "suite\te2e-browser",
            ],
        )

    def test_list_prints_none_for_an_empty_selection(self) -> None:
        completed = self.run_script("list", make_plan("none"))

        self.assertEqual((completed.returncode, completed.stdout), (0, "none\n"))

    def test_list_refuses_an_invalid_plan(self) -> None:
        completed = self.run_script("list", make_plan("backend ./..."))

        self.assertEqual((completed.returncode, completed.stdout), (1, ""))
        self.assertIn("selects the whole test set", completed.stderr)


if __name__ == "__main__":
    unittest.main()
