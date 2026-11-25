#!/usr/bin/env python3
"""Reads the selection of an OpenSpec change's test plan, its tests.md.

Usage: test_plan.py check <change directory>
       test_plan.py list <change directory>

`check` validates the selection for the review hooks and prints the reason on failure. `list`
prints one selector per line, its fields separated by tabs, for scripts/test.sh. Neither runs a
selected test. docs/testing.md, "Test plans", describes the format.
"""

import re
import shlex
import sys
from dataclasses import dataclass
from pathlib import Path

PLAN_FILE_NAME = "tests.md"
SELECTION_HEADING = "## Selection"
SELECTION_FENCE = "```tests"
NOTHING_SELECTED = "none"

SUITE_NAMES = (
    "browser-ui",
    "e2e-browser",
    "e2e-unit",
    "lifecycle",
    "lifecycle-interruption-checks",
    "upgrade",
    "scripts",
    "review-hooks",
)
# Each of these trees holds nearly all backend tests or all tests of one layer; a plan names a part.
WHOLE_BACKEND_PATTERNS = frozenset(
    {
        "./...",
        "./internal/...",
        "./internal/modules/...",
        "./internal/engines/...",
        "./internal/lib/...",
    }
)
UPGRADE_TEST_DIRECTORY = "./e2e/upgrade"
WHOLE_FRONTEND_PATHS = frozenset(
    {"src", "src/pages", "src/widgets", "src/features", "src/entities", "src/shared"}
)
ALLOWED_FORMS = (
    f"Allowed selectors, one per line in one {SELECTION_FENCE} block under {SELECTION_HEADING}:\n"
    "- `backend <package pattern> [-run <regex>]`, a pattern relative to backend/ such as "
    "`./internal/modules/notifiers/...`\n"
    "- `frontend <path> [-t <test name>]`, a path relative to frontend/ under `src/`\n"
    f"- `suite <name>`, one of {', '.join(f'`{name}`' for name in SUITE_NAMES)}\n"
    f"- `{NOTHING_SELECTED}` as the only line, when the change can break no test"
)


class TestPlanError(Exception):
    pass


@dataclass(frozen=True)
class BackendSelector:
    pattern: str
    run_filter: str

    def get_fields(self) -> tuple[str, ...]:
        return ("backend", self.pattern, self.run_filter)


@dataclass(frozen=True)
class FrontendSelector:
    path: str
    test_name: str

    def get_fields(self) -> tuple[str, ...]:
        return ("frontend", self.path, self.test_name)


@dataclass(frozen=True)
class SuiteSelector:
    name: str

    def get_fields(self) -> tuple[str, ...]:
        return ("suite", self.name)


Selector = BackendSelector | FrontendSelector | SuiteSelector


def read_selection_lines(plan_text: str) -> list[str]:
    lines = plan_text.splitlines()
    try:
        start = next(index for index, line in enumerate(lines) if line.strip() == SELECTION_HEADING)
    except StopIteration:
        raise TestPlanError(f"the plan has no {SELECTION_HEADING} section") from None

    for index in range(start + 1, len(lines)):
        line = lines[index].strip()
        if line == SELECTION_FENCE:
            return read_block(lines[index + 1 :])
        if line.startswith("```") or re.match(r"#{1,2} ", line):
            break
    raise TestPlanError(
        f"the plan has no selection: no {SELECTION_FENCE} block under {SELECTION_HEADING}"
    )


def read_block(lines: list[str]) -> list[str]:
    for index, line in enumerate(lines):
        if line.strip() == "```":
            return [
                entry.strip()
                for entry in lines[:index]
                if entry.strip() and not entry.strip().startswith("#")
            ]
    raise TestPlanError(f"the {SELECTION_FENCE} block is not closed")


def parse_selection(plan_text: str) -> list[Selector]:
    lines = read_selection_lines(plan_text)
    if not lines:
        raise TestPlanError("the selection holds no selector")
    if NOTHING_SELECTED in lines:
        if len(lines) > 1:
            raise TestPlanError(f"`{NOTHING_SELECTED}` must be the only line of the selection")
        return []

    return [parse_selector(line) for line in lines]


def parse_selector(line: str) -> Selector:
    try:
        words = shlex.split(line)
    except ValueError as error:
        raise TestPlanError(f"`{line}` cannot be split into words: {error}") from None
    if any("\t" in word or not word for word in words):
        raise TestPlanError(f"`{line}` holds an empty word or a tab")

    kind, arguments = words[0], words[1:]
    if kind == "backend":
        pattern, run_filter = read_optional_filter(line, arguments, "-run")
        check_backend_pattern(line, pattern)
        return BackendSelector(pattern, run_filter)
    if kind == "frontend":
        path, test_name = read_optional_filter(line, arguments, "-t")
        return FrontendSelector(get_checked_frontend_path(line, path), test_name)
    if kind == "suite":
        if len(arguments) != 1:
            raise TestPlanError(f"`{line}` does not match the form `suite <name>`")
        if arguments[0] not in SUITE_NAMES:
            raise TestPlanError(f"`{line}` names an unknown suite")
        return SuiteSelector(arguments[0])
    raise TestPlanError(f"`{line}` is not a selector; a plan names tests, never commands")


def read_optional_filter(line: str, arguments: list[str], flag: str) -> tuple[str, str]:
    if len(arguments) == 1:
        return arguments[0], ""
    if len(arguments) == 3 and arguments[1] == flag:
        return arguments[0], arguments[2]
    raise TestPlanError(
        f"`{line}` does not match its form: a path, then optionally `{flag} <value>`"
    )


def check_backend_pattern(line: str, pattern: str) -> None:
    elements = pattern.split("/")
    if elements[0] != "." or len(elements) < 2 or ".." in elements:
        raise TestPlanError(f"`{line}` leaves the backend module: a pattern starts with `./`")
    if has_unclean_element(elements[1:]):
        raise TestPlanError(
            f"`{line}` holds an empty or `.` element; write the path in its clean form"
        )

    is_recursive = pattern.endswith("/...")
    base = pattern.removesuffix("/...") if is_recursive else pattern
    if "..." in base:
        raise TestPlanError(f"`{line}` uses `...` other than as its last element")
    if pattern in WHOLE_BACKEND_PATTERNS or (base == "." and is_recursive):
        raise TestPlanError(f"`{line}` selects the whole test set; a plan names a part of it")
    if is_upgrade_test_reached(base, is_recursive):
        raise TestPlanError(f"`{line}` reaches the upgrade test, which runs as `suite upgrade`")


# Go and Vitest read `a//b` and `a/./b` as `a/b`, so only the clean form can be compared with the
# refused patterns.
def has_unclean_element(elements: list[str]) -> bool:
    return any(element in ("", ".") for element in elements)


def is_upgrade_test_reached(base: str, is_recursive: bool) -> bool:
    if base == UPGRADE_TEST_DIRECTORY or base.startswith(UPGRADE_TEST_DIRECTORY + "/"):
        return True
    return is_recursive and UPGRADE_TEST_DIRECTORY.startswith(base + "/")


def get_checked_frontend_path(line: str, path: str) -> str:
    normalized_path = path.removesuffix("/")
    elements = normalized_path.split("/")
    if elements[0] != "src" or ".." in elements:
        raise TestPlanError(f"`{line}` leaves the frontend source tree: a path starts with `src/`")
    if has_unclean_element(elements):
        raise TestPlanError(
            f"`{line}` holds an empty or `.` element; write the path in its clean form"
        )
    if normalized_path in WHOLE_FRONTEND_PATHS:
        raise TestPlanError(f"`{line}` selects the whole test set; a plan names a part of it")
    return normalized_path


def read_plan(change_directory: Path) -> list[Selector]:
    plan_path = change_directory / PLAN_FILE_NAME
    try:
        plan_text = plan_path.read_text()
    except FileNotFoundError:
        raise TestPlanError(f"the change has no test plan: {plan_path} is missing") from None
    return parse_selection(plan_text)


def main(arguments: list[str]) -> int:
    if len(arguments) != 2 or arguments[0] not in ("check", "list"):
        print(__doc__, file=sys.stderr)
        return 2

    action, change_directory = arguments[0], Path(arguments[1])
    try:
        selectors = read_plan(change_directory)
    except TestPlanError as error:
        print(
            f"The test plan of `{change_directory.name}` is not valid: {error}.\n\n{ALLOWED_FORMS}",
            file=sys.stderr,
        )
        return 1

    if action == "list":
        for selector in selectors:
            print("\t".join(selector.get_fields()))
        if not selectors:
            print(NOTHING_SELECTED)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
