import json
import os
import socket
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
TASK = ROOT / "mise-tasks/agent/e2e-podman"
PACKAGE = "databasus-backend/internal/agent/e2e"
SCENARIOS = (
    "TestVerifyBackup/completes_with_an_encrypted_backup_on_postgres_19",
    "TestVerifyBackup/fails_after_one_attempt_when_the_backup_is_broken",
    "TestRestoreContainer/runs_the_pinned_server_version_without_root_or_capabilities_when_restoring_a_backup",
)


class AgentE2EPodmanTaskTest(unittest.TestCase):
    def run_task(self, finished_scenarios: tuple[str, ...]) -> subprocess.CompletedProcess[str]:
        with tempfile.TemporaryDirectory(prefix="agent-e2e-podman-") as workspace:
            directory = Path(workspace)
            events = directory / "events.jsonl"
            events.write_text(
                "".join(
                    json.dumps({"Action": "pass", "Package": PACKAGE, "Test": scenario}) + "\n"
                    for scenario in finished_scenarios
                )
            )
            arguments = directory / "arguments.json"
            binary = directory / "bin"
            binary.mkdir()
            go = binary / "go"
            go.write_text(
                "#!/usr/bin/env python3\n"
                "import json, os, sys\n"
                'with open(os.environ["ARGUMENTS_PATH"], "w") as arguments:\n'
                "    json.dump(sys.argv[1:], arguments)\n"
                'sys.stdout.write(open(os.environ["EVENTS_PATH"]).read())\n'
            )
            go.chmod(0o755)

            engine_socket = directory / "engine.sock"
            with socket.socket(socket.AF_UNIX) as listener:
                listener.bind(str(engine_socket))
                listener.listen()
                execution = subprocess.run(
                    ["bash", str(TASK)],
                    cwd=ROOT / "backend",
                    env={
                        **os.environ,
                        "PATH": f"{binary}:{os.environ['PATH']}",
                        "E2E_ENGINE_SOCKET": str(engine_socket),
                        "EVENTS_PATH": str(events),
                        "ARGUMENTS_PATH": str(arguments),
                        "CI": "",
                    },
                    capture_output=True,
                    text=True,
                    timeout=30,
                )

            self.go_arguments: list[str] = json.loads(arguments.read_text())
        return execution

    def test_passes_when_every_scenario_finished(self) -> None:
        execution = self.run_task(SCENARIOS)

        self.assertEqual(execution.returncode, 0, execution.stderr)
        self.assertIn("-json", self.go_arguments)
        run_pattern = self.go_arguments[self.go_arguments.index("-run") + 1]
        self.assertEqual(
            run_pattern,
            "^(TestRestoreContainer|TestVerifyBackup)$/^("
            "completes_with_an_encrypted_backup_on_postgres_19"
            "|fails_after_one_attempt_when_the_backup_is_broken"
            "|runs_the_pinned_server_version_without_root_or_capabilities_when_restoring_a_backup"
            ")$",
        )

    def test_fails_and_names_a_scenario_that_reported_no_result(self) -> None:
        execution = self.run_task(SCENARIOS[:2])

        self.assertNotEqual(execution.returncode, 0)
        self.assertIn(
            "Podman subset scenario reported no result: " + SCENARIOS[2], execution.stderr
        )


if __name__ == "__main__":
    unittest.main()
