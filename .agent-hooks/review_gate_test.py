import io
import json
import os
import shlex
import shutil
import signal
import subprocess
import sys
import tempfile
import textwrap
import time
import tomllib
import unittest
from collections.abc import Mapping, Sequence
from dataclasses import replace
from pathlib import Path
from unittest import mock

import review_gate
from hook_configuration import HookDefinition, HookGroup, get_hook_configuration
from hook_response import HookResponse, get_hook_response
from review_state import Finding
from reviewer_call import MiseCall, ReviewerCall, is_mise_call, is_reviewer_call

HOOK_PATH = Path(__file__).resolve().parent / "review_gate.py"
CHANGE_NAME = "add-widgets"
PROPOSAL_NAME = "add-gadgets"
STUB_NAMES = ("codex", "claude", "openspec", "golangci-lint", "pnpm", "npm", "mise")

# The stub stands in for every external command: it takes its answer from <stubs>/<name>.json
# and records each call in <stubs>/calls, so a test sees the arguments, the working directory
# and the prompt. openspec answers by its first argument (openspec.json["status"]),
# npm by its script name (npm.json["build"]). sleep_seconds delays the answer.
STUB_SOURCE = textwrap.dedent(
    """
    import json
    import os
    from pathlib import Path
    import sys
    import time
    import uuid

    stubs_dir = Path(os.environ["REVIEW_GATE_STUBS"])
    name = Path(sys.argv[0]).name
    arguments = sys.argv[1:]
    prompt = sys.stdin.read() if name == "claude" or (name == "codex" and "exec" in arguments) else ""
    calls_dir = stubs_dir / "calls"
    calls_dir.mkdir(exist_ok=True)
    call_path = calls_dir / f"{name}-{uuid.uuid4().hex}.json"
    call_temporary_path = call_path.with_suffix(".tmp")
    call_temporary_path.write_text(
        json.dumps(
            {
                "name": name,
                "arguments": arguments,
                "cwd": os.getcwd(),
                "prompt": prompt,
                "effort_env": (next((argument.split("=", 1)[1].strip(chr(34)) for argument in arguments
                                     if argument.startswith("model_reasoning_effort=")), None)
                               if name == "codex" else os.environ.get("CLAUDE_CODE_EFFORT_LEVEL")),
                "hook_active": os.environ.get("AGENT_REVIEW_HOOK_ACTIVE"),
                "session_env": os.environ.get("CLAUDECODE"),
                "answer_mode": (oct(Path(arguments[arguments.index("--output-last-message") + 1]).parent.stat().st_mode & 0o777)
                                if "--output-last-message" in arguments else None),
                "started_at": time.time_ns(),
            }
        )
    )

    call_temporary_path.replace(call_path)

    config_path = stubs_dir / f"{name}.json"
    config = json.loads(config_path.read_text()) if config_path.exists() else {}
    if name == "codex" and "app-server" in arguments:
        if config.get("app_server_error"):
            print("WARNING: startup warning", file=sys.stderr)
            print(config["app_server_error"], file=sys.stderr)
            sys.exit(1)
        for line in sys.stdin:
            request = json.loads(line)
            if request.get("method") == "initialize":
                print(json.dumps({"id": request["id"], "result": {}}), flush=True)
            elif request.get("method") == "hooks/list":
                print(json.dumps({"id": request["id"], "result": config.get("hooks_response", {"data": []})}), flush=True)
            elif request.get("method") == "config/read":
                print(json.dumps({"id": request["id"], "result": {"config": {"features": config.get("features", {})}}}), flush=True)
        sys.exit(0)
    if name == "openspec":
        config = config.get(arguments[0], {})
    elif name == "npm":
        config = config.get(arguments[-1], {})

    if name == "codex" and any(argument.startswith('mcp_servers."') for argument in arguments):
        sys.stderr.write("invalid transport: CLI dotted keys preserve quote characters")
        sys.exit(1)

    if config.get("reject_overrides") and any(argument.startswith("mcp_servers=") for argument in arguments):
        sys.stderr.write("managed policy rejects reviewer restrictions")
        sys.exit(2)

    if config.get("nested_snapshot") and name == "codex" and "exec" in arguments:
        import subprocess
        subprocess.run([sys.executable, config["nested_snapshot"], "snapshot", "--client", "codex"],
                       input=json.dumps({"session_id": "session-1", "prompt": "nested reviewer"}),
                       text=True, check=True, capture_output=True)

    required_efforts = set(config.get("wait_for_efforts", []))
    if required_efforts and (name == "claude" or "exec" in arguments):
        deadline = time.monotonic() + 3
        while True:
            started_efforts = {
                json.loads(path.read_text())["effort_env"]
                for path in calls_dir.glob("*.json")
            }
            if required_efforts <= started_efforts:
                break
            if time.monotonic() >= deadline:
                sys.stderr.write("reviewers did not run concurrently")
                sys.exit(1)
            time.sleep(0.01)

    if name == "codex" and "exec" not in arguments:
        if config.get("metadata_exit_code"):
            sys.exit(config["metadata_exit_code"])
        time.sleep(config.get("metadata_seconds", 0))
        if "mcp" in arguments:
            sys.stdout.write(json.dumps(config.get("servers", [{"name": "browser"}])))
        elif "plugin" in arguments:
            sys.stdout.write(json.dumps(config.get("plugins", {"installed": [{"pluginId": "external@market"}]})))
        sys.exit(0)

    concurrent_target = config.get("wait_for_concurrent")
    if concurrent_target:
        import fcntl
        def update_concurrency(delta):
            with (stubs_dir / "concurrency.lock").open("a") as lock:
                fcntl.flock(lock, fcntl.LOCK_EX)
                state_path = stubs_dir / "concurrency.json"
                state = json.loads(state_path.read_text()) if state_path.exists() else {"active": 0, "peak": 0}
                state["active"] += delta
                state["peak"] = max(state["peak"], state["active"])
                state_path.write_text(json.dumps(state))
                return state
        state = update_concurrency(1)
        deadline = time.monotonic() + 5
        while state["peak"] < concurrent_target:
            if time.monotonic() >= deadline:
                sys.stderr.write("reviewers did not reach the concurrency barrier")
                sys.exit(1)
            time.sleep(0.01)
            state = update_concurrency(0)

    time.sleep(config.get("sleep_seconds", 0))
    if config.get("exit_code"):
        sys.stderr.write(config.get("output", ""))
        sys.exit(config["exit_code"])

    is_full = "Role:" in prompt
    result = config.get("result", {"findings": []} if is_full else {"ok": True, "reason": ""})
    result = config.get("full_result" if is_full else "quick_result", result)
    if name == "claude":
        sys.stdout.write(json.dumps({"structured_output": result}))
    elif name == "codex":
        answer_path = Path(arguments[arguments.index("--output-last-message") + 1])
        if not config.get("missing_answer"):
            answer_path.write_text(config.get("answer_text", json.dumps(result)))
        sys.stdout.write(config.get("progress", '{"ok": false, "reason": "misleading progress"}'))
    elif name == "openspec" and arguments[0] == "status":
        sys.stdout.write(config.get("output", json.dumps({"isPlanningComplete": True})))
    else:
        sys.stdout.write(config.get("output", ""))
    if concurrent_target:
        update_concurrency(-1)
    """
)
FAILING_HOOK_TEST = textwrap.dedent(
    """
    import unittest


    class SampleTest(unittest.TestCase):
        def test_reports_broken_gate(self):
            self.fail("gate returned block for an accepted turn")
    """
)


# The stub stands in for mise in the launcher tests: it records the subcommand, its arguments,
# the environment variable that forbids installs and its stdin to <stubs>/mise-calls.json, then
# answers from <stubs>/mise.json by subcommand (mise.json["ls"]). Like real mise, `ls --missing`
# prints nothing when every pinned version is installed, and `exec` runs even when one is missing.
MISE_STUB_SOURCE = textwrap.dedent(
    """
    import json
    import os
    from pathlib import Path
    import sys

    stubs_dir = Path(os.environ["REVIEW_GATE_STUBS"])
    subcommand = sys.argv[3]
    calls_path = stubs_dir / "mise-calls.json"
    calls = json.loads(calls_path.read_text()) if calls_path.exists() else []
    calls.append(
        {
            "subcommand": subcommand,
            "arguments": sys.argv[1:],
            "exec_auto_install": os.environ.get("MISE_EXEC_AUTO_INSTALL"),
            "stdin": sys.stdin.read() if subcommand == "exec" else "",
        }
    )
    calls_path.write_text(json.dumps(calls))

    config_path = stubs_dir / "mise.json"
    config = json.loads(config_path.read_text()) if config_path.exists() else {}
    answer = config.get(subcommand, {})
    sys.stdout.write(answer.get("stdout", "{}" if subcommand == "exec" else ""))
    sys.stderr.write(answer.get("stderr", ""))
    sys.exit(answer.get("exit_code", 0))
    """
)
# The check starts a child that outlives it and writes the child's pid to the path in its argument.
CHECK_WITH_CHILD_SOURCE = textwrap.dedent(
    """
    from pathlib import Path
    import subprocess
    import sys

    child = subprocess.Popen([sys.executable, "-c", "import time; time.sleep(60)"])
    Path(sys.argv[1]).write_text(str(child.pid))
    child.wait()
    """
)
# The same, but the child starts its own session, so it leaves the check's process group and keeps the output pipe open.
CHECK_WITH_ESCAPED_CHILD_SOURCE = CHECK_WITH_CHILD_SOURCE.replace(
    '"import time; time.sleep(60)"])', '"import time; time.sleep(60)"], start_new_session=True)'
)
LAUNCHER_PATH = HOOK_PATH.parent / "run.sh"
LAUNCHER_COMMANDS = ("bash", "mktemp", "tail", "tr", "rm")


class HookRepository:
    # A missing tool or an isolated PATH takes the machine's PATH away, so a copy installed there
    # stays out of reach; only git is linked in.
    def __init__(
        self,
        test_case: unittest.TestCase,
        missing_tools: Sequence[str] = (),
        is_path_isolated: bool = False,
    ) -> None:
        temporary_dir = tempfile.TemporaryDirectory(prefix="review-gate-test-")
        test_case.addCleanup(temporary_dir.cleanup)
        base = Path(temporary_dir.name)

        self.review_client = getattr(test_case, "review_client", "claude")
        self.full_effort = "high" if self.review_client == "codex" else "medium"
        self.quick_effort = "medium" if self.review_client == "codex" else "low"
        if hasattr(test_case, "review_client"):
            missing_tools = (*missing_tools, "claude" if self.review_client == "codex" else "codex")

        self.root = base / "repo"
        self.stubs_dir = base / "stubs"
        bin_dir = base / "bin"
        for directory in (self.root, self.stubs_dir, bin_dir):
            directory.mkdir()

        for name in STUB_NAMES:
            if name in missing_tools:
                continue
            stub = bin_dir / name
            stub.write_text(f"#!{sys.executable}\n{STUB_SOURCE}")
            stub.chmod(0o755)

        self.environment = {
            name: value
            for name, value in os.environ.items()
            if not name.startswith("GIT_") and name != "AGENT_REVIEW_HOOK_ACTIVE"
        }
        if missing_tools or is_path_isolated:
            system_dir = base / "system-bin"
            system_dir.mkdir()
            git_path = shutil.which("git")
            assert git_path is not None
            (system_dir / "git").symlink_to(git_path)
            self.environment["PATH"] = f"{bin_dir}{os.pathsep}{system_dir}"
        else:
            self.environment["PATH"] = f"{bin_dir}{os.pathsep}{os.environ['PATH']}"

        self.environment["AGENT_REVIEW_CLIENT"] = self.review_client
        self.environment["CODEX_HOME"] = str(base / "codex-home")
        self.environment["REVIEW_GATE_STUBS"] = str(self.stubs_dir)
        self.environment["PYTHONDONTWRITEBYTECODE"] = "1"

        self.git("init", "--quiet")
        self.git("config", "user.email", "test@example.com")
        self.git("config", "user.name", "test")

        self.write("README.md", "test repository\n")
        self.write_change(spec_names=("widgets",), tasks="- [x] 1.1 Build\n- [ ] 1.2 Verify\n")

        self.git("add", ".")
        self.git("commit", "--quiet", "-m", "init")

    def git(self, *arguments: str) -> None:
        subprocess.run(
            ["git", *arguments],
            cwd=self.root,
            env=self.environment,
            check=True,
            capture_output=True,
        )

    def write(self, relative_path: str, content: str) -> None:
        path = self.root / relative_path
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)

    def write_change(self, spec_names: Sequence[str], tasks: str, name: str = CHANGE_NAME) -> None:
        change_dir = f"openspec/changes/{name}"
        self.write(f"{change_dir}/proposal.md", "# Proposal\n")
        self.write(f"{change_dir}/design.md", "# Design\n")
        self.write(f"{change_dir}/tasks.md", f"# Tasks\n\n{tasks}")
        for spec_name in spec_names:
            self.write(f"{change_dir}/specs/{spec_name}/spec.md", f"# {spec_name}\n")

    def write_proposal(
        self,
        spec_names: Sequence[str] = ("gadgets",),
        tasks: str = "- [ ] 1.1 Build\n- [ ] 1.2 Verify\n",
    ) -> None:
        self.write_change(spec_names=spec_names, tasks=tasks, name=PROPOSAL_NAME)

    def complete_tasks(self, name: str = CHANGE_NAME, verification_section: str = "") -> None:
        self.write(
            f"openspec/changes/{name}/tasks.md",
            f"# Tasks\n\n- [x] 1.1 Build\n- [x] 1.2 Verify\n{verification_section}",
        )

    def set_reviewer_result(self, result: Mapping[str, object]) -> None:
        if "ok" in result and "findings" in result:
            self.set_stub(
                self.review_client,
                {
                    "quick_result": {"ok": result["ok"], "reason": result["reason"]},
                    "full_result": {"findings": result["findings"]},
                },
            )
        else:
            self.set_stub(self.review_client, {"result": result})

    def set_reviewer_failure(self, output: str) -> None:
        self.set_stub(self.review_client, {"exit_code": 1, "output": output})

    def set_stub(self, name: str, config: Mapping[str, object]) -> None:
        (self.stubs_dir / f"{name}.json").write_text(json.dumps(config))

    def calls(self, name: str | None = None) -> list[ReviewerCall]:
        calls_dir = self.stubs_dir / "calls"
        if not calls_dir.is_dir():
            return []
        calls: list[ReviewerCall] = []
        for path in calls_dir.iterdir():
            call: object = json.loads(path.read_text())
            if not is_reviewer_call(call):
                raise ValueError("Invalid reviewer call record")
            calls.append(call)
        calls.sort(key=lambda call: call["started_at"])
        return [call for call in calls if name is None or call["name"] == name]

    def check_calls(self) -> list[ReviewerCall]:
        return [
            call
            for call in self.calls()
            if call["name"] in ("golangci-lint", "pnpm", "npm", "mise")
        ]

    def reviewer_calls(self) -> list[ReviewerCall]:
        return [
            call
            for call in self.calls()
            if call["name"] == "claude" or (call["name"] == "codex" and "exec" in call["arguments"])
        ]

    def clear_calls(self) -> None:
        for path in (self.stubs_dir / "calls").glob("*.json"):
            path.unlink()

    def run_hook(self, action: str, payload: dict[str, object]) -> HookResponse:
        process = subprocess.run(
            [sys.executable, str(HOOK_PATH), action],
            cwd=self.root,
            env=self.environment,
            input=json.dumps({"cwd": str(self.root), **payload}),
            text=True,
            capture_output=True,
            check=True,
        )
        return get_hook_response(json.loads(process.stdout) if process.stdout.strip() else {})

    def run_doctor(self) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(HOOK_PATH), "doctor"],
            cwd=self.root,
            env=self.environment,
            stdin=subprocess.DEVNULL,
            text=True,
            capture_output=True,
        )

    # Runs the hook in this process, so a test can shrink the time limits.
    def stop_in_process(
        self,
        action: str,
        budget_seconds: float,
        reviewer_timeout_seconds: int,
        checks_timeout_seconds: float | None = None,
    ) -> HookResponse:
        payload = {
            "hook_event_name": "Stop",
            "session_id": "session-1",
            "last_assistant_message": "Done.",
            "cwd": str(self.root),
        }
        full_review = review_gate.FULL_REVIEWS[action]
        shrunk_review = replace(
            full_review,
            reviewer=replace(full_review.reviewer, timeout_seconds=reviewer_timeout_seconds),
        )
        stdout = io.StringIO()

        with (
            mock.patch.dict(os.environ, self.environment, clear=True),
            mock.patch.dict(review_gate.FULL_REVIEWS, {action: shrunk_review}),
            mock.patch.object(review_gate, "FULL_REVIEW_BUDGET_SECONDS", budget_seconds),
            mock.patch.object(
                review_gate,
                "CHECKS_TOTAL_TIMEOUT_SECONDS",
                checks_timeout_seconds or review_gate.CHECKS_TOTAL_TIMEOUT_SECONDS,
            ),
            mock.patch.object(sys, "argv", ["review_gate.py", action]),
            mock.patch.object(sys, "stdin", io.StringIO(json.dumps(payload))),
            mock.patch.object(sys, "stdout", stdout),
        ):
            review_gate.main()

        return get_hook_response(json.loads(stdout.getvalue()))

    def submit_prompt(self, prompt: str, session_id: str = "session-1") -> None:
        self.run_hook(
            "snapshot",
            {"hook_event_name": "UserPromptSubmit", "session_id": session_id, "prompt": prompt},
        )

    def stop(
        self, action: str, session_id: str = "session-1", message: str = "Done."
    ) -> HookResponse:
        payload = {
            "hook_event_name": "Stop",
            "session_id": session_id,
            "last_assistant_message": message,
        }
        return self.run_hook(action, payload)


def finding(category: str, summary: str, file: str = "src/widgets.py:10") -> Finding:
    return {"category": category, "file": file, "summary": summary}


def verification_section(*lines: str, fence: str = "sh") -> str:
    body = "".join(f"{line}\n" for line in lines)
    return f"\n## Change verification\n\n```{fence}\n{body}```\n"


def is_process_gone(pid: int) -> bool:
    try:
        os.kill(pid, 0)
    except ProcessLookupError:
        return True
    try:
        return Path(f"/proc/{pid}/stat").read_text().split(")")[-1].split()[0] == "Z"
    except OSError:
        return True


def wait_until_process_gone(pid: int, timeout_seconds: float = 5) -> bool:
    deadline = time.monotonic() + timeout_seconds
    while time.monotonic() < deadline:
        if is_process_gone(pid):
            return True
        time.sleep(0.05)
    return False


# With these limits a wave of reviewers that sleep 5.5 seconds fits the budget twice but not a third time,
# with about 2 seconds of margin on each side for the hook's own setup.
THIRD_WAVE_BUDGET_SECONDS = 14
THIRD_WAVE_REVIEWER_TIMEOUT_SECONDS = 6
THIRD_WAVE_REVIEWER_SECONDS = 5.5
# These keep the production relationship, budget >= checks total + 2 x reviewer limit, while the check
# and two reviewer waves exceed one reviewer deadline. Two seconds per subprocess allow startup under host contention.
TWO_WAVES_CHECKS_TIMEOUT_SECONDS = 4
TWO_WAVES_REVIEWER_TIMEOUT_SECONDS = 6
TWO_WAVES_BUDGET_SECONDS = (
    TWO_WAVES_CHECKS_TIMEOUT_SECONDS + 2 * TWO_WAVES_REVIEWER_TIMEOUT_SECONDS + 2
)


class ShortReplyAssertions(unittest.TestCase):
    def assert_asks_for_short_reply(self, reason: str) -> None:
        self.assertIn(
            "one short line about your fixes that names every rejected finding with the reason",
            reason,
        )
        self.assertIn("Do not write the summary for the user yet", reason)


class SummaryAssertions(unittest.TestCase):
    def assert_asks_for_summary_without_review(self, reason: str) -> None:
        self.assertIn("as if there had been no review", reason)
        self.assertIn("Leave out review rounds, findings and the fixes made for them.", reason)
        self.assertNotIn("each round", reason)

    def assert_accepts_change_and_asks_for_summary(self, response: HookResponse) -> None:
        self.assertEqual(response["decision"], "block")
        self.assertIn("Full implementation review accepted the change.", response["reason"])
        self.assertIn(
            "the automatic review passed and accepted the change in its current state",
            response["reason"],
        )
        self.assertIn("End with a short list of what the review made you fix", response["reason"])


class SessionStartTest(unittest.TestCase):
    def test_tells_agent_to_load_communication_skill_on_start_and_compaction(self) -> None:
        for client in ("codex", "claude"):
            for source in ("startup", "compact"):
                with self.subTest(client=client, source=source):
                    repository = HookRepository(self)
                    repository.environment["AGENT_REVIEW_CLIENT"] = client
                    response = repository.run_hook(
                        "session-start",
                        {
                            "hook_event_name": "SessionStart",
                            "session_id": "session-1",
                            "source": source,
                        },
                    )
                    self.assertIn(
                        ".agents/skills/humanizer/SKILL.md",
                        response["hookSpecificOutput"]["additionalContext"],
                    )


class ReviewClientSelectionTest(unittest.TestCase):
    def test_resolves_client_through_session_start_and_snapshot(self) -> None:
        for action in ("session-start", "snapshot"):
            for inherited, explicit, is_valid in (
                (None, None, False),
                ("invalid", None, False),
                ("claude", None, True),
                ("codex", None, True),
                ("invalid", "codex", True),
                ("codex", "claude", True),
                ("claude", "invalid", False),
            ):
                with self.subTest(action=action, inherited=inherited, explicit=explicit):
                    repository = HookRepository(self)
                    repository.environment.pop("AGENT_REVIEW_CLIENT", None)
                    if inherited is not None:
                        repository.environment["AGENT_REVIEW_CLIENT"] = inherited

                    arguments = [sys.executable, str(HOOK_PATH), action]
                    if explicit is not None:
                        arguments.extend(["--client", explicit])

                    process = subprocess.run(
                        arguments,
                        cwd=repository.root,
                        env=repository.environment,
                        input=json.dumps({"session_id": "selection"}),
                        text=True,
                        capture_output=True,
                    )

                    self.assertEqual(process.returncode, 0 if is_valid else 2, process.stderr)
                    if not is_valid:
                        self.assertIn("--client codex or --client claude", process.stderr)
                    self.assertEqual(repository.reviewer_calls(), [])


class Launcher:
    def __init__(self, test_case: unittest.TestCase) -> None:
        temporary_dir = tempfile.TemporaryDirectory(prefix="review-gate-launcher-test-")
        test_case.addCleanup(temporary_dir.cleanup)
        base = Path(temporary_dir.name)

        self.home = base / "home"
        self.stubs_dir = base / "stubs"
        self.bin_dir = base / "bin"
        for directory in (self.home, self.stubs_dir, self.bin_dir):
            directory.mkdir()

        # PATH holds only what the launcher needs before mise, so a mise installed on this machine stays out of reach.
        for name in LAUNCHER_COMMANDS:
            executable = shutil.which(name)
            assert executable is not None
            (self.bin_dir / name).symlink_to(executable)

        self.environment = {
            "PATH": str(self.bin_dir),
            "HOME": str(self.home),
            "REVIEW_GATE_STUBS": str(self.stubs_dir),
        }
        self.mise_answers: dict[str, dict[str, str | int]] = {}

    def install_mise(self, directory: Path) -> None:
        directory.mkdir(parents=True, exist_ok=True)
        stub = directory / "mise"
        stub.write_text(f"#!{sys.executable}\n{MISE_STUB_SOURCE}")
        stub.chmod(0o755)

    def set_mise_answer(self, subcommand: str, **answer: str | int) -> None:
        self.mise_answers[subcommand] = answer
        (self.stubs_dir / "mise.json").write_text(json.dumps(self.mise_answers))

    def mise_calls(self, subcommand: str) -> list[MiseCall]:
        calls_path = self.stubs_dir / "mise-calls.json"
        calls = json.loads(calls_path.read_text()) if calls_path.exists() else []
        if not isinstance(calls, list) or not all(is_mise_call(call) for call in calls):
            raise ValueError("Invalid mise call record")
        return [call for call in calls if is_mise_call(call) and call["subcommand"] == subcommand]

    def run(self, *arguments: str, stdin: str = "") -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [str(LAUNCHER_PATH), *arguments],
            env=self.environment,
            input=stdin,
            text=True,
            capture_output=True,
        )


MISSING_OPENSPEC_LINE = "npm:@fission-ai/openspec  1.11.0 (missing)  mise.toml  1.11.0"


class LauncherTest(unittest.TestCase):
    def assert_stops_agent(
        self, process: subprocess.CompletedProcess[str], *fragments: str
    ) -> HookResponse:
        self.assertEqual(process.returncode, 0)
        response = get_hook_response(json.loads(process.stdout))
        self.assertIs(response["continue"], False)
        for fragment in fragments:
            self.assertIn(fragment, response["stopReason"])
            self.assertIn(fragment, response["systemMessage"])
        return response

    def test_runs_hook_through_mise_from_home_local_bin_without_installs_when_path_has_no_mise(
        self,
    ) -> None:
        launcher = Launcher(self)
        launcher.install_mise(launcher.home / ".local" / "bin")
        payload = json.dumps({"hook_event_name": "Stop", "session_id": "session-1"})

        process = launcher.run("code-shape", stdin=payload)

        self.assertEqual(process.returncode, 0)
        self.assertEqual(json.loads(process.stdout), {})
        root = str(HOOK_PATH.parent.parent)
        [missing_check] = launcher.mise_calls("ls")
        self.assertEqual(
            missing_check["arguments"],
            [
                "-C",
                root,
                "ls",
                "--current",
                "--missing",
                "--no-truncate",
                "python",
                "npm:@fission-ai/openspec",
            ],
        )
        [call] = launcher.mise_calls("exec")
        self.assertEqual(
            call["arguments"], ["-C", root, "exec", "--", "python3", str(HOOK_PATH), "code-shape"]
        )
        self.assertEqual(call["exec_auto_install"], "false")
        self.assertEqual(call["stdin"], payload)

    def test_stops_agent_without_running_hook_when_pinned_version_is_missing(self) -> None:
        launcher = Launcher(self)
        launcher.install_mise(launcher.bin_dir)
        launcher.set_mise_answer("ls", stdout=f"{MISSING_OPENSPEC_LINE}\n")

        process = launcher.run("proposal")

        self.assert_stops_agent(
            process, "not installed", MISSING_OPENSPEC_LINE, "mise install", "mise run doctor"
        )
        self.assertEqual(launcher.mise_calls("exec"), [])

    def test_stops_agent_without_running_hook_when_missing_tools_check_fails(self) -> None:
        launcher = Launcher(self)
        launcher.install_mise(launcher.bin_dir)
        launcher.set_mise_answer(
            "ls", stdout="", stderr="mise ERROR Config file mise.toml is not trusted\n", exit_code=1
        )

        process = launcher.run("diff")

        self.assert_stops_agent(
            process, "mise exited with code 1", "is not trusted", "mise run doctor"
        )
        self.assertEqual(launcher.mise_calls("exec"), [])

    def test_stops_agent_when_version_manager_is_missing(self) -> None:
        launcher = Launcher(self)

        process = launcher.run("diff")

        self.assert_stops_agent(
            process,
            "could not find the version manager mise",
            "Install mise first.",
            "mise run doctor",
        )

    def test_stops_agent_with_action_and_exit_code_when_hook_crashes(self) -> None:
        launcher = Launcher(self)
        launcher.install_mise(launcher.bin_dir)
        launcher.set_mise_answer("exec", stdout="", stderr="Traceback: KeyError\n", exit_code=3)

        process = launcher.run("implementation")

        self.assert_stops_agent(process, "`implementation`", "exit code 3", "mise run doctor")

    def test_keeps_stop_message_valid_json_when_error_output_has_quotes_and_control_characters(
        self,
    ) -> None:
        launcher = Launcher(self)
        launcher.install_mise(launcher.bin_dir)
        launcher.set_mise_answer(
            "exec", stdout="", stderr='KeyError: "baseline"\\ \t\x1b[31m\r\n', exit_code=1
        )

        process = launcher.run("diff")

        response = self.assert_stops_agent(process)
        self.assertIn('KeyError: "baseline"\\', response["stopReason"])

    def test_doctor_exits_with_code_of_environment_check(self) -> None:
        launcher = Launcher(self)
        launcher.install_mise(launcher.bin_dir)
        launcher.set_mise_answer("exec", stdout="openspec: not found\n", exit_code=1)

        process = launcher.run("doctor")

        self.assertEqual(process.returncode, 1)
        self.assertEqual(process.stdout, "openspec: not found\n")
        [call] = launcher.mise_calls("exec")
        self.assertEqual(call["arguments"][-2:], [str(HOOK_PATH), "doctor"])

    def test_doctor_exits_with_code_1_when_pinned_version_is_missing(self) -> None:
        launcher = Launcher(self)
        launcher.install_mise(launcher.bin_dir)
        launcher.set_mise_answer("ls", stdout=f"{MISSING_OPENSPEC_LINE}\n")

        process = launcher.run("doctor")

        self.assertEqual(process.returncode, 1)
        self.assertIn(MISSING_OPENSPEC_LINE, process.stderr)
        self.assertEqual(launcher.mise_calls("exec"), [])

    def test_review_passes_change_and_exits_with_code_of_manual_review(self) -> None:
        launcher = Launcher(self)
        launcher.install_mise(launcher.bin_dir)
        launcher.set_mise_answer(
            "exec", stdout="Result: the review did not accept the change.\n", exit_code=1
        )

        process = launcher.run("review", "add-widgets")

        self.assertEqual(process.returncode, 1)
        self.assertEqual(process.stdout, "Result: the review did not accept the change.\n")
        [call] = launcher.mise_calls("exec")
        self.assertEqual(call["arguments"][-3:], [str(HOOK_PATH), "review", "add-widgets"])

    def test_review_exits_with_code_1_when_version_manager_is_missing(self) -> None:
        launcher = Launcher(self)

        process = launcher.run("review")

        self.assertEqual(process.returncode, 1)
        self.assertIn("could not find the version manager mise", process.stderr)

    def test_doctor_exits_with_code_1_when_version_manager_is_missing(self) -> None:
        launcher = Launcher(self)

        process = launcher.run("doctor")

        self.assertEqual(process.returncode, 1)
        self.assertIn("could not find the version manager mise", process.stderr)


class MissingToolTest(unittest.TestCase):
    def test_stops_agent_naming_openspec_and_doctor_when_openspec_is_missing(self) -> None:
        repository = HookRepository(self, missing_tools=("openspec",))
        repository.submit_prompt("Write a spec for gadgets")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertIs(response["continue"], False)
        self.assertIn("could not find `openspec`", response["stopReason"])
        self.assertIn("mise run doctor", response["stopReason"])
        self.assertIn("mise run doctor", response["systemMessage"])
        self.assertEqual(repository.reviewer_calls(), [])


class DoctorTest(unittest.TestCase):
    def test_reports_path_of_every_tool_and_exits_with_code_0(self) -> None:
        repository = HookRepository(self, is_path_isolated=True)
        bin_dir = repository.root.parent / "bin"

        process = repository.run_doctor()

        self.assertEqual(process.returncode, 0, process.stdout)
        self.assertEqual(
            process.stdout.splitlines(),
            [
                "Review client: claude",
                f"python: {sys.executable}",
                f"git: {repository.root.parent / 'system-bin' / 'git'}",
                f"openspec: {bin_dir / 'openspec'}",
                f"claude: {bin_dir / 'claude'}",
            ],
        )

    def test_reports_missing_reviewer_cli_and_exits_with_nonzero_code(self) -> None:
        repository = HookRepository(self, missing_tools=("claude",))

        process = repository.run_doctor()

        self.assertNotEqual(process.returncode, 0)
        self.assertIn("claude: not found", process.stdout.splitlines())


class RunChecksTest(unittest.TestCase):
    def setUp(self) -> None:
        temporary_dir = tempfile.TemporaryDirectory(prefix="review-gate-checks-test-")
        self.addCleanup(temporary_dir.cleanup)
        self.root = Path(temporary_dir.name)

    def read_pid(self, name: str) -> int:
        return int((self.root / name).read_text())

    def test_stops_check_with_every_child_process_when_check_exceeds_its_limit(self) -> None:
        check = review_gate.Check(".", (sys.executable, "-c", CHECK_WITH_CHILD_SOURCE, "child.pid"))

        failure = review_gate.run_checks(
            self.root, [check], check_timeout_seconds=1, total_timeout_seconds=10
        )

        assert failure is not None
        self.assertIn("did not finish within its limit of 1 seconds per check", failure)
        self.assertTrue(wait_until_process_gone(self.read_pid("child.pid")))

    def test_stops_running_check_and_names_total_limit_when_checks_together_exceed_it(self) -> None:
        checks = [
            review_gate.Check(".", (sys.executable, "-c", "import time; time.sleep(1.5)")),
            review_gate.Check(".", (sys.executable, "-c", CHECK_WITH_CHILD_SOURCE, "child.pid")),
        ]

        failure = review_gate.run_checks(
            self.root, checks, check_timeout_seconds=2, total_timeout_seconds=3
        )

        assert failure is not None
        self.assertIn("did not finish within the total limit of 3 seconds for all checks", failure)
        self.assertIn("child.wait()", failure)
        self.assertTrue(wait_until_process_gone(self.read_pid("child.pid")))

    def test_names_exceeded_limit_when_child_outside_process_group_keeps_output_open(self) -> None:
        check = review_gate.Check(
            ".", (sys.executable, "-c", CHECK_WITH_ESCAPED_CHILD_SOURCE, "child.pid")
        )
        self.addCleanup(
            lambda: (
                is_process_gone(self.read_pid("child.pid"))
                or os.kill(self.read_pid("child.pid"), signal.SIGKILL)
            )
        )

        with mock.patch.object(review_gate, "KILLED_CHECK_OUTPUT_TIMEOUT_SECONDS", 0.5):
            failure = review_gate.run_checks(
                self.root, [check], check_timeout_seconds=1, total_timeout_seconds=10
            )

        assert failure is not None
        self.assertIn("did not finish within its limit of 1 seconds per check", failure)
        self.assertIn("End of output:\n(no output)", failure)


def create_configured_repository(test_case: unittest.TestCase) -> HookRepository:
    repository = HookRepository(test_case)
    shutil.copytree(
        HOOK_PATH.parent,
        repository.root / ".agent-hooks",
        ignore=shutil.ignore_patterns("__pycache__"),
    )
    repository.git("add", ".agent-hooks")
    repository.git("commit", "--quiet", "-m", "Add hook configuration fixture")

    repository.environment.pop("CLAUDE_PROJECT_DIR", None)
    mise_stub = Path(repository.environment["PATH"].split(os.pathsep)[0]) / "mise"
    mise_stub.write_text(
        f"#!{sys.executable}\n"
        "import os, sys\n"
        "if sys.argv[3] == 'exec':\n"
        "    os.execv(sys.executable, [sys.executable, *sys.argv[6:]])\n"
    )

    return repository


class HookConfigurationTest(unittest.TestCase):
    def setUp(self) -> None:
        root = HOOK_PATH.parent.parent
        self.claude_config = json.loads((root / ".claude/settings.json").read_text())
        self.claude_hooks = get_hook_configuration(self.claude_config["hooks"])
        self.codex_hooks = get_hook_configuration(
            json.loads((root / ".codex/hooks.json").read_text())["hooks"]
        )

    def get_hooks(
        self, configuration: dict[str, list[HookGroup]], event: str
    ) -> list[HookDefinition]:
        return [hook for group in configuration.get(event, []) for hook in group["hooks"]]

    def test_matches_actions_timeouts_and_status_messages_when_both_clients_finish_turns(
        self,
    ) -> None:
        self.assertEqual(set(self.codex_hooks), {"SessionStart", "UserPromptSubmit", "Stop"})
        for event in self.codex_hooks:
            claude_hooks = self.get_hooks(self.claude_hooks, event)
            codex_hooks = self.get_hooks(self.codex_hooks, event)

            self.assertEqual(
                [shlex.split(hook["command"])[-3] for hook in claude_hooks],
                [shlex.split(hook["command"])[-3] for hook in codex_hooks],
                event,
            )

            for claude_hook, codex_hook in zip(claude_hooks, codex_hooks, strict=True):
                action = shlex.split(claude_hook["command"])[-3]
                with self.subTest(event=event, action=action):
                    self.assertEqual(
                        codex_hook["command"],
                        f'"$(git rev-parse --show-toplevel)/.agent-hooks/run.sh" {action} --client codex',
                    )
                    self.assertEqual(
                        {key: value for key, value in claude_hook.items() if key != "command"},
                        {key: value for key, value in codex_hook.items() if key != "command"},
                    )

    def test_intercepts_plans_only_when_client_is_claude_code(self) -> None:
        [plan_group] = self.claude_hooks["PreToolUse"]
        self.assertEqual(plan_group["matcher"], "ExitPlanMode")
        self.assertEqual(
            [shlex.split(hook["command"])[-3] for hook in plan_group["hooks"]],
            ["plan", "necessity", "architecture"],
        )
        self.assertNotIn("PreToolUse", self.codex_hooks)
        self.assertNotIn(
            "plan",
            [shlex.split(hook["command"])[-3] for hook in self.get_hooks(self.codex_hooks, "Stop")],
        )

    def test_matches_browser_launcher_without_a_helper_tag_in_project_settings(self) -> None:
        root = HOOK_PATH.parent.parent
        codex_config = tomllib.loads((root / ".codex/config.toml").read_text())
        browser_config = json.loads((root / ".mcp.json").read_text())["mcpServers"][
            "chrome-devtools"
        ]

        self.assertEqual(codex_config["mcp_servers"]["chrome-devtools"], browser_config)
        self.assertNotIn("GOFLAGS", codex_config["shell_environment_policy"]["set"])
        self.assertNotIn("GOFLAGS", self.claude_config["env"])
        self.assertEqual(
            codex_config["shell_environment_policy"]["set"]["AGENT_REVIEW_CLIENT"], "codex"
        )
        self.assertEqual(self.claude_config["env"]["AGENT_REVIEW_CLIENT"], "claude")

    def create_repository(self) -> HookRepository:
        return create_configured_repository(self)

    def run_client_hook(
        self,
        repository: HookRepository,
        action: str,
        payload: dict[str, object],
        client: str = "codex",
        event_override: str | None = None,
    ) -> HookResponse | None:
        event = event_override or {
            "session-start": "SessionStart",
            "snapshot": "UserPromptSubmit",
        }.get(action, "Stop")
        configuration = self.codex_hooks if client == "codex" else self.claude_hooks
        hook = next(
            hook
            for hook in self.get_hooks(configuration, event)
            if hook["command"].endswith(f" {action} --client {client}")
        )

        repository.environment["CLAUDE_PROJECT_DIR"] = str(repository.root)
        subdirectory = repository.root / "nested folder"
        subdirectory.mkdir(exist_ok=True)

        process = subprocess.run(
            ["bash", "-c", hook["command"]],
            cwd=subdirectory,
            env=repository.environment,
            input=json.dumps(
                {
                    "hook_event_name": event,
                    "session_id": "codex-session",
                    "cwd": str(subdirectory),
                    **payload,
                }
            ),
            text=True,
            capture_output=True,
            check=True,
        )

        return get_hook_response(json.loads(process.stdout)) if process.stdout.strip() else None

    def test_runs_every_tracked_hook_from_nested_directory_with_inherited_other_client(
        self,
    ) -> None:
        for client, configuration in (("codex", self.codex_hooks), ("claude", self.claude_hooks)):
            repository = self.create_repository()
            repository.environment["AGENT_REVIEW_CLIENT"] = (
                "claude" if client == "codex" else "codex"
            )
            for event in ("SessionStart", "UserPromptSubmit", "Stop", "PreToolUse"):
                for hook in self.get_hooks(configuration, event):
                    action = shlex.split(hook["command"])[-3]
                    with self.subTest(client=client, event=event, action=action):
                        response = self.run_client_hook(
                            repository,
                            action,
                            {
                                "prompt": "Read the repository",
                                "tool_input": {"plan": "Inspect the repository"},
                            },
                            client,
                            event,
                        )
                        if action == "session-start":
                            assert response is not None
                            self.assertIn(
                                ".agents/skills/humanizer/SKILL.md",
                                response["hookSpecificOutput"]["additionalContext"],
                            )
                        elif action == "snapshot":
                            self.assertIsNone(response)
                        else:
                            self.assertEqual(response, {})

            self.assertTrue(
                (repository.root / ".git/agent-review-hooks/codex-session.json").is_file()
            )

    def test_runs_codex_reviewers_with_required_effort_when_codex_finishes_changes(self) -> None:
        for gate, effort, reviewer_count in (("diff", "medium", 1), ("implementation", "high", 4)):
            with self.subTest(gate=gate):
                repository = self.create_repository()
                self.run_client_hook(
                    repository, "snapshot", {"prompt": "Continue implementing widgets"}
                )
                repository.write("src/widgets.py", "WIDGETS = []\n")
                if gate == "implementation":
                    repository.complete_tasks()

                response = self.run_client_hook(
                    repository, gate, {"last_assistant_message": "Done."}
                )

                if gate == "implementation":
                    assert response is not None
                    self.assertIn(
                        "Full implementation review accepted the change.", response["reason"]
                    )
                else:
                    self.assertEqual(response, {})
                reviewer_calls = repository.reviewer_calls()
                self.assertEqual(len(reviewer_calls), reviewer_count)
                for reviewer_call in reviewer_calls:
                    self.assertEqual(reviewer_call["name"], "codex")
                    self.assertIn(f'model_reasoning_effort="{effort}"', reviewer_call["arguments"])

    def test_preserves_user_request_baseline_when_codex_submits_automatic_review_continuation(
        self,
    ) -> None:
        repository = self.create_repository()
        repository.write("existing.md", "Existing user edit\n")
        self.run_client_hook(repository, "snapshot", {"prompt": "Add widgets"})

        request_path = repository.root / ".git/agent-review-hooks/codex-session.json"
        original_request = json.loads(request_path.read_text())

        repository.write("src/widgets.py", "WIDGETS = []\n")
        self.run_client_hook(
            repository,
            "snapshot",
            {"prompt": f"{review_gate.AUTOMATIC_REVIEW_PREFIX} Fix the widgets"},
        )

        self.assertEqual(json.loads(request_path.read_text()), original_request)
        response = self.run_client_hook(repository, "diff", {"last_assistant_message": "Fixed."})

        self.assertEqual(response, {})
        [reviewer_call] = repository.reviewer_calls()
        self.assertIn("src/widgets.py", reviewer_call["prompt"])
        self.assertIn("Add widgets", reviewer_call["prompt"])


class FullReviewBudgetTest(unittest.TestCase):
    def get_hook_timeouts(self, gate: str, config_path: str) -> list[int]:
        config = json.loads((HOOK_PATH.parent.parent / config_path).read_text())
        return [
            hook["timeout"]
            for groups in config["hooks"].values()
            for group in groups
            for hook in group["hooks"]
            if shlex.split(hook["command"])[-3] == gate
        ]

    def test_budget_fits_checks_and_two_reviewer_waves(self) -> None:
        for full_review in review_gate.FULL_REVIEWS.values():
            self.assertGreaterEqual(
                review_gate.FULL_REVIEW_BUDGET_SECONDS,
                review_gate.CHECKS_TOTAL_TIMEOUT_SECONDS + 2 * full_review.reviewer.timeout_seconds,
            )

    def test_budget_ends_before_full_review_hook_timeouts(self) -> None:
        for config_path in (".claude/settings.json", ".codex/hooks.json"):
            for gate in review_gate.FULL_REVIEWS:
                with self.subTest(configuration=config_path, gate=gate):
                    [timeout_seconds] = self.get_hook_timeouts(gate, config_path)
                    self.assertLess(review_gate.FULL_REVIEW_BUDGET_SECONDS, timeout_seconds, gate)


class ChangeVerificationParsingTest(unittest.TestCase):
    def setUp(self) -> None:
        temporary_dir = tempfile.TemporaryDirectory(prefix="review-gate-verification-test-")
        self.addCleanup(temporary_dir.cleanup)
        self.change_dir = Path(temporary_dir.name) / CHANGE_NAME
        self.change_dir.mkdir()

    def parse(self, section: str) -> list[review_gate.Check]:
        (self.change_dir / "tasks.md").write_text(f"# Tasks\n\n- [x] 1.1 Build\n{section}")
        return review_gate.parse_change_verification(self.change_dir)

    def assert_rejected(self, section: str, reason: str) -> None:
        with self.assertRaises(review_gate.ChangeVerificationError) as context:
            self.parse(section)
        message = str(context.exception)
        self.assertIn(reason, message)
        self.assertIn(f"`{CHANGE_NAME}/tasks.md`", message)
        self.assertIn(review_gate.ALLOWED_VERIFICATION_FORMS, message)

    def test_accepts_the_full_run_without_execution(self) -> None:
        with mock.patch.object(
            review_gate.subprocess, "run", side_effect=AssertionError("must not run")
        ):
            checks = self.parse(verification_section("mise run test"))
        self.assertEqual(checks, [review_gate.Check(".", ("mise", "run", "test"))])

    def test_rejects_removed_narrow_commands(self) -> None:
        commands = (
            "mise run backend:test-package ./internal/features/intervals/...",
            "mise run backend:test-group physical-api-1-of-3",
            "mise run agent:test",
            "pnpm --dir frontend test",
            "mise run test-filesystem",
        )
        for command in commands:
            with self.subTest(command=command):
                self.assert_rejected(
                    verification_section(command), f"`{command}` is not an allowed command"
                )

    def test_rejects_old_just_form(self) -> None:
        self.assert_rejected(
            verification_section("just test-package ./internal/features/intervals/..."),
            "`just test-package ./internal/features/intervals/...` is not an allowed command",
        )

    def test_rejects_shell_operator_after_the_full_run(self) -> None:
        self.assert_rejected(
            verification_section("mise run test && rm -rf /tmp/x"),
            "`mise run test && rm -rf /tmp/x` does not match the form `mise run test`",
        )

    def test_rejects_arguments_after_the_full_run(self) -> None:
        self.assert_rejected(
            verification_section("mise run test ./x"), "does not match the form `mise run test`"
        )

    def test_rejects_shell_substitution_in_the_full_run(self) -> None:
        self.assert_rejected(
            verification_section("mise run test;rm"), "`mise run test;rm` is not an allowed command"
        )

    def test_accepts_exact_isolated_browser_form_without_execution(self) -> None:
        with mock.patch.object(
            review_gate.subprocess, "run", side_effect=AssertionError("must not run")
        ):
            checks = self.parse(
                verification_section("pnpm --dir frontend test:browser --project=ui")
            )
        self.assertEqual(
            checks,
            [review_gate.Check(".", ("pnpm", "--dir", "frontend", "test:browser", "--project=ui"))],
        )

    def test_accepts_exact_browser_suite_form_without_execution(self) -> None:
        with mock.patch.object(
            review_gate.subprocess, "run", side_effect=AssertionError("must not run")
        ):
            checks = self.parse(verification_section("mise run e2e:browser"))
        self.assertEqual(checks, [review_gate.Check(".", ("mise", "run", "e2e:browser"))])

    def test_rejects_unbounded_browser_forms_and_shell_arguments(self) -> None:
        commands = (
            "pnpm --dir frontend test:browser",
            "pnpm --dir frontend test:browser --project=other",
            "pnpm --dir frontend test:browser --project ui",
            "pnpm --dir frontend test:browser --project=ui --headed",
            "pnpm --dir frontend test:browser --project=e2e",
            "mise run e2e",
            "mise run test-full",
            "mise run e2e:browser --headed",
            "mise run e2e:browser;touch",
            "mise run e2e:browser ./x",
            "pnpm --dir frontend arbitrary-script --project=ui",
            "pnpm --dir frontend test:browser --project=ui;touch",
            "pnpm --dir frontend test:browser --project=ui && touch /tmp/x",
            "pnpm --dir frontend test:browser --project=$(touch)",
        )
        for command in commands:
            with self.subTest(command=command):
                with self.assertRaises(review_gate.ChangeVerificationError):
                    self.parse(verification_section(command))

    def test_accepts_every_allowed_form_and_skips_empty_and_comment_lines(self) -> None:
        checks = self.parse(
            verification_section(
                "# everything",
                "mise run test",
                "",
                "pnpm --dir frontend test:browser --project=ui",
                "mise run e2e:browser",
            )
        )

        self.assertEqual(
            checks,
            [
                review_gate.Check(".", ("mise", "run", "test")),
                review_gate.Check(
                    ".", ("pnpm", "--dir", "frontend", "test:browser", "--project=ui")
                ),
                review_gate.Check(".", ("mise", "run", "e2e:browser")),
            ],
        )

    def test_rejects_section_with_only_bash_fenced_block(self) -> None:
        self.assert_rejected(
            verification_section("mise run test", fence="bash"),
            "the section has no ```sh code block",
        )

    def test_rejects_unclosed_block(self) -> None:
        self.assert_rejected(
            "\n## Change verification\n\n```sh\nmise run test\n",
            "the ```sh code block is not closed",
        )

    def test_rejects_block_without_commands(self) -> None:
        self.assert_rejected(
            verification_section("# nothing yet", ""), "the ```sh code block holds no command"
        )

    def test_returns_no_commands_without_section(self) -> None:
        self.assertEqual(self.parse(""), [])


class PlanGateTest(unittest.TestCase):
    def exit_plan_mode(self, repository: HookRepository, gate: str, plan: str) -> HookResponse:
        payload = {
            "hook_event_name": "PreToolUse",
            "session_id": "session-1",
            "tool_name": "ExitPlanMode",
            "tool_input": {"plan": plan},
        }
        return repository.run_hook(gate, payload)

    def test_allows_plan_when_reviewer_approves(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Plan the widgets")

        response = self.exit_plan_mode(repository, "plan", "# Plan\n\nAdd widgets.\n")

        self.assertEqual(response, {})
        [call] = repository.reviewer_calls()
        self.assertIn("## Plan review", call["prompt"])
        self.assertIn("Add widgets.", call["prompt"])

    def test_denies_plan_with_reviewer_reason_when_reviewer_rejects(self) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result(
            {"ok": False, "reason": "The plan keeps a compatibility alias."}
        )
        repository.submit_prompt("Plan the widgets")

        response = self.exit_plan_mode(repository, "necessity", "# Plan\n")

        output = response["hookSpecificOutput"]
        self.assertEqual(output["permissionDecision"], "deny")
        self.assertIn("The plan keeps a compatibility alias.", output["permissionDecisionReason"])
        self.assertIn("Revise the plan and show it again", output["permissionDecisionReason"])

    def test_denies_exit_plan_mode_without_plan_text(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Plan the widgets")

        response = self.exit_plan_mode(repository, "architecture", "")

        self.assertEqual(response["hookSpecificOutput"]["permissionDecision"], "deny")
        self.assertEqual(repository.reviewer_calls(), [])


class DiffGateTest(unittest.TestCase):
    def test_accepts_turn_when_reviewer_approves(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        response = repository.stop("diff")

        self.assertEqual(response, {})
        self.assertEqual(len(repository.reviewer_calls()), 1)

    def test_blocks_turn_with_reviewer_reason_when_reviewer_rejects(self) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result({"ok": False, "reason": "No test for an empty list."})
        repository.submit_prompt("Add a widget")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        response = repository.stop("diff")

        self.assertEqual(response["decision"], "block")
        self.assertIn("No test for an empty list.", response["reason"])
        self.assertIn(
            "a summary of the final state, without retelling this review", response["reason"]
        )
        self.assertNotIn("Fix the result", response["reason"])

    def test_diff_reviewer_prompt_checks_agent_instructions(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Fix the docs")
        repository.write("AGENTS.md", "# AGENTS.md\n")

        repository.stop("diff")

        [call] = repository.reviewer_calls()
        self.assertIn("Check edits to agent instructions per the same section.", call["prompt"])

    def test_reviewer_prompt_maps_paths_to_module_docs(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        repository.stop("diff")

        [call] = repository.reviewer_calls()
        self.assertNotIn("| `agent/**` |", call["prompt"])
        self.assertIn("| `backend/**` | `backend/AGENTS.md` |", call["prompt"])

    def test_runs_selected_quick_profile(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        repository.stop("diff")

        [call] = repository.reviewer_calls()
        self.assertEqual(call["name"], repository.review_client)
        self.assertEqual(get_effort(call), repository.quick_effort)
        self.assertEqual(call["effort_env"], repository.quick_effort)

    def test_stops_agent_after_revision_limit(self) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result({"ok": False, "reason": "No test for an empty list."})
        repository.submit_prompt("Add a widget")

        responses = []
        for attempt in range(review_gate.MAX_REVISIONS + 1):
            repository.write("src/widgets.py", f"WIDGETS = [{attempt}]\n")
            responses.append(repository.stop("diff"))

        self.assertEqual(
            [response.get("decision") for response in responses[:-1]],
            ["block"] * review_gate.MAX_REVISIONS,
        )
        self.assertIs(responses[-1]["continue"], False)


class ArchitectureGateTest(unittest.TestCase):
    def test_architecture_prompt_reads_authoritative_frontend_architecture(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Improve a screen")
        repository.write("frontend/src/pages/Settings.tsx", "export {};\n")

        repository.stop("architecture")

        [call] = repository.reviewer_calls()
        self.assertIn('`docs/architecture.md`: "Frontend"', call["prompt"])
        self.assertIn("`frontend/AGENTS.md`: the applicable frontend conventions", call["prompt"])

    def test_architecture_prompt_checks_http_contract_against_api_design_skill(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        repository.stop("architecture")

        [call] = repository.reviewer_calls()
        self.assertIn(
            "When a plan adds or changes an HTTP operation, or a finished change edits a file under `backend/api/`, "
            "also read `.agents/skills/api-design/SKILL.md` and check the operation or the edit against it.",
            call["prompt"],
        )

    def test_architecture_prompt_leaves_split_inside_package_to_code_shape_check(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        repository.stop("architecture")

        [call] = repository.reviewer_calls()
        self.assertIn(
            "leave the split of code into files and functions inside a package to the code-shape check.",
            call["prompt"],
        )


class FullReviewSchemaTest(unittest.TestCase):
    def test_reviewer_schemas_allow_exactly_the_categories_of_their_full_review(self) -> None:
        for full_review in (review_gate.IMPLEMENTATION_REVIEW, review_gate.PROPOSAL_REVIEW):
            with self.subTest(full_review.gate):
                schema = json.loads(full_review.reviewer.schema_path.read_text())
                category_enum = schema["properties"]["findings"]["items"]["properties"]["category"][
                    "enum"
                ]
                self.assertEqual(tuple(category_enum), full_review.categories)


class CodeShapeGateTest(unittest.TestCase):
    def test_runs_reviewer_when_turn_changes_backend_go_file(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")

        response = repository.stop("code-shape")

        self.assertEqual(response, {})
        [call] = repository.reviewer_calls()
        self.assertIn("## Code shape review", call["prompt"])

    def test_runs_reviewer_when_turn_changes_agent_go_file(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Add a heartbeat field")
        repository.write("backend/internal/agent/heartbeat/heartbeat.go", "package heartbeat\n")

        repository.stop("code-shape")

        self.assertEqual(len(repository.reviewer_calls()), 1)

    def test_runs_reviewer_when_turn_changes_frontend_tsx_file(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("frontend/src/pages/widgets/ui/WidgetsPage.tsx", "export {};\n")

        repository.stop("code-shape")

        self.assertEqual(len(repository.reviewer_calls()), 1)

    def test_runs_reviewer_when_turn_changes_website_tsx_file(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Add a page")
        repository.write("website/app/[lang]/pricing/page.tsx", "export {};\n")

        repository.stop("code-shape")

        self.assertEqual(len(repository.reviewer_calls()), 1)

    def test_blocks_turn_with_reviewer_reason_when_reviewer_rejects(self) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result(
            {"ok": False, "reason": "Active() is a getter, name it GetRunningVerificationIDs()."}
        )
        repository.submit_prompt("Add a widget")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")

        response = repository.stop("code-shape")

        self.assertEqual(response["decision"], "block")
        self.assertIn("GetRunningVerificationIDs", response["reason"])

    def test_skips_turn_that_changes_only_markdown(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Fix the docs")
        repository.write("backend/AGENTS.md", "# AGENTS.md\n")

        response = repository.stop("code-shape")

        self.assertEqual(response, {})
        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_turn_that_changes_only_frontend_config(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Bump Vite")
        repository.write("frontend/vite.config.ts", "export default {};\n")

        repository.stop("code-shape")

        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_markdown_turn_when_go_file_was_changed_before_request(self) -> None:
        repository = HookRepository(self)
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")
        repository.submit_prompt("Fix the docs")
        repository.write("backend/AGENTS.md", "# AGENTS.md\n")

        repository.stop("code-shape")

        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_turn_that_changes_only_ready_proposal(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("code-shape")

        self.assertEqual(response, {})
        self.assertEqual(repository.reviewer_calls(), [])

    def test_runs_selected_quick_profile(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")

        repository.stop("code-shape")

        [call] = repository.reviewer_calls()
        self.assertEqual(call["name"], repository.review_client)
        self.assertEqual(get_effort(call), repository.quick_effort)

    def test_prompt_holds_code_to_logical_units_rule_and_reads_created_files_in_full(self) -> None:
        prompt = self.get_finished_turn_prompt()

        self.assertIn('"Code is split into logical units"', prompt)
        self.assertIn("Read every source file the turn created in full", prompt)

    def test_prompt_rejects_new_file_or_function_with_two_concepts_and_concept_added_to_other_file(
        self,
    ) -> None:
        prompt = self.get_finished_turn_prompt()

        self.assertIn(
            "Reject a new file or a new function that holds more than one concept, "
            "and a concept the turn added to an existing file that holds a different one.",
            prompt,
        )

    def test_prompt_leaves_placement_across_packages_to_architecture_check(self) -> None:
        prompt = self.get_finished_turn_prompt()

        self.assertIn(
            "Do not judge which package, module, layer or role a unit belongs to; the architecture check does that.",
            prompt,
        )

    def test_prompt_accepts_additions_to_files_whose_content_architecture_fixes(self) -> None:
        prompt = self.get_finished_turn_prompt()

        self.assertIn(
            "an event or a contract type in `module.go`, an error in `errors.go`, "
            "an enum next to its entity, a method in its role file.",
            prompt,
        )

    def test_prompt_does_not_judge_concepts_mixed_before_turn(self) -> None:
        prompt = self.get_finished_turn_prompt()

        self.assertIn(
            "Do not judge concepts that were already mixed in a file before the turn.", prompt
        )

    def test_prompt_treats_comment_that_restates_code_as_naming_problem(self) -> None:
        prompt = self.get_finished_turn_prompt()

        self.assertIn("A comment that restates what the code does is a naming problem", prompt)

    def get_finished_turn_prompt(self) -> str:
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")

        repository.stop("code-shape")

        [call] = repository.reviewer_calls()
        return call["prompt"]


class ImplementationGateStartTest(SummaryAssertions, unittest.TestCase):
    def test_runs_full_review_when_agent_completes_last_task(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assert_accepts_change_and_asks_for_summary(response)
        self.assertEqual(len(repository.reviewer_calls()), 4)

    def test_skips_review_when_tasks_remain_unchecked(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_review_when_workspace_is_unchanged_during_turn(self) -> None:
        repository = HookRepository(self)
        repository.complete_tasks()
        repository.submit_prompt("What is left?")

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_review_in_new_session_when_workspace_matches_accepted_state(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Continue", session_id="first")
        repository.submit_prompt("Look at the tasks", session_id="second")
        repository.complete_tasks()
        repository.stop("implementation", session_id="first")
        repository.clear_calls()

        response = repository.stop("implementation", session_id="second")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_runs_review_again_when_files_change_after_accepted_review(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.complete_tasks()
        repository.stop("implementation")
        repository.clear_calls()
        repository.submit_prompt("Rename it")
        repository.write("src/widgets.py", "WIDGETS = ()\n")

        repository.stop("implementation")

        self.assertEqual(len(repository.reviewer_calls()), 4)

    def test_skips_review_when_prompt_has_openspec_planning_command(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("/opsx:propose add widgets")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_archived_changes(self) -> None:
        repository = HookRepository(self)
        repository.git("rm", "-r", "--quiet", f"openspec/changes/{CHANGE_NAME}")
        repository.git("commit", "--quiet", "-m", "archive")
        repository.submit_prompt("Archive it")
        repository.write_change(
            spec_names=("widgets",),
            tasks="- [x] 1.1 Build\n",
            name=f"archive/2026-10-01-{CHANGE_NAME}",
        )

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_returns_empty_response_without_payload(self) -> None:
        repository = HookRepository(self)

        process = subprocess.run(
            [sys.executable, str(HOOK_PATH), "implementation"],
            cwd=repository.root,
            env=repository.environment,
            input="",
            text=True,
            capture_output=True,
        )

        self.assertEqual(process.returncode, 0)
        self.assertEqual(json.loads(process.stdout), {})


class ImplementationGateChecksTest(ShortReplyAssertions, SummaryAssertions, unittest.TestCase):
    def test_runs_no_verification_commands_when_implementation_review_starts(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")
        repository.set_stub("golangci-lint", {"exit_code": 1, "output": "must not run"})
        repository.complete_tasks(
            verification_section=verification_section(
                "mise run test", "pnpm --dir frontend test:browser --project=ui"
            )
        )

        response = repository.stop("implementation")

        self.assert_accepts_change_and_asks_for_summary(response)
        self.assertEqual(repository.check_calls(), [])
        self.assertEqual(len(repository.reviewer_calls()), 4)

    def test_starts_reviewers_without_checks_when_change_has_no_verification_section(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("frontend/src/pages/widgets/ui/WidgetsPage.tsx", "export {};\n")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assert_accepts_change_and_asks_for_summary(response)
        self.assertEqual(repository.check_calls(), [])
        self.assertEqual(len(repository.reviewer_calls()), 4)

    def test_runs_no_shared_verification_command_when_two_changes_name_it(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.complete_tasks(verification_section=verification_section("mise run test"))
        repository.write_change(
            spec_names=("gizmos",),
            tasks="- [x] 1.1 Build\n" + verification_section("mise run test"),
            name="add-gizmos",
        )

        response = repository.stop("implementation")

        self.assert_accepts_change_and_asks_for_summary(response)
        self.assertEqual(repository.check_calls(), [])
        self.assertEqual(len(repository.reviewer_calls()), 8)

    def test_blocks_turn_before_reviewers_when_verification_section_is_malformed(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.complete_tasks(verification_section=verification_section("mise run agent:test"))

        response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn("`mise run agent:test` is not an allowed command", response["reason"])
        self.assertIn(review_gate.ALLOWED_VERIFICATION_FORMS, response["reason"])
        self.assertEqual(repository.check_calls(), [])
        self.assertEqual(repository.reviewer_calls(), [])

    def test_runs_no_hook_tests_when_change_touches_hooks(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write(".agent-hooks/sample_test.py", FAILING_HOOK_TEST)
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assert_accepts_change_and_asks_for_summary(response)
        self.assertEqual(repository.check_calls(), [])
        self.assertEqual(len(repository.reviewer_calls()), 4)

    def test_does_not_claim_checks_passed_when_reviewer_prompt_is_built(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        repository.stop("implementation")

        for call in repository.reviewer_calls():
            self.assertNotIn("The hook already ran the linters", call["prompt"])
            self.assertIn("does not run linters, builds or tests", call["prompt"])


class ImplementationGateReviewTest(ShortReplyAssertions, SummaryAssertions, unittest.TestCase):
    def test_runs_selected_full_profile_per_spec_and_for_architecture_docs_and_tests(self) -> None:
        repository = HookRepository(self)
        repository.write_change(spec_names=("widgets", "gadgets"), tasks="- [x] 1.1 Build\n")
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        repository.stop("implementation")

        calls = repository.reviewer_calls()
        self.assertEqual(len(calls), 5)
        for call in calls:
            self.assertEqual(call["name"], repository.review_client)
            self.assertEqual(call["effort_env"], repository.full_effort)
        roles = sorted(
            line
            for call in calls
            for line in call["prompt"].splitlines()
            if line.startswith("Role:")
        )
        self.assertEqual(
            roles,
            [
                "Role: architecture",
                f"Role: delta spec `openspec/changes/{CHANGE_NAME}/specs/gadgets/spec.md`",
                f"Role: delta spec `openspec/changes/{CHANGE_NAME}/specs/widgets/spec.md`",
                "Role: documentation and tasks",
                "Role: tests",
            ],
        )

    def test_implementation_architecture_reviewer_prompt_checks_all_backend_files_and_contract(
        self,
    ) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        repository.stop("implementation")

        [call] = [
            call
            for call in repository.reviewer_calls()
            if "\nRole: architecture\n" in call["prompt"]
        ]
        prompt = call["prompt"]
        architecture_role = prompt[
            prompt.index("### Role: architecture") : prompt.index(
                "### Role: documentation and tasks"
            )
        ]

        self.assertIn("Read `docs/architecture.md` completely", architecture_role)
        self.assertIn("including parts committed or written in earlier requests", architecture_role)
        self.assertIn(
            "When the change edits an HTTP contract file under `backend/api/`, read `.agents/skills/api-design/SKILL.md`",
            architecture_role,
        )
        self.assertIn("Check all backend files belonging to the change", architecture_role)
        self.assertIn("including every command", architecture_role)
        self.assertIn("Frontend architecture rules remain applicable.", architecture_role)
        self.assertNotIn("keeps its existing structure", architecture_role)

    def test_implementation_docs_reviewer_prompt_checks_agent_instructions(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        repository.stop("implementation")

        [call] = [
            call
            for call in repository.reviewer_calls()
            if "\nRole: documentation and tasks\n" in call["prompt"]
        ]
        prompt = call["prompt"]
        docs_role = prompt[
            prompt.index("### Role: documentation and tasks") : prompt.index("### Role: tests")
        ]
        self.assertIn("Check the edits to agent instructions the change made", docs_role)

    def test_tests_reviewer_prompt_requires_failing_tests_reproduced_bugs_waits_and_covered_packages(
        self,
    ) -> None:
        repository = HookRepository(self)
        repository.write_change(spec_names=("widgets", "gadgets"), tasks="- [x] 1.1 Build\n")
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        repository.stop("implementation")

        self.assertEqual(len(repository.reviewer_calls()), 5)
        [call] = [
            call for call in repository.reviewer_calls() if "\nRole: tests\n" in call["prompt"]
        ]
        prompt = call["prompt"]
        tests_role = prompt[
            prompt.index("### Role: tests") : prompt.index("### Role: repeat review")
        ]

        self.assertIn("`.agents/skills/testing/SKILL.md`", tests_role)
        self.assertIn("The test fails without the behavior it pins.", tests_role)
        self.assertIn(
            "A bug fix has a test that reproduces the bug and fails on the code before the fix.",
            tests_role,
        )
        self.assertIn("waits for a condition up to a limit instead of a fixed pause", tests_role)
        self.assertIn(
            "Select verification commands according to the scope rule in `AGENTS.md` and `docs/testing.md#checking-work`.",
            tests_role,
        )
        self.assertIn(
            "For a small, localized change, accept targeted package or screen tests named in the task's own verification",
            tests_role,
        )
        self.assertIn(
            "Do not require `mise run test` solely because ordinary tests were added or changed.",
            tests_role,
        )
        self.assertIn(
            "When the scope rule requires `mise run test`, the change names it in its `## Change verification` section.",
            tests_role,
        )
        self.assertIn(
            "A task that changes tests in `backend/internal/agent/e2e`, `backend/internal/modules/backups/e2e` or `backend/e2e/upgrade`, the verification agent's code or the `Dockerfile` names the command that repeats those tests in its own verification.",
            tests_role,
        )
        self.assertIn(
            "No section or task verification runs `mise run test-full`, which agents run only on the user's direct request.",
            tests_role,
        )
        self.assertIn(
            "A verification that runs `mise run test-full` is a `missing_test` finding that asks for the specific commands instead.",
            tests_role,
        )
        self.assertIn(
            "Naming `mise run test-full` as something a task adds or edits, with a verification that runs another command, is never a finding.",
            tests_role,
        )
        self.assertIn("and so is a task that leaves out its own command.", tests_role)
        self.assertIn(
            "a change that adds or changes scenarios of the browser end-to-end suite in `e2e/browser/scenarios` also names `mise run e2e:browser` there",
            tests_role,
        )
        self.assertIn(
            "A section that leaves out the command it owes is a `missing_test` finding", tests_role
        )
        self.assertIn(
            "such as the hook tests in `.agent-hooks/`, need an explicit verification in their task instead",
            tests_role,
        )

    def test_blocks_turn_with_budget_reason_when_third_wave_of_reviewers_does_not_fit(self) -> None:
        repository = HookRepository(self)
        repository.set_stub(
            repository.review_client, {"sleep_seconds": THIRD_WAVE_REVIEWER_SECONDS}
        )
        repository.write_change(
            spec_names=("widgets", *(f"spec{index}" for index in range(10))),
            tasks="- [x] 1.1 Build\n",
        )
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop_in_process(
            "implementation", THIRD_WAVE_BUDGET_SECONDS, THIRD_WAVE_REVIEWER_TIMEOUT_SECONDS
        )

        self.assertEqual(response["decision"], "block")
        self.assertIn(review_gate.BUDGET_EXHAUSTED_ERROR, response["reason"])
        self.assertEqual(len(repository.reviewer_calls()), 12)

    def test_runs_two_waves_of_reviewers_without_running_checks(self) -> None:
        repository = HookRepository(self)
        repository.set_stub(
            "golangci-lint", {"sleep_seconds": TWO_WAVES_CHECKS_TIMEOUT_SECONDS - 2}
        )
        repository.set_stub(
            repository.review_client, {"sleep_seconds": TWO_WAVES_REVIEWER_TIMEOUT_SECONDS - 2}
        )
        repository.write_change(
            spec_names=("widgets", *(f"spec{index}" for index in range(4))),
            tasks="- [x] 1.1 Build\n",
        )
        repository.submit_prompt("Continue")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")
        repository.complete_tasks()

        response = repository.stop_in_process(
            "implementation",
            TWO_WAVES_BUDGET_SECONDS,
            TWO_WAVES_REVIEWER_TIMEOUT_SECONDS,
            checks_timeout_seconds=TWO_WAVES_CHECKS_TIMEOUT_SECONDS,
        )

        self.assert_accepts_change_and_asks_for_summary(response)
        self.assertEqual(repository.check_calls(), [])
        self.assertEqual(len(repository.reviewer_calls()), 8)

    def test_blocks_turn_and_requires_agent_to_resolve_findings_when_reviewer_finds_bug(
        self,
    ) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result(
            {"findings": [finding("bug", "Race between the check and the write.")]}
        )
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn(
            "[bug] src/widgets.py:10: Race between the check and the write.", response["reason"]
        )
        self.assertIn("without asking the user", response["reason"])
        self.assert_asks_for_short_reply(response["reason"])

    def test_blocks_turn_and_asks_to_move_code_or_change_architecture_doc_when_reviewer_finds_architecture_violation(
        self,
    ) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result(
            {
                "findings": [
                    finding("architecture", "The service imports another module's repository.")
                ]
            }
        )
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn(
            "[architecture] src/widgets.py:10: The service imports another module's repository.",
            response["reason"],
        )
        self.assertIn(
            "On an architecture finding, move the code to follow docs/architecture.md, "
            "or change docs/architecture.md in the same change when the rule itself is wrong.",
            response["reason"],
        )
        self.assert_asks_for_short_reply(response["reason"])

    def test_marks_minor_findings_optional_when_turn_is_blocked(self) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result(
            {
                "findings": [
                    finding("docs", "The README does not mention the command."),
                    finding("minor", "Long function."),
                ]
            }
        )
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertIn("[minor] (optional) src/widgets.py:10: Long function.", response["reason"])

    def test_accepts_change_and_asks_for_summary_with_minor_findings_once(self) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [finding("minor", "Long function.")]})
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")
        repository.clear_calls()
        summary_response = repository.stop("implementation")

        self.assert_accepts_change_and_asks_for_summary(response)
        self.assertIn("[minor] (optional) src/widgets.py:10: Long function.", response["reason"])
        self.assertEqual(summary_response, {})
        self.assertEqual(repository.calls(), [])

    def test_blocks_turn_when_reviewer_answer_does_not_match_schema(self) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result(
            {"findings": [{"category": "typo", "file": "a", "summary": "b"}]}
        )
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn("does not match the schema", response["reason"])

    def test_blocks_turn_with_failure_reason_when_reviewer_crashes(self) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_failure("rate limit exceeded")
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn("rate limit exceeded", response["reason"])


class ImplementationGateRepeatTest(SummaryAssertions, unittest.TestCase):
    def block_with_bug(self, repository: HookRepository) -> None:
        repository.set_reviewer_result(
            {"findings": [finding("bug", "Race between the check and the write.")]}
        )
        repository.submit_prompt("Continue")
        repository.write("src/widgets.py", "WIDGETS = []\n")
        repository.write("src/gadgets.py", "GADGETS = []\n")
        repository.complete_tasks()

        repository.stop("implementation")
        snapshot_path = repository.root / ".git/agent-review-hooks/session-1.json"
        original_snapshot = snapshot_path.read_bytes()

        repository.submit_prompt("AUTOMATIC REVIEW: implementation\nFix the findings")
        self.assertEqual(snapshot_path.read_bytes(), original_snapshot)
        repository.clear_calls()

    def test_repeat_runs_one_reviewer_with_previous_findings_and_paths_changed_after_review(
        self,
    ) -> None:
        repository = HookRepository(self)
        self.block_with_bug(repository)
        repository.write("src/widgets.py", "WIDGETS = ()\n")

        repository.stop("implementation")

        [call] = repository.reviewer_calls()
        self.assertIn("Role: repeat review", call["prompt"])
        self.assertIn("Race between the check and the write.", call["prompt"])
        changed_paths = call["prompt"].split("Paths changed after the previous review:")[1]
        self.assertIn("- src/widgets.py", changed_paths)
        self.assertNotIn("src/gadgets.py", changed_paths)

    def test_accepted_repeat_clears_findings(self) -> None:
        repository = HookRepository(self)
        self.block_with_bug(repository)
        repository.set_reviewer_result({"findings": []})
        repository.write("src/widgets.py", "WIDGETS = ()\n")

        repository.stop("implementation")
        repository.write("src/widgets.py", "WIDGETS = (1,)\n")
        repository.clear_calls()
        repository.stop("implementation")

        self.assertEqual(len(repository.reviewer_calls()), 4)

    def test_asks_for_summary_with_review_fixes_once_after_repeat_accepts_change(self) -> None:
        repository = HookRepository(self)
        self.block_with_bug(repository)
        repository.set_reviewer_result({"findings": [finding("minor", "Long function.")]})
        repository.write("src/widgets.py", "WIDGETS = ()\n")

        response = repository.stop("implementation")
        repository.clear_calls()
        summary_response = repository.stop("implementation")

        self.assert_accepts_change_and_asks_for_summary(response)
        self.assertIn("Long function.", response["reason"])
        self.assertEqual(summary_response, {})
        self.assertEqual(repository.calls(), [])

    def test_stops_agent_with_remaining_findings_after_repeat_limit(self) -> None:
        repository = HookRepository(self)
        self.block_with_bug(repository)
        repeat_responses = []
        for attempt in range(review_gate.MAX_FULL_REVIEW_REVISIONS):
            repository.write("src/widgets.py", f"WIDGETS = ({attempt},)\n")
            repeat_responses.append(repository.stop("implementation"))

        last_response = repeat_responses[-1]

        self.assertEqual(
            [response.get("decision") for response in repeat_responses[:-1]],
            ["block"] * (review_gate.MAX_FULL_REVIEW_REVISIONS - 1),
        )
        self.assertIs(last_response["continue"], False)
        self.assertIn("Race between the check and the write.", last_response["stopReason"])
        self.assertIn("the change is not accepted", last_response["stopReason"])
        self.assertIn("`mise run review`", last_response["stopReason"])


def get_effort(call: ReviewerCall) -> str:
    effort = call["effort_env"]
    assert effort is not None
    return effort


class ManualReviewTest(unittest.TestCase):
    review_client: str

    def run_review(
        self, repository: HookRepository, *arguments: str
    ) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(HOOK_PATH), "review", *arguments],
            cwd=repository.root,
            env=repository.environment,
            stdin=subprocess.DEVNULL,
            text=True,
            capture_output=True,
        )

    def test_runs_no_verification_commands_when_manual_review_starts(self) -> None:
        repository = HookRepository(self)
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")
        repository.write("frontend/src/widgets.ts", "export {};\n")
        repository.write("website/app/widgets.tsx", "export {};\n")
        repository.write("backend/internal/agent/widgets/widgets.go", "package widgets\n")
        repository.write(".agent-hooks/sample_test.py", FAILING_HOOK_TEST)
        for name in ("golangci-lint", "pnpm", "npm", "mise"):
            repository.set_stub(name, {"exit_code": 1, "output": "must not run"})
        repository.complete_tasks(
            verification_section=verification_section(
                "mise run test",
                "mise run e2e:browser",
            )
        )

        process = self.run_review(repository)

        self.assertEqual(process.returncode, 0, process.stdout + process.stderr)
        self.assertEqual(repository.check_calls(), [])
        self.assertEqual(len(repository.reviewer_calls()), 7)

    def test_runs_manual_reviewers_concurrently_across_quick_and_full_reviews(self) -> None:
        repository = HookRepository(self)
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")
        repository.complete_tasks()
        repository.set_stub(
            repository.review_client,
            {
                "wait_for_efforts": ["medium", "high"]
                if repository.review_client == "codex"
                else ["low", "medium"]
            },
        )

        process = self.run_review(repository)

        self.assertEqual(process.returncode, 0, process.stdout + process.stderr)
        self.assertEqual(len(repository.reviewer_calls()), 7)

    def test_runs_quick_checks_and_full_review_and_exits_with_code_0_when_everything_accepts(
        self,
    ) -> None:
        repository = HookRepository(self)
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")
        repository.complete_tasks()

        process = self.run_review(repository)

        self.assertEqual(process.returncode, 0, process.stdout + process.stderr)

        efforts = sorted(get_effort(call) for call in repository.reviewer_calls())
        self.assertEqual(
            efforts, sorted([repository.quick_effort] * 3 + [repository.full_effort] * 4)
        )
        quick_prompts = [
            call["prompt"]
            for call in repository.reviewer_calls()
            if get_effort(call) == repository.quick_effort
        ]
        self.assertTrue(
            all("started `mise run review` by hand" in prompt for prompt in quick_prompts)
        )

        self.assertEqual(repository.check_calls(), [])
        self.assertIn("Quick check `code-shape`: accepted.", process.stdout)
        self.assertIn("Full implementation review accepted the change.", process.stdout)
        self.assertIn(
            "Result: the review accepted the change in its current state.", process.stdout
        )

    def test_skips_code_shape_when_no_code_file_changed(self) -> None:
        repository = HookRepository(self)
        repository.complete_tasks()

        process = self.run_review(repository)

        self.assertEqual(process.returncode, 0, process.stdout + process.stderr)
        self.assertEqual(
            len(
                [
                    call
                    for call in repository.reviewer_calls()
                    if get_effort(call) == repository.quick_effort
                ]
            ),
            2,
        )
        self.assertIn("Quick check `code-shape`: skipped", process.stdout)

    def test_runs_whole_review_again_when_hook_already_accepted_state(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.complete_tasks()
        repository.stop("implementation")
        repository.clear_calls()

        process = self.run_review(repository, CHANGE_NAME)

        self.assertEqual(process.returncode, 0, process.stdout + process.stderr)
        full_prompts = [
            call["prompt"]
            for call in repository.reviewer_calls()
            if get_effort(call) == repository.full_effort
        ]
        self.assertEqual(len(full_prompts), 4)
        self.assertFalse(any("\nRole: repeat review\n" in prompt for prompt in full_prompts))

    def test_exits_with_code_1_and_prints_findings_that_next_stop_hook_repeats(self) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result(
            {
                "ok": True,
                "reason": "",
                "findings": [finding("bug", "Race between the check and the write.")],
            }
        )
        repository.complete_tasks()

        process = self.run_review(repository)
        repository.clear_calls()
        repository.submit_prompt("Run mise review and fix what it finds")
        repository.write("src/widgets.py", "WIDGETS = ()\n")
        repository.stop("implementation")

        self.assertEqual(process.returncode, 1)
        self.assertIn(
            "[bug] src/widgets.py:10: Race between the check and the write.", process.stdout
        )
        self.assertIn("Result: the review did not accept the change.", process.stdout)
        [call] = repository.reviewer_calls()
        self.assertIn("Role: repeat review", call["prompt"])

    def test_repeats_review_over_findings_of_previous_manual_run(self) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result(
            {
                "ok": True,
                "reason": "",
                "findings": [finding("bug", "Race between the check and the write.")],
            }
        )
        repository.complete_tasks()
        self.run_review(repository)
        repository.clear_calls()
        repository.set_reviewer_result({"ok": True, "reason": "", "findings": []})
        repository.write("src/widgets.py", "WIDGETS = ()\n")

        process = self.run_review(repository)

        self.assertEqual(process.returncode, 0, process.stdout + process.stderr)
        [full_prompt] = [
            call["prompt"]
            for call in repository.reviewer_calls()
            if get_effort(call) == repository.full_effort
        ]
        self.assertIn("\nRole: repeat review\n", full_prompt)
        self.assertIn("Race between the check and the write.", full_prompt)

    def test_exits_with_code_1_and_names_failed_quick_check(self) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result(
            {"ok": False, "reason": "README.md: the command is not documented.", "findings": []}
        )
        repository.complete_tasks()

        process = self.run_review(repository)

        self.assertEqual(process.returncode, 1)
        self.assertIn(
            "Quick check `diff`: not accepted.\nREADME.md: the command is not documented.",
            process.stdout,
        )

    def test_skips_manual_review_inside_reviewer(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Implement widgets")
        repository.complete_tasks()
        snapshot_path = repository.root / ".git/agent-review-hooks/session-1.json"
        original_snapshot = snapshot_path.read_bytes()
        repository.environment["AGENT_REVIEW_HOOK_ACTIVE"] = "1"

        process = self.run_review(repository, CHANGE_NAME)

        self.assertEqual(process.returncode, 0, process.stdout + process.stderr)
        self.assertEqual(repository.calls(), [])
        self.assertEqual(snapshot_path.read_bytes(), original_snapshot)
        self.assertIn("skipped", process.stdout)

    def test_reports_selected_client_when_reviewer_cannot_run(self) -> None:
        for failure in ("missing", "authentication failed", "model unavailable"):
            with self.subTest(failure=failure):
                repository = HookRepository(
                    self, missing_tools=(self.review_client,) if failure == "missing" else ()
                )
                repository.complete_tasks()
                if failure != "missing":
                    repository.set_reviewer_failure(failure)

                process = self.run_review(repository)

                self.assertEqual(process.returncode, 1, process.stdout + process.stderr)
                self.assertIn(f"Review client: {self.review_client}", process.stdout)
                self.assertIn(
                    self.review_client if failure == "missing" else failure, process.stdout
                )
                self.assertFalse(
                    any(call["name"] != self.review_client for call in repository.reviewer_calls())
                )

    def test_exits_with_code_2_without_reviewers_when_tasks_remain_unchecked(self) -> None:
        repository = HookRepository(self)

        process = self.run_review(repository)

        self.assertEqual(process.returncode, 2)
        self.assertIn(
            f"No active OpenSpec change has every task checked. Active changes: `{CHANGE_NAME}`.",
            process.stdout,
        )
        self.assertEqual(repository.calls(), [])

    def test_exits_with_code_2_when_named_change_does_not_exist(self) -> None:
        repository = HookRepository(self)
        repository.complete_tasks()

        process = self.run_review(repository, "add-gizmos")

        self.assertEqual(process.returncode, 2)
        self.assertIn("No active OpenSpec change is named `add-gizmos`.", process.stdout)
        self.assertEqual(repository.calls(), [])


class NecessityDuringImplementationReviewTest(unittest.TestCase):
    def block_with_spec_mismatch(self, repository: HookRepository) -> None:
        repository.set_reviewer_result(
            {"findings": [finding("spec_mismatch", "The spec requires 409, the code answers 400.")]}
        )
        repository.submit_prompt("Continue")
        repository.complete_tasks()
        repository.stop("implementation")
        repository.clear_calls()

    def test_skips_necessity_when_agent_fixes_delta_spec_after_finding(self) -> None:
        repository = HookRepository(self)
        self.block_with_spec_mismatch(repository)
        repository.write(
            f"openspec/changes/{CHANGE_NAME}/specs/widgets/spec.md", "# widgets\n\nAnswers 400.\n"
        )

        response = repository.stop("necessity")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_runs_necessity_when_prompt_has_openspec_planning_command(self) -> None:
        repository = HookRepository(self)
        self.block_with_spec_mismatch(repository)
        repository.submit_prompt("/opsx:propose a new change")
        repository.write(
            f"openspec/changes/{CHANGE_NAME}/specs/widgets/spec.md", "# widgets\n\nAnswers 400.\n"
        )

        repository.stop("necessity")

        self.assertEqual(len(repository.reviewer_calls()), 1)


class ProposalGateStartTest(unittest.TestCase):
    def test_runs_full_review_when_agent_writes_all_artifacts_without_command(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec for gadgets")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(len(repository.reviewer_calls()), 6)

    def test_reviews_complete_proposal_from_command_skill_and_plain_requests(self) -> None:
        for request in (
            "/opsx:propose add-gadgets",
            "Use openspec-propose to add gadgets",
            "Write a spec for gadgets",
        ):
            with self.subTest(request=request):
                repository = HookRepository(self)
                repository.submit_prompt(request)
                repository.write_proposal()

                self.assertEqual(repository.stop("proposal"), {})
                self.assertEqual(len(repository.reviewer_calls()), 6)
                self.assertEqual(repository.calls("openspec")[0]["arguments"][0], "status")

    def test_skips_review_without_calling_openspec_when_tasks_are_missing(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec for gadgets")
        repository.write(f"openspec/changes/{PROPOSAL_NAME}/proposal.md", "# Proposal\n")

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_review_without_calling_openspec_when_tasks_file_has_no_tasks(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec for gadgets")
        repository.write_proposal(tasks="")

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_review_when_openspec_reports_incomplete_artifacts(self) -> None:
        repository = HookRepository(self)
        repository.set_stub(
            "openspec", {"status": {"output": json.dumps({"isPlanningComplete": False})}}
        )
        repository.submit_prompt("Write a spec for gadgets")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_review_when_openspec_status_fails(self) -> None:
        repository = HookRepository(self)
        repository.set_stub(
            "openspec", {"status": {"exit_code": 1, "output": "openspec: unknown change"}}
        )
        repository.submit_prompt("Write a spec for gadgets")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_review_without_calling_openspec_when_task_is_checked(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Fix the widgets spec")
        repository.write(
            f"openspec/changes/{CHANGE_NAME}/specs/widgets/spec.md", "# widgets\n\nAnswers 409.\n"
        )

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_review_when_change_files_are_unchanged_during_turn(self) -> None:
        repository = HookRepository(self)
        repository.write_proposal()
        repository.submit_prompt("What is in the proposal?")

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_review_in_new_session_when_files_match_accepted_state(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec", session_id="first")
        repository.submit_prompt("Look at the spec", session_id="second")
        repository.write_proposal()
        repository.stop("proposal", session_id="first")
        repository.clear_calls()

        response = repository.stop("proposal", session_id="second")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_runs_review_again_when_design_changes_after_accepted_review(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()
        repository.stop("proposal")
        repository.clear_calls()
        repository.submit_prompt("Fix the design")
        repository.write(
            f"openspec/changes/{PROPOSAL_NAME}/design.md", "# Design\n\nIn-memory cache.\n"
        )

        repository.stop("proposal")

        self.assertEqual(len(repository.reviewer_calls()), 6)

    def test_skips_archived_changes(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Archive it")
        repository.write_change(
            spec_names=("gadgets",),
            tasks="- [ ] 1.1 Build\n",
            name=f"archive/2026-10-01-{PROPOSAL_NAME}",
        )

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])


class ProposalGateValidationTest(unittest.TestCase):
    def test_blocks_turn_with_validation_output_when_requirement_has_no_scenario(self) -> None:
        repository = HookRepository(self)
        repository.set_stub(
            "openspec",
            {
                "validate": {
                    "exit_code": 1,
                    "output": '✗ [ERROR] gadgets/spec.md: ADDED "Gadgets" must include at least one scenario\n',
                }
            },
        )
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn(f"`openspec validate {PROPOSAL_NAME} --strict`", response["reason"])
        self.assertIn("must include at least one scenario", response["reason"])
        self.assertEqual(repository.reviewer_calls(), [])

    def test_blocks_turn_without_running_command_when_verification_section_has_forbidden_command(
        self,
    ) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal(
            tasks="- [ ] 1.1 Build\n"
            + verification_section("mise run backend:test-package ./internal/lib/...")
        )

        response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn(
            "`mise run backend:test-package ./internal/lib/...` is not an allowed command",
            response["reason"],
        )
        self.assertIn(review_gate.ALLOWED_VERIFICATION_FORMS, response["reason"])
        self.assertEqual(repository.calls("mise"), [])
        self.assertEqual(
            [call["arguments"][0] for call in repository.calls("openspec")], ["status"]
        )
        self.assertEqual(repository.reviewer_calls(), [])


class ProposalGateReviewTest(ShortReplyAssertions, SummaryAssertions, unittest.TestCase):
    def test_runs_selected_full_profile_per_spec_and_proposal_role(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal(spec_names=("gadgets", "gizmos"))

        repository.stop("proposal")

        calls = repository.reviewer_calls()
        self.assertEqual(len(calls), 7)
        for call in calls:
            self.assertEqual(call["name"], repository.review_client)
            self.assertEqual(call["effort_env"], repository.full_effort)
        roles = sorted(
            line
            for call in calls
            for line in call["prompt"].splitlines()
            if line.startswith("Role:")
        )
        self.assertEqual(
            roles,
            [
                "Role: architecture",
                f"Role: delta spec `openspec/changes/{PROPOSAL_NAME}/specs/gadgets/spec.md`",
                f"Role: delta spec `openspec/changes/{PROPOSAL_NAME}/specs/gizmos/spec.md`",
                "Role: names",
                "Role: necessity",
                "Role: proposal and design",
                "Role: tasks",
            ],
        )

    def test_runs_seven_reviewers_of_two_spec_proposal_at_most_six_at_a_time(self) -> None:
        repository = HookRepository(self)
        repository.set_stub(repository.review_client, {"wait_for_concurrent": 6})
        repository.submit_prompt("Write a spec")
        repository.write_proposal(spec_names=("gadgets", "gizmos"))

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(len(repository.reviewer_calls()), 7)
        concurrency = json.loads((repository.stubs_dir / "concurrency.json").read_text())
        self.assertEqual(concurrency, {"active": 0, "peak": 6})

    def get_proposal_role_section(
        self, repository: HookRepository, role: str, next_role: str
    ) -> str:
        [call] = [
            call for call in repository.reviewer_calls() if f"\nRole: {role}\n" in call["prompt"]
        ]
        prompt = call["prompt"]
        return prompt[prompt.index(f"### Role: {role}") : prompt.index(f"### Role: {next_role}")]

    def test_necessity_reviewer_prompt_returns_simplify_or_scope(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        repository.stop("proposal")

        necessity_role = self.get_proposal_role_section(repository, "necessity", "architecture")
        self.assertIn('"The best code is unwritten"', necessity_role)
        self.assertIn("Return a `simplify` finding when the user loses nothing", necessity_role)
        self.assertIn(
            "a `scope` finding when the user loses a feature, screen, scenario or setting",
            necessity_role,
        )

    def test_architecture_reviewer_prompt_checks_design_and_operations_and_exempts_old_code(
        self,
    ) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        repository.stop("proposal")

        architecture_role = self.get_proposal_role_section(repository, "architecture", "names")
        self.assertIn("Read `docs/architecture.md` completely", architecture_role)
        self.assertIn(
            "When the change adds or changes an HTTP operation, read `.agents/skills/api-design/SKILL.md`",
            architecture_role,
        )
        self.assertIn("Check all backend files belonging to the change", architecture_role)
        self.assertIn("including every command", architecture_role)
        self.assertIn("Frontend architecture rules remain applicable.", architecture_role)
        self.assertNotIn("keeps its existing structure", architecture_role)

    def test_names_reviewer_prompt_checks_naming_rules_data_section_and_glossary(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        repository.stop("proposal")

        names_role = self.get_proposal_role_section(repository, "names", "proposal and design")
        self.assertIn(
            'the sections "Naming" and "Functions, methods and types" of `AGENTS.md`', names_role
        )
        self.assertIn(
            'the table naming rules in the "Data" section of `docs/architecture.md`', names_role
        )
        self.assertIn("is a `mismatch` finding that names the rule and suggests a name", names_role)
        self.assertIn(
            "`docs/glossary.md` does not define, while the change does not add it, is a `gap` finding",
            names_role,
        )

    def test_tasks_reviewer_prompt_requires_every_scenario_covered_by_task_with_test(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        repository.stop("proposal")

        [call] = [
            call for call in repository.reviewer_calls() if "\nRole: tasks\n" in call["prompt"]
        ]
        self.assertIn(
            "every scenario of the delta specs is covered by a task with a test", call["prompt"]
        )
        self.assertIn("the task order can be followed", call["prompt"])

    def test_tasks_reviewer_prompt_requires_verification_section_for_tested_packages_except_hook_tests(
        self,
    ) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        repository.stop("proposal")

        [call] = [
            call for call in repository.reviewer_calls() if "\nRole: tasks\n" in call["prompt"]
        ]
        prompt = call["prompt"]
        tasks_role = prompt[
            prompt.index("### Role: tasks") : prompt.index("### Role: repeat review")
        ]

        self.assertIn(
            "Select verification commands according to the scope rule in `AGENTS.md` and `docs/testing.md#checking-work`.",
            tasks_role,
        )
        self.assertIn(
            "For a small, localized change, accept targeted package or screen tests named in the task's own verification",
            tasks_role,
        )
        self.assertIn(
            "Do not require `mise run test` solely because ordinary tests were added or changed.",
            tasks_role,
        )
        self.assertIn(
            "When the scope rule requires `mise run test`, the change has a `## Change verification` section that names it.",
            tasks_role,
        )
        self.assertIn(
            "Running `mise run test` in a task or in the section is never a finding.", tasks_role
        )
        self.assertIn(
            "a section that leaves out the command it owes, is a `tasks` finding that asks for it",
            tasks_role,
        )
        self.assertIn(
            "A task that changes tests in `backend/internal/agent/e2e`, `backend/internal/modules/backups/e2e` or `backend/e2e/upgrade`, the verification agent's code or the `Dockerfile` names the command that repeats those tests in its own verification.",
            tasks_role,
        )
        self.assertIn(
            "No section or task verification runs `mise run test-full`, which agents run only on the user's direct request.",
            tasks_role,
        )
        self.assertIn(
            "A verification that runs `mise run test-full` is a `tasks` finding that asks for the specific commands instead.",
            tasks_role,
        )
        self.assertIn(
            "Naming `mise run test-full` as something a task adds or edits, with a verification that runs another command, is never a finding.",
            tasks_role,
        )
        self.assertIn("and so is a task that leaves out its own command.", tasks_role)
        self.assertIn(
            "a change adding or changing scenarios of the browser end-to-end suite in `e2e/browser/scenarios` also names `mise run e2e:browser` there",
            tasks_role,
        )
        self.assertIn(
            "such as the hook tests in `.agent-hooks/`, need an explicit verification in their task instead",
            tasks_role,
        )

    def test_blocks_turn_with_budget_reason_when_third_wave_of_reviewers_does_not_fit(self) -> None:
        repository = HookRepository(self)
        repository.set_stub(
            repository.review_client, {"sleep_seconds": THIRD_WAVE_REVIEWER_SECONDS}
        )
        repository.submit_prompt("Write a spec")
        repository.write_proposal(spec_names=tuple(f"spec{index}" for index in range(11)))

        response = repository.stop_in_process(
            "proposal", THIRD_WAVE_BUDGET_SECONDS, THIRD_WAVE_REVIEWER_TIMEOUT_SECONDS
        )

        self.assertEqual(response["decision"], "block")
        self.assertIn(review_gate.BUDGET_EXHAUSTED_ERROR, response["reason"])
        self.assertEqual(len(repository.reviewer_calls()), 12)

    def test_runs_two_waves_of_reviewers_after_validation_uses_checks_total_limit(self) -> None:
        repository = HookRepository(self)
        repository.set_stub(
            "openspec", {"validate": {"sleep_seconds": TWO_WAVES_CHECKS_TIMEOUT_SECONDS - 2}}
        )
        repository.set_stub(
            repository.review_client, {"sleep_seconds": TWO_WAVES_REVIEWER_TIMEOUT_SECONDS - 2}
        )
        repository.submit_prompt("Write a spec")
        repository.write_proposal(spec_names=tuple(f"spec{index}" for index in range(5)))

        response = repository.stop_in_process(
            "proposal",
            TWO_WAVES_BUDGET_SECONDS,
            TWO_WAVES_REVIEWER_TIMEOUT_SECONDS,
            checks_timeout_seconds=TWO_WAVES_CHECKS_TIMEOUT_SECONDS,
        )

        self.assertEqual(response, {})
        self.assertEqual(len(repository.reviewer_calls()), 10)

    def test_blocks_turn_and_requires_agent_to_resolve_simplify_finding_without_asking_user(
        self,
    ) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result(
            {
                "findings": [
                    finding(
                        "simplify",
                        "A field in an existing table replaces the new table.",
                        file="design.md:12",
                    )
                ]
            }
        )
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn(
            "[simplify] design.md:12: A field in an existing table replaces the new table.",
            response["reason"],
        )
        self.assertIn("without asking the user", response["reason"])
        self.assert_asks_for_short_reply(response["reason"])

    def test_asks_for_summary_with_scope_finding_without_requiring_fix(self) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result(
            {
                "findings": [
                    finding("scope", "The settings screen can be dropped.", file="proposal.md:5")
                ]
            }
        )
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("proposal")
        repository.clear_calls()
        summary_response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn("Full proposal review accepted the proposal.", response["reason"])
        self.assertIn(
            "[scope] (the user decides) proposal.md:5: The settings screen can be dropped.",
            response["reason"],
        )
        self.assertIn("summary of the proposal", response["reason"])
        self.assert_asks_for_summary_without_review(response["reason"])
        self.assertIn("what the user loses and what they save", response["reason"])
        self.assertEqual(summary_response, {})
        self.assertEqual(repository.calls(), [])

    def test_accepts_proposal_and_shows_minor_findings_to_user(self) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result(
            {"findings": [finding("minor", "Long paragraph.", file="design.md:3")]}
        )
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("proposal")
        repository.clear_calls()
        repeated_response = repository.stop("proposal")

        self.assertNotIn("decision", response)
        self.assertIn("Long paragraph.", response["systemMessage"])
        self.assertEqual(repeated_response, {})
        self.assertEqual(repository.calls(), [])

    def test_blocks_turn_when_reviewer_uses_implementation_category(self) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [finding("bug", "Race.")]})
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn("does not match the schema", response["reason"])


class ProposalGateRepeatTest(ShortReplyAssertions, SummaryAssertions, unittest.TestCase):
    def block_with_gap(self, repository: HookRepository) -> None:
        repository.set_reviewer_result(
            {
                "findings": [
                    finding(
                        "gap",
                        "The answer for an empty list is missing.",
                        file="specs/gadgets/spec.md:8",
                    )
                ]
            }
        )
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        repository.stop("proposal")
        snapshot_path = repository.root / ".git/agent-review-hooks/session-1.json"
        original_snapshot = snapshot_path.read_bytes()

        repository.submit_prompt("AUTOMATIC REVIEW: proposal\nFix the findings")
        self.assertEqual(snapshot_path.read_bytes(), original_snapshot)
        repository.clear_calls()

    def test_blocks_turn_with_gap_and_asks_for_short_reply_without_summary(self) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result(
            {
                "findings": [
                    finding(
                        "gap",
                        "The answer for an empty list is missing.",
                        file="specs/gadgets/spec.md:8",
                    )
                ]
            }
        )
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn(
            "[gap] specs/gadgets/spec.md:8: The answer for an empty list is missing.",
            response["reason"],
        )
        self.assertIn("without asking the user", response["reason"])
        self.assert_asks_for_short_reply(response["reason"])

    def test_repeat_runs_one_reviewer_with_previous_findings_and_files_changed_after_review(
        self,
    ) -> None:
        repository = HookRepository(self)
        self.block_with_gap(repository)
        repository.write(
            f"openspec/changes/{PROPOSAL_NAME}/specs/gadgets/spec.md", "# gadgets\n\nEmpty list.\n"
        )

        repository.stop("proposal")

        [call] = repository.reviewer_calls()
        self.assertIn("Role: repeat review", call["prompt"])
        self.assertIn("The answer for an empty list is missing.", call["prompt"])
        changed_paths = call["prompt"].split("Paths changed after the previous review:")[1]
        self.assertIn(f"- openspec/changes/{PROPOSAL_NAME}/specs/gadgets/spec.md", changed_paths)
        self.assertNotIn("design.md", changed_paths)

    def test_asks_for_summary_without_review_once_after_repeat_accepts_proposal(self) -> None:
        repository = HookRepository(self)
        self.block_with_gap(repository)
        repository.set_reviewer_result({"findings": []})
        repository.write(
            f"openspec/changes/{PROPOSAL_NAME}/specs/gadgets/spec.md", "# gadgets\n\nEmpty list.\n"
        )

        response = repository.stop("proposal")
        repository.clear_calls()
        summary_response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn("accepted the proposal", response["reason"])
        self.assertIn("summary of the proposal", response["reason"])
        self.assert_asks_for_summary_without_review(response["reason"])
        self.assertEqual(summary_response, {})
        self.assertEqual(repository.calls(), [])

    def test_lists_scope_option_in_summary_without_review_when_repeat_accepts_proposal(
        self,
    ) -> None:
        repository = HookRepository(self)
        self.block_with_gap(repository)
        repository.set_reviewer_result(
            {
                "findings": [
                    finding("scope", "The settings screen can be dropped.", file="proposal.md:5")
                ]
            }
        )
        repository.write(
            f"openspec/changes/{PROPOSAL_NAME}/specs/gadgets/spec.md", "# gadgets\n\nEmpty list.\n"
        )

        response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn(
            "[scope] (the user decides) proposal.md:5: The settings screen can be dropped.",
            response["reason"],
        )
        self.assertIn(
            "List every scope finding separately: what the user loses and what they save",
            response["reason"],
        )
        self.assert_asks_for_summary_without_review(response["reason"])

    def test_stops_agent_with_remaining_findings_after_repeat_limit(self) -> None:
        repository = HookRepository(self)
        self.block_with_gap(repository)
        repeat_responses = []
        for attempt in range(review_gate.MAX_FULL_REVIEW_REVISIONS):
            repository.write(
                f"openspec/changes/{PROPOSAL_NAME}/design.md", f"# Design\n\nAttempt {attempt}.\n"
            )
            repeat_responses.append(repository.stop("proposal"))

        last_response = repeat_responses[-1]

        self.assertEqual(
            [response.get("decision") for response in repeat_responses[:-1]],
            ["block"] * (review_gate.MAX_FULL_REVIEW_REVISIONS - 1),
        )
        self.assertIs(last_response["continue"], False)
        self.assertIn("Full proposal review", last_response["stopReason"])
        self.assertIn("The answer for an empty list is missing.", last_response["stopReason"])


class QuickGatesDuringProposalReviewTest(unittest.TestCase):
    def test_skips_quick_checks_when_turn_changes_only_ready_proposal(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        responses = [
            repository.stop(gate) for gate in ("necessity", "architecture", "diff", "code-shape")
        ]

        self.assertEqual(responses, [{}, {}, {}, {}])
        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_code_shape_that_blocked_earlier_in_request_when_turn_leaves_only_ready_proposal(
        self,
    ) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result({"ok": False, "reason": "service.go mixes two concepts."})
        repository.submit_prompt("Add a widget")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")

        self.assertEqual(repository.stop("code-shape")["decision"], "block")
        repository.clear_calls()

        (repository.root / "backend/internal/features/widgets/service.go").unlink()
        repository.write_proposal()

        response = repository.stop("code-shape")

        self.assertEqual(response, {})
        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_quick_checks_in_summary_turn_after_accepted_review(self) -> None:
        repository = HookRepository(self)
        repository.set_reviewer_result(
            {"findings": [finding("scope", "The settings screen can be dropped.")]}
        )
        repository.submit_prompt("Write a spec")
        repository.write_proposal()
        repository.stop("proposal")
        repository.clear_calls()

        responses = [
            repository.stop(gate, message="Proposal summary.")
            for gate in ("necessity", "architecture", "diff", "code-shape")
        ]

        self.assertEqual(responses, [{}, {}, {}, {}])
        self.assertEqual(repository.reviewer_calls(), [])

    def test_runs_necessity_when_agent_wrote_only_proposal(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write(f"openspec/changes/{PROPOSAL_NAME}/proposal.md", "# Proposal\n")

        repository.stop("necessity", message="Is the settings screen needed?")

        [call] = repository.reviewer_calls()
        self.assertIn("Under review: OpenSpec proposal", call["prompt"])

    def test_runs_quick_checks_without_calling_openspec_when_code_changes_too(self) -> None:
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec and start")
        repository.write_proposal()
        repository.write("src/gadgets.py", "GADGETS = []\n")

        response = repository.stop("diff")

        self.assertEqual(response, {})
        self.assertEqual(len(repository.reviewer_calls()), 1)
        self.assertEqual(repository.calls("openspec"), [])


PARITY_TEST_CLASSES = (
    DiffGateTest,
    CodeShapeGateTest,
    ImplementationGateStartTest,
    ImplementationGateChecksTest,
    ImplementationGateReviewTest,
    ImplementationGateRepeatTest,
    ManualReviewTest,
    NecessityDuringImplementationReviewTest,
    ProposalGateStartTest,
    ProposalGateValidationTest,
    ProposalGateReviewTest,
    ProposalGateRepeatTest,
    QuickGatesDuringProposalReviewTest,
)


def load_tests(
    loader: unittest.TestLoader, standard_tests: unittest.TestSuite, pattern: str | None
) -> unittest.TestSuite:
    suite = unittest.TestSuite()
    parity_types: dict[tuple[type[unittest.TestCase], str], type[unittest.TestCase]] = {}
    for group in standard_tests:
        assert isinstance(group, unittest.TestSuite)
        for test in group:
            assert isinstance(test, unittest.TestCase)
            if type(test) not in PARITY_TEST_CLASSES:
                suite.addTest(test)
                continue
            for client in ("claude", "codex"):
                key = (type(test), client)
                if key not in parity_types:
                    parity_types[key] = type(
                        f"{type(test).__name__}_{client}", (type(test),), {"review_client": client}
                    )
                suite.addTest(parity_types[key](test._testMethodName))
    return suite


if __name__ == "__main__":
    unittest.main()
