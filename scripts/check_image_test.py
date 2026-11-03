import os
from pathlib import Path
import re
import subprocess
import tempfile
import textwrap
import unittest

ROOT = Path(__file__).resolve().parents[1]
IMAGE = "databasus:smoke-test"

STUB_SOURCES = {
    "docker": """
        echo "docker $*" >>"$STUB_CALLS"
        case "$1" in
            logs) echo "${STUB_DOCKER_LOGS:-}" ;;
            exec) exit "${STUB_DOCKER_EXEC_EXIT_STATUS:-0}" ;;
        esac
    """,
    "curl": r"""
        output_path=""
        status_format=""
        url=""
        while [[ $# -gt 0 ]]; do
            case "$1" in
                -o) output_path=$2; shift 2 ;;
                -w) status_format=$2; shift 2 ;;
                -s|-S|-f|-sS|-sf) shift ;;
                *) url=$1; shift ;;
            esac
        done

        case "$url" in
            */api/v1/system/health)
                if [[ -n "${STUB_HEALTH_CODE:-}" ]]; then
                    printf '%s' "$STUB_HEALTH_CODE"
                    exit 0
                fi
                if [[ -n "${STUB_HEALTH_BODY:-}" ]]; then
                    printf '%s' "$STUB_HEALTH_BODY" >"$output_path"
                else
                    printf '{"status":"ok"}' >"$output_path"
                fi
                printf '200'
                ;;
            */api/v1/system/verification-agent*)
                echo "curl $url" >>"$STUB_CALLS"
                printf '%b' "${STUB_AGENT_BODY:-\\x7fELF-agent}" >"$output_path"
                printf '%s' "${STUB_AGENT_CODE:-200}"
                ;;
            *)
                echo "${STUB_INDEX_BODY:-<div id=\"root\"></div>}"
                ;;
        esac
    """,
    "trivy": """
        echo "trivy $*" >>"$STUB_CALLS"
        if [[ " $* " == *" --severity CRITICAL "* ]]; then
            exit "${STUB_TRIVY_CRITICAL_EXIT_STATUS:-0}"
        fi
    """,
    "sleep": "exit 0",
}


class CheckImageTest(unittest.TestCase):
    def setUp(self):
        workspace = tempfile.TemporaryDirectory()
        self.addCleanup(workspace.cleanup)
        self.workspace = Path(workspace.name)

        stubs_directory = self.workspace / "stubs"
        stubs_directory.mkdir()
        for name, source in STUB_SOURCES.items():
            stub_path = stubs_directory / name
            stub_path.write_text("#!/usr/bin/env bash\n" + textwrap.dedent(source))
            stub_path.chmod(0o755)

        self.calls_path = self.workspace / "calls"
        self.calls_path.touch()
        self.environment = {
            **os.environ,
            "PATH": f"{stubs_directory}:{os.environ['PATH']}",
            "STUB_CALLS": str(self.calls_path),
            "TMPDIR": str(self.workspace),
        }

    def run_image_check(self, **stub_settings):
        return subprocess.run(
            ["bash", str(ROOT / "scripts/check_image.sh"), IMAGE],
            env={**self.environment, **stub_settings},
            capture_output=True,
            text=True,
        )

    def read_calls(self):
        return self.calls_path.read_text().splitlines()

    def assert_container_removed(self):
        started_container = re.search(r"^docker run -d --name (\S+)", self.calls_path.read_text(), re.M).group(1)

        self.assertIn(f"docker rm -f {started_container}", self.read_calls())

    def test_passes_when_the_image_is_healthy(self):
        image_check = self.run_image_check()

        self.assertEqual(image_check.returncode, 0, image_check.stdout + image_check.stderr)
        self.assert_container_removed()

    def test_fails_when_the_health_check_never_answers(self):
        image_check = self.run_image_check(STUB_HEALTH_CODE="503")

        self.assertEqual(image_check.returncode, 1)
        self.assertIn("healthcheck never returned valid JSON", image_check.stdout)
        self.assert_container_removed()

    def test_fails_when_the_health_response_is_not_valid(self):
        image_check = self.run_image_check(STUB_HEALTH_BODY="<html></html>")

        self.assertEqual(image_check.returncode, 1)
        self.assertIn("healthcheck never returned valid JSON", image_check.stdout)
        self.assert_container_removed()

    def test_fails_when_index_html_has_no_react_root(self):
        image_check = self.run_image_check(STUB_INDEX_BODY="<html></html>")

        self.assertEqual(image_check.returncode, 1)
        self.assertIn("index.html did not contain React root div", image_check.stdout)
        self.assert_container_removed()

    def test_fails_when_the_verification_agent_download_fails(self):
        image_check = self.run_image_check(STUB_AGENT_CODE="404")

        self.assertEqual(image_check.returncode, 1)
        self.assertIn("verification agent (amd64) download failed: HTTP 404", image_check.stdout)
        self.assert_container_removed()

    def test_fails_when_the_verification_agent_is_not_an_elf_binary(self):
        image_check = self.run_image_check(STUB_AGENT_BODY="<html>")

        self.assertEqual(image_check.returncode, 1)
        self.assertIn("verification agent (amd64) is not an ELF binary", image_check.stdout)
        self.assert_container_removed()

    def test_checks_the_verification_agent_of_both_architectures(self):
        self.run_image_check()

        agent_calls = [call for call in self.read_calls() if call.startswith("curl ")]
        self.assertEqual(len(agent_calls), 2)
        self.assertTrue(agent_calls[0].endswith("arch=amd64"))
        self.assertTrue(agent_calls[1].endswith("arch=arm64"))

    def test_fails_when_a_console_command_cannot_reach_the_metadata_database(self):
        image_check = self.run_image_check(STUB_DOCKER_EXEC_EXIT_STATUS="1")

        self.assertEqual(image_check.returncode, 1)
        self.assert_container_removed()

    def test_fails_when_a_bundled_client_tool_cannot_run(self):
        image_check = self.run_image_check(STUB_DOCKER_LOGS="client tools check failed")

        self.assertEqual(image_check.returncode, 1)
        self.assertIn("a bundled client tool cannot run in the image", image_check.stdout)
        self.assert_container_removed()

    def test_fails_when_the_image_has_a_fixable_critical_vulnerability(self):
        image_check = self.run_image_check(STUB_TRIVY_CRITICAL_EXIT_STATUS="1")

        self.assertEqual(image_check.returncode, 1)
        self.assert_container_removed()

    def test_scans_block_only_on_fixable_critical_vulnerabilities(self):
        image_check = self.run_image_check()

        self.assertEqual(image_check.returncode, 0, image_check.stdout + image_check.stderr)
        trivy_calls = [call for call in self.read_calls() if call.startswith("trivy ")]
        self.assertEqual(len(trivy_calls), 2)
        self.assertIn("--severity CRITICAL --ignore-unfixed", trivy_calls[0])
        self.assertIn("--exit-code 1", trivy_calls[0])
        self.assertIn(IMAGE, trivy_calls[0])
        self.assertIn("--severity HIGH --ignore-unfixed", trivy_calls[1])
        self.assertIn("--exit-code 0", trivy_calls[1])

    def test_reports_high_vulnerabilities_even_after_a_critical_failure(self):
        self.run_image_check(STUB_TRIVY_CRITICAL_EXIT_STATUS="1")

        self.assertTrue(any("--severity HIGH" in call for call in self.read_calls()))


if __name__ == "__main__":
    unittest.main()
