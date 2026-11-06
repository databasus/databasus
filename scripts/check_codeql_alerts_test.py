import json
import os
import subprocess
import tempfile
import textwrap
import unittest
from collections.abc import Mapping, Sequence
from pathlib import Path
from typing import TypedDict

ROOT = Path(__file__).resolve().parents[1]
REPOSITORY = "databasus/databasus"
REF = "refs/heads/main"
COMMIT = "abc123"
RELEASE_ANALYSIS_KEY = ".github/workflows/ci-release.yml:analyze"
SCHEDULED_ANALYSIS_KEY = ".github/workflows/codeql.yml:analyze"
CATEGORIES = ("/language:go", "/language:javascript-typescript", "/language:actions")

STUB_GH = """\
    #!/usr/bin/env python3
    import json, os, sys

    endpoint = next(argument for argument in sys.argv[1:] if argument.startswith("repos/"))
    with open(os.environ["STUB_GH_CALLS"], "a") as calls:
        calls.write(endpoint + "\\n")

    responses = json.load(open(os.environ["STUB_GH_RESPONSES"]))
    for fragment, response in responses.items():
        if fragment in endpoint:
            pages = response["pages"] if isinstance(response, dict) else [response]
            for page in pages if "--paginate" in sys.argv else pages[:1]:
                print(json.dumps(page))
            sys.exit(0)

    print(f"gh: not found: {endpoint}", file=sys.stderr)
    sys.exit(1)
"""


class Analysis(TypedDict):
    analysis_key: str
    category: str
    commit_sha: str


class AlertRule(TypedDict):
    id: str


class Alert(TypedDict):
    number: int
    html_url: str
    rule: AlertRule


class Instance(TypedDict):
    analysis_key: str
    state: str


def get_release_analyses() -> list[Analysis]:
    return [
        {"analysis_key": RELEASE_ANALYSIS_KEY, "category": category, "commit_sha": COMMIT}
        for category in CATEGORIES
    ]


def get_alert(number: int) -> Alert:
    return {
        "number": number,
        "html_url": f"https://example.test/alerts/{number}",
        "rule": {"id": "go/rule"},
    }


def get_instance(analysis_key: str, state: str = "open") -> Instance:
    return {"analysis_key": analysis_key, "state": state}


class CheckCodeqlAlertsTest(unittest.TestCase):
    def setUp(self) -> None:
        workspace = tempfile.TemporaryDirectory()
        self.addCleanup(workspace.cleanup)
        self.workspace = Path(workspace.name)

        stubs_directory = self.workspace / "stubs"
        stubs_directory.mkdir()
        stub_path = stubs_directory / "gh"
        stub_path.write_text(textwrap.dedent(STUB_GH))
        stub_path.chmod(0o755)

        self.calls_path = self.workspace / "calls"
        self.calls_path.touch()
        self.responses_path = self.workspace / "responses.json"
        self.environment = {
            **os.environ,
            "PATH": f"{stubs_directory}:{os.environ['PATH']}",
            "STUB_GH_CALLS": str(self.calls_path),
            "STUB_GH_RESPONSES": str(self.responses_path),
        }

    def run_severity_gate(
        self,
        analyses: Sequence[Analysis] | None = None,
        critical: Sequence[Alert] = (),
        high: Sequence[Alert] = (),
        instances: Mapping[int, object] | None = None,
    ) -> subprocess.CompletedProcess[str]:
        responses = {
            "code-scanning/analyses": get_release_analyses() if analyses is None else analyses,
            "severity=critical": list(critical),
            "severity=high": list(high),
        }
        for alert_number, alert_instances in (instances or {}).items():
            responses[f"alerts/{alert_number}/instances"] = alert_instances

        return self.run_severity_gate_with_responses(responses)

    def run_severity_gate_with_responses(
        self, responses: Mapping[str, object]
    ) -> subprocess.CompletedProcess[str]:
        self.responses_path.write_text(json.dumps(responses))

        return subprocess.run(
            ["bash", str(ROOT / "scripts/check_codeql_alerts.sh"), REPOSITORY, REF, COMMIT],
            env=self.environment,
            capture_output=True,
            text=True,
        )

    def test_fails_and_lists_critical_alert_open_in_the_release_analysis(self) -> None:
        severity_gate = self.run_severity_gate(
            critical=[get_alert(7)], instances={7: [get_instance(RELEASE_ANALYSIS_KEY)]}
        )

        self.assertEqual(severity_gate.returncode, 1, severity_gate.stderr)
        self.assertIn("https://example.test/alerts/7", severity_gate.stdout)

    def test_passes_when_critical_alert_is_open_only_in_the_scheduled_analysis(self) -> None:
        severity_gate = self.run_severity_gate(
            critical=[get_alert(7)],
            instances={
                7: [
                    get_instance(RELEASE_ANALYSIS_KEY, "fixed"),
                    get_instance(SCHEDULED_ANALYSIS_KEY),
                ]
            },
        )

        self.assertEqual(severity_gate.returncode, 0, severity_gate.stderr)

    def test_fails_when_the_commit_misses_one_category_of_the_release_analysis(self) -> None:
        severity_gate = self.run_severity_gate(analyses=get_release_analyses()[:2])

        self.assertEqual(severity_gate.returncode, 1)
        self.assertIn("/language:actions", severity_gate.stdout)

    def test_fails_when_the_analyses_belong_to_another_key_or_commit(self) -> None:
        other_key: list[Analysis] = [
            {**analysis, "analysis_key": SCHEDULED_ANALYSIS_KEY}
            for analysis in get_release_analyses()
        ]
        other_commit: list[Analysis] = [
            {**analysis, "commit_sha": "def456"} for analysis in get_release_analyses()
        ]

        for analyses in (other_key, other_commit):
            with self.subTest(analyses=analyses):
                self.assertEqual(self.run_severity_gate(analyses=analyses).returncode, 1)

    def test_passes_with_a_warning_for_high_alert_open_in_the_release_analysis(self) -> None:
        severity_gate = self.run_severity_gate(
            high=[get_alert(9)], instances={9: [get_instance(RELEASE_ANALYSIS_KEY)]}
        )

        self.assertEqual(severity_gate.returncode, 0, severity_gate.stderr)
        self.assertIn("::warning::", severity_gate.stdout)
        self.assertIn("https://example.test/alerts/9", severity_gate.stdout)

    def test_prints_no_warning_for_high_alert_open_only_in_the_scheduled_analysis(self) -> None:
        severity_gate = self.run_severity_gate(
            high=[get_alert(9)], instances={9: [get_instance(SCHEDULED_ANALYSIS_KEY)]}
        )

        self.assertEqual(severity_gate.returncode, 0, severity_gate.stderr)
        self.assertNotIn("::warning::", severity_gate.stdout)

    def test_passes_without_alerts(self) -> None:
        severity_gate = self.run_severity_gate()

        self.assertEqual(severity_gate.returncode, 0, severity_gate.stderr)

    def test_fails_when_analyses_cannot_be_read(self) -> None:
        severity_gate = self.run_severity_gate_with_responses(
            {"severity=critical": [], "severity=high": []}
        )

        self.assertNotEqual(severity_gate.returncode, 0)

    def test_fails_when_critical_alerts_cannot_be_read(self) -> None:
        severity_gate = self.run_severity_gate_with_responses(
            {"code-scanning/analyses": get_release_analyses(), "severity=high": []}
        )

        self.assertNotEqual(severity_gate.returncode, 0)

    def test_fails_when_high_alerts_cannot_be_read(self) -> None:
        severity_gate = self.run_severity_gate_with_responses(
            {"code-scanning/analyses": get_release_analyses(), "severity=critical": []}
        )

        self.assertNotEqual(severity_gate.returncode, 0)

    def test_fails_on_a_critical_alert_from_a_later_page(self) -> None:
        severity_gate = self.run_severity_gate_with_responses(
            {
                "code-scanning/analyses": get_release_analyses(),
                "severity=critical": {"pages": [[], [get_alert(7)]]},
                "severity=high": [],
                "alerts/7/instances": [get_instance(RELEASE_ANALYSIS_KEY)],
            }
        )

        self.assertEqual(severity_gate.returncode, 1, severity_gate.stderr)
        self.assertIn("https://example.test/alerts/7", severity_gate.stdout)

    def test_fails_on_a_critical_alert_whose_release_instance_is_on_a_later_page(self) -> None:
        severity_gate = self.run_severity_gate(
            critical=[get_alert(7)],
            instances={
                7: {
                    "pages": [
                        [get_instance(SCHEDULED_ANALYSIS_KEY)],
                        [get_instance(RELEASE_ANALYSIS_KEY)],
                    ]
                }
            },
        )

        self.assertEqual(severity_gate.returncode, 1, severity_gate.stderr)
        self.assertIn("https://example.test/alerts/7", severity_gate.stdout)

    def test_fails_when_the_instances_response_is_not_a_list(self) -> None:
        severity_gate = self.run_severity_gate(
            critical=[get_alert(7)], instances={7: {"pages": [{"message": "odd"}]}}
        )

        self.assertNotEqual(severity_gate.returncode, 0)

    def test_fails_when_instances_cannot_be_read(self) -> None:
        severity_gate = self.run_severity_gate(critical=[get_alert(7)])

        self.assertNotEqual(severity_gate.returncode, 0)

    def test_asks_only_for_open_codeql_alerts_of_the_given_ref(self) -> None:
        self.run_severity_gate()

        alert_requests = [
            call for call in self.calls_path.read_text().splitlines() if "/alerts?" in call
        ]
        self.assertEqual(len(alert_requests), 2)
        for alert_request in alert_requests:
            with self.subTest(alert_request=alert_request):
                self.assertTrue(
                    alert_request.startswith(f"repos/{REPOSITORY}/code-scanning/alerts?")
                )
                self.assertIn("state=open", alert_request)
                self.assertIn("tool_name=CodeQL", alert_request)
                self.assertIn(f"ref={REF}", alert_request)

    def test_passes_after_the_only_critical_alert_is_dismissed(self) -> None:
        blocked_gate = self.run_severity_gate(
            critical=[get_alert(7)], instances={7: [get_instance(RELEASE_ANALYSIS_KEY)]}
        )
        self.assertEqual(blocked_gate.returncode, 1)

        rerun_gate = self.run_severity_gate(critical=[])

        self.assertEqual(rerun_gate.returncode, 0, rerun_gate.stderr)


if __name__ == "__main__":
    unittest.main()
