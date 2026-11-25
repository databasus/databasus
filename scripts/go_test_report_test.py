import io
import json
import unittest

from go_test_report import GoTestReport

PACKAGE = "databasus-backend/internal/modules/notifiers"
OTHER_PACKAGE = "databasus-backend/internal/lib/process"


def event(action: str, package: str = PACKAGE, test: str = "", output: str = "") -> str:
    fields: dict[str, str] = {"Action": action, "Package": package}
    if test:
        fields["Test"] = test
    if output:
        fields["Output"] = output
    return json.dumps(fields) + "\n"


def report_stream(lines: list[str]) -> tuple[str, list[str]]:
    output = io.StringIO()
    ran_tests = io.StringIO()
    report = GoTestReport(output, ran_tests)
    for line in lines:
        report.read_line(line)
    report.finish()
    return output.getvalue(), ran_tests.getvalue().splitlines()


class GoTestReportTest(unittest.TestCase):
    def test_prints_only_the_result_of_a_passing_package(self) -> None:
        printed, ran_tests = report_stream(
            [
                event("start"),
                event("run", test="TestSend"),
                event("output", test="TestSend", output="=== RUN   TestSend\n"),
                event("output", test="TestSend", output="    send_test.go:12: sent\n"),
                event("output", test="TestSend", output="--- PASS: TestSend (0.01s)\n"),
                event("pass", test="TestSend"),
                event("output", output="PASS\n"),
                event("output", output=f"ok  \t{PACKAGE}\t0.02s\n"),
                event("pass"),
            ]
        )

        self.assertEqual(printed, f"ok  \t{PACKAGE}\t0.02s\n")
        self.assertEqual(ran_tests, [f"{PACKAGE}\tTestSend\tpass"])

    def test_prints_the_output_of_a_failing_test(self) -> None:
        printed, ran_tests = report_stream(
            [
                event("run", test="TestSend"),
                event("output", test="TestSend", output="=== RUN   TestSend\n"),
                event("output", test="TestSend", output="    send_test.go:12: broken\n"),
                event("output", test="TestSend", output="--- FAIL: TestSend (0.01s)\n"),
                event("fail", test="TestSend"),
                event("output", output="FAIL\n"),
                event("output", output=f"FAIL\t{PACKAGE}\t0.02s\n"),
                event("fail"),
            ]
        )

        self.assertEqual(
            printed,
            "    send_test.go:12: broken\n--- FAIL: TestSend (0.01s)\nFAIL\n"
            f"FAIL\t{PACKAGE}\t0.02s\n",
        )
        self.assertEqual(ran_tests, [f"{PACKAGE}\tTestSend\tfail"])

    def test_records_a_top_level_test_whose_subtests_were_filtered_out(self) -> None:
        _, ran_tests = report_stream(
            [
                event("run", test="TestBackupAndRestorePostgresql"),
                event("pass", test="TestBackupAndRestorePostgresql"),
            ]
        )

        self.assertEqual(ran_tests, [f"{PACKAGE}\tTestBackupAndRestorePostgresql\tpass"])

    def test_records_subtests_by_their_full_name(self) -> None:
        _, ran_tests = report_stream(
            [
                event("pass", test="TestSource/postgres:17/TestConnection"),
                event("pass", test="TestSource/postgres:17"),
                event("pass", test="TestSource"),
            ]
        )

        self.assertEqual(
            ran_tests,
            [
                f"{PACKAGE}\tTestSource/postgres:17/TestConnection\tpass",
                f"{PACKAGE}\tTestSource/postgres:17\tpass",
                f"{PACKAGE}\tTestSource\tpass",
            ],
        )

    def test_prints_and_records_a_skipped_test_with_its_reason(self) -> None:
        printed, ran_tests = report_stream(
            [
                event("output", test="TestNeon", output="=== RUN   TestNeon\n"),
                event(
                    "output",
                    test="TestNeon",
                    output="    neon_test.go:20: TEST_NEON_DATABASE_URL is empty\n",
                ),
                event("output", test="TestNeon", output="--- SKIP: TestNeon (0.00s)\n"),
                event("skip", test="TestNeon"),
            ]
        )

        self.assertEqual(
            printed,
            "    neon_test.go:20: TEST_NEON_DATABASE_URL is empty\n--- SKIP: TestNeon (0.00s)\n",
        )
        self.assertEqual(ran_tests, [f"{PACKAGE}\tTestNeon\tskip"])

    def test_keeps_interleaved_packages_apart(self) -> None:
        printed, ran_tests = report_stream(
            [
                event("output", test="TestSend", output="    notifier detail\n"),
                event(
                    "output", package=OTHER_PACKAGE, test="TestSend", output="    process detail\n"
                ),
                event("pass", test="TestSend"),
                event("fail", package=OTHER_PACKAGE, test="TestSend"),
            ]
        )

        self.assertEqual(printed, "    process detail\n")
        self.assertEqual(
            ran_tests, [f"{PACKAGE}\tTestSend\tpass", f"{OTHER_PACKAGE}\tTestSend\tfail"]
        )

    def test_prints_build_errors_and_lines_that_are_not_json(self) -> None:
        printed, ran_tests = report_stream(
            [
                json.dumps({"Action": "build-output", "Output": "c_test.go:3:28: undefined: run\n"})
                + "\n",
                "go: downloading example.com/module v1.0.0\n",
            ]
        )

        self.assertEqual(
            printed, "c_test.go:3:28: undefined: run\ngo: downloading example.com/module v1.0.0\n"
        )
        self.assertEqual(ran_tests, [])

    def test_prints_the_output_of_a_test_cut_off_by_a_timeout(self) -> None:
        printed, _ = report_stream(
            [
                event("output", test="TestWait", output="    wait_test.go:9: waiting\n"),
                event("output", output="panic: test timed out after 15m0s\n"),
            ]
        )

        self.assertEqual(
            printed, "panic: test timed out after 15m0s\n    wait_test.go:9: waiting\n"
        )


if __name__ == "__main__":
    unittest.main()
