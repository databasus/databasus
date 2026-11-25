#!/usr/bin/env python3
"""Turns the `go test -json` stream on standard input back into readable text output.

Usage: go test -json ... 2>&1 | go_test_report.py <ran tests file>

It prints what `go test` prints without `-json`, package results, build errors and the output of
failed tests, and also the output of skipped tests, so that a skip and its reason show in the run's
log. The output of passed tests is dropped. It appends one line per finished test,
`<package>\t<test name>\t<pass|fail|skip>`, to the ran tests file. scripts/test.sh reads that file
to fail a selector of a test plan that ran no test at the depth of its filter.
"""

import json
import sys
from pathlib import Path
from typing import TextIO

# The verbose frames that `go test -json` adds around every test; plain `go test` prints none of them.
VERBOSE_FRAME_PREFIXES = ("=== RUN", "=== PAUSE", "=== CONT", "=== NAME")
PACKAGE_PASS_LINE = "PASS\n"


class GoTestReport:
    def __init__(self, output: TextIO, ran_tests: TextIO) -> None:
        self.output = output
        self.ran_tests = ran_tests
        self.test_outputs: dict[tuple[str, str], list[str]] = {}

    def read_line(self, line: str) -> None:
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            self.write(line if line.endswith("\n") else line + "\n")
            return
        if not isinstance(event, dict):
            self.write(line)
            return

        action = str(event.get("Action", ""))
        package = str(event.get("Package", ""))
        test = str(event.get("Test", ""))
        text = str(event.get("Output", ""))
        if action == "build-output":
            self.write(text)
        elif action == "output" and not test:
            if text != PACKAGE_PASS_LINE:
                self.write(text)
        elif action == "output":
            if not text.lstrip().startswith(VERBOSE_FRAME_PREFIXES):
                self.test_outputs.setdefault((package, test), []).append(text)
        elif action in ("pass", "fail", "skip") and test:
            self.finish_test(package, test, action)

    def finish_test(self, package: str, test: str, action: str) -> None:
        test_output = self.test_outputs.pop((package, test), [])
        if action in ("fail", "skip"):
            for text in test_output:
                self.write(text)
        self.ran_tests.write(f"{package}\t{test}\t{action}\n")
        self.ran_tests.flush()

    def finish(self) -> None:
        # A test still running when the stream ends was cut off by a timeout or a crash, so its
        # output explains the failure.
        for test_output in self.test_outputs.values():
            for text in test_output:
                self.write(text)
        self.test_outputs.clear()

    def write(self, text: str) -> None:
        self.output.write(text)
        self.output.flush()


def main(arguments: list[str]) -> int:
    if len(arguments) != 1:
        print(__doc__, file=sys.stderr)
        return 2

    with Path(arguments[0]).open("a") as ran_tests:
        report = GoTestReport(sys.stdout, ran_tests)
        for line in sys.stdin:
            report.read_line(line)
        report.finish()
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
