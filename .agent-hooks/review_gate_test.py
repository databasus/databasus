from dataclasses import replace
import io
import json
import os
import shutil
import signal
from pathlib import Path
import subprocess
import sys
import tempfile
import textwrap
import time
import unittest
from unittest import mock

import review_gate


HOOK_PATH = Path(__file__).resolve().parent / "review_gate.py"
CHANGE_NAME = "add-widgets"
PROPOSAL_NAME = "add-gadgets"
STUB_NAMES = ("claude", "openspec", "golangci-lint", "pnpm", "npm", "just")

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
    prompt = sys.stdin.read() if name == "claude" else ""
    calls_dir = stubs_dir / "calls"
    calls_dir.mkdir(exist_ok=True)
    (calls_dir / f"{name}-{uuid.uuid4().hex}.json").write_text(
        json.dumps(
            {
                "name": name,
                "arguments": arguments,
                "cwd": os.getcwd(),
                "prompt": prompt,
                "effort_env": os.environ.get("CLAUDE_CODE_EFFORT_LEVEL"),
                "started_at": time.time_ns(),
            }
        )
    )

    config_path = stubs_dir / f"{name}.json"
    config = json.loads(config_path.read_text()) if config_path.exists() else {}
    if name == "openspec":
        config = config.get(arguments[0], {})
    elif name == "npm":
        config = config.get(arguments[-1], {})

    time.sleep(config.get("sleep_seconds", 0))
    if config.get("exit_code"):
        sys.stdout.write(config.get("output", ""))
        sys.exit(config["exit_code"])

    result = config.get("result", {"ok": True, "reason": "", "findings": []})
    if name == "claude":
        sys.stdout.write(json.dumps({"structured_output": result}))
    elif name == "openspec" and arguments[0] == "status":
        sys.stdout.write(config.get("output", json.dumps({"isPlanningComplete": True})))
    else:
        sys.stdout.write(config.get("output", ""))
    """
)
PASSING_HOOK_TEST = textwrap.dedent(
    """
    import unittest


    class SampleTest(unittest.TestCase):
        def test_passes(self):
            self.assertTrue(True)
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
    def __init__(self, test_case, missing_tools=(), is_path_isolated=False):
        temporary_dir = tempfile.TemporaryDirectory(prefix="review-gate-test-")
        test_case.addCleanup(temporary_dir.cleanup)
        base = Path(temporary_dir.name)

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
            (system_dir / "git").symlink_to(shutil.which("git"))
            self.environment["PATH"] = f"{bin_dir}{os.pathsep}{system_dir}"
        else:
            self.environment["PATH"] = f"{bin_dir}{os.pathsep}{os.environ['PATH']}"
        self.environment["REVIEW_GATE_STUBS"] = str(self.stubs_dir)
        self.environment["PYTHONDONTWRITEBYTECODE"] = "1"

        self.git("init", "--quiet")
        self.git("config", "user.email", "test@example.com")
        self.git("config", "user.name", "test")
        self.write("README.md", "test repository\n")
        self.write_change(spec_names=("widgets",), tasks="- [x] 1.1 Build\n- [ ] 1.2 Verify\n")
        self.git("add", ".")
        self.git("commit", "--quiet", "-m", "init")

    def git(self, *arguments):
        subprocess.run(["git", *arguments], cwd=self.root, env=self.environment, check=True, capture_output=True)

    def write(self, relative_path, content):
        path = self.root / relative_path
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)

    def write_change(self, spec_names, tasks, name=CHANGE_NAME):
        change_dir = f"openspec/changes/{name}"
        self.write(f"{change_dir}/proposal.md", "# Proposal\n")
        self.write(f"{change_dir}/design.md", "# Design\n")
        self.write(f"{change_dir}/tasks.md", f"# Tasks\n\n{tasks}")
        for spec_name in spec_names:
            self.write(f"{change_dir}/specs/{spec_name}/spec.md", f"# {spec_name}\n")

    def write_proposal(self, spec_names=("gadgets",), tasks="- [ ] 1.1 Build\n- [ ] 1.2 Verify\n"):
        self.write_change(spec_names=spec_names, tasks=tasks, name=PROPOSAL_NAME)

    def complete_tasks(self, name=CHANGE_NAME, verification_section=""):
        self.write(f"openspec/changes/{name}/tasks.md", f"# Tasks\n\n- [x] 1.1 Build\n- [x] 1.2 Verify\n{verification_section}")

    def set_reviewer_result(self, result):
        self.set_stub("claude", {"result": result})

    def set_reviewer_failure(self, output):
        self.set_stub("claude", {"exit_code": 1, "output": output})

    def set_stub(self, name, config):
        (self.stubs_dir / f"{name}.json").write_text(json.dumps(config))

    def calls(self, name=None):
        calls_dir = self.stubs_dir / "calls"
        if not calls_dir.is_dir():
            return []
        calls = [json.loads(path.read_text()) for path in calls_dir.iterdir()]
        return [call for call in calls if name is None or call["name"] == name]

    def check_calls(self):
        return [call for call in self.calls() if call["name"] in ("golangci-lint", "pnpm", "npm", "just")]

    def reviewer_calls(self):
        return [call for call in self.calls() if call["name"] == "claude"]

    def clear_calls(self):
        for path in (self.stubs_dir / "calls").glob("*.json"):
            path.unlink()

    def run_hook(self, action, payload):
        process = subprocess.run(
            [sys.executable, str(HOOK_PATH), action],
            cwd=self.root,
            env=self.environment,
            input=json.dumps({"cwd": str(self.root), **payload}),
            text=True,
            capture_output=True,
            check=True,
        )
        return json.loads(process.stdout) if process.stdout.strip() else None

    def run_doctor(self):
        return subprocess.run(
            [sys.executable, str(HOOK_PATH), "doctor"],
            cwd=self.root,
            env=self.environment,
            stdin=subprocess.DEVNULL,
            text=True,
            capture_output=True,
        )

    # Runs the hook in this process, so a test can shrink the time limits.
    def stop_in_process(self, action, budget_seconds, reviewer_timeout_seconds, checks_timeout_seconds=None):
        payload = {"hook_event_name": "Stop", "session_id": "session-1", "last_assistant_message": "Done.", "cwd": str(self.root)}
        full_review = review_gate.FULL_REVIEWS[action]
        shrunk_review = replace(full_review, reviewer=replace(full_review.reviewer, timeout_seconds=reviewer_timeout_seconds))
        stdout = io.StringIO()
        with (
            mock.patch.dict(os.environ, self.environment, clear=True),
            mock.patch.dict(review_gate.FULL_REVIEWS, {action: shrunk_review}),
            mock.patch.object(review_gate, "FULL_REVIEW_BUDGET_SECONDS", budget_seconds),
            mock.patch.object(
                review_gate, "CHECKS_TOTAL_TIMEOUT_SECONDS", checks_timeout_seconds or review_gate.CHECKS_TOTAL_TIMEOUT_SECONDS
            ),
            mock.patch.object(sys, "argv", ["review_gate.py", action]),
            mock.patch.object(sys, "stdin", io.StringIO(json.dumps(payload))),
            mock.patch.object(sys, "stdout", stdout),
        ):
            review_gate.main()
        return json.loads(stdout.getvalue())

    def submit_prompt(self, prompt, session_id="session-1"):
        self.run_hook("snapshot", {"hook_event_name": "UserPromptSubmit", "session_id": session_id, "prompt": prompt})

    def stop(self, action, session_id="session-1", message="Done."):
        payload = {"hook_event_name": "Stop", "session_id": session_id, "last_assistant_message": message}
        return self.run_hook(action, payload)


def finding(category, summary, file="src/widgets.py:10"):
    return {"category": category, "file": file, "summary": summary}


def verification_section(*lines, fence="sh"):
    body = "".join(f"{line}\n" for line in lines)
    return f"\n## Change verification\n\n```{fence}\n{body}```\n"


def is_process_gone(pid):
    try:
        os.kill(pid, 0)
    except ProcessLookupError:
        return True
    try:
        return Path(f"/proc/{pid}/stat").read_text().split(")")[-1].split()[0] == "Z"
    except OSError:
        return True


def wait_until_process_gone(pid, timeout_seconds=5):
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
# and the reviewers use almost all of their limits.
TWO_WAVES_CHECKS_TIMEOUT_SECONDS = 2
TWO_WAVES_REVIEWER_TIMEOUT_SECONDS = 4
TWO_WAVES_BUDGET_SECONDS = TWO_WAVES_CHECKS_TIMEOUT_SECONDS + 2 * TWO_WAVES_REVIEWER_TIMEOUT_SECONDS + 2


class ShortReplyAssertions:
    def assert_asks_for_short_reply(self, reason):
        self.assertIn("one short line about your fixes that names every rejected finding with the reason", reason)
        self.assertIn("Do not write the summary for the user yet", reason)


class SummaryAssertions:
    def assert_asks_for_summary_without_review(self, reason):
        self.assertIn("as if there had been no review", reason)
        self.assertIn("Leave out review rounds, findings and the fixes made for them.", reason)
        self.assertNotIn("each round", reason)


class SessionStartTest(unittest.TestCase):
    def test_tells_agent_to_load_communication_skill(self):
        repository = HookRepository(self)

        response = repository.run_hook("session-start", {"hook_event_name": "SessionStart", "session_id": "session-1"})

        context = response["hookSpecificOutput"]["additionalContext"]
        self.assertIn(".agents/skills/humanizer/SKILL.md", context)


class Launcher:
    def __init__(self, test_case):
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
            (self.bin_dir / name).symlink_to(shutil.which(name))

        self.environment = {
            "PATH": str(self.bin_dir),
            "HOME": str(self.home),
            "REVIEW_GATE_STUBS": str(self.stubs_dir),
        }
        self.mise_answers = {}

    def install_mise(self, directory):
        directory.mkdir(parents=True, exist_ok=True)
        stub = directory / "mise"
        stub.write_text(f"#!{sys.executable}\n{MISE_STUB_SOURCE}")
        stub.chmod(0o755)

    def set_mise_answer(self, subcommand, **answer):
        self.mise_answers[subcommand] = answer
        (self.stubs_dir / "mise.json").write_text(json.dumps(self.mise_answers))

    def mise_calls(self, subcommand):
        calls_path = self.stubs_dir / "mise-calls.json"
        calls = json.loads(calls_path.read_text()) if calls_path.exists() else []
        return [call for call in calls if call["subcommand"] == subcommand]

    def run(self, *arguments, stdin=""):
        return subprocess.run(
            [str(LAUNCHER_PATH), *arguments],
            env=self.environment,
            input=stdin,
            text=True,
            capture_output=True,
        )


MISSING_OPENSPEC_LINE = "npm:@fission-ai/openspec  1.11.0 (missing)  mise.toml  1.11.0"


class LauncherTest(unittest.TestCase):
    def assert_stops_agent(self, process, *fragments):
        self.assertEqual(process.returncode, 0)
        response = json.loads(process.stdout)
        self.assertIs(response["continue"], False)
        for fragment in fragments:
            self.assertIn(fragment, response["stopReason"])
            self.assertIn(fragment, response["systemMessage"])
        return response

    def test_runs_hook_through_mise_from_home_local_bin_without_installs_when_path_has_no_mise(self):
        launcher = Launcher(self)
        launcher.install_mise(launcher.home / ".local" / "bin")
        payload = json.dumps({"hook_event_name": "Stop", "session_id": "session-1"})

        process = launcher.run("naming", stdin=payload)

        self.assertEqual(process.returncode, 0)
        self.assertEqual(json.loads(process.stdout), {})
        root = str(HOOK_PATH.parent.parent)
        [missing_check] = launcher.mise_calls("ls")
        self.assertEqual(
            missing_check["arguments"],
            ["-C", root, "ls", "--current", "--missing", "--no-truncate", "python", "npm:@fission-ai/openspec"],
        )
        [call] = launcher.mise_calls("exec")
        self.assertEqual(call["arguments"], ["-C", root, "exec", "--", "python3", str(HOOK_PATH), "naming"])
        self.assertEqual(call["exec_auto_install"], "false")
        self.assertEqual(call["stdin"], payload)

    def test_stops_agent_without_running_hook_when_pinned_version_is_missing(self):
        launcher = Launcher(self)
        launcher.install_mise(launcher.bin_dir)
        launcher.set_mise_answer("ls", stdout=f"{MISSING_OPENSPEC_LINE}\n")

        process = launcher.run("proposal")

        self.assert_stops_agent(process, "not installed", MISSING_OPENSPEC_LINE, "mise install", "just doctor")
        self.assertEqual(launcher.mise_calls("exec"), [])

    def test_stops_agent_without_running_hook_when_missing_tools_check_fails(self):
        launcher = Launcher(self)
        launcher.install_mise(launcher.bin_dir)
        launcher.set_mise_answer("ls", stdout="", stderr="mise ERROR Config file mise.toml is not trusted\n", exit_code=1)

        process = launcher.run("diff")

        self.assert_stops_agent(process, "mise exited with code 1", "is not trusted", "just doctor")
        self.assertEqual(launcher.mise_calls("exec"), [])

    def test_stops_agent_when_version_manager_is_missing(self):
        launcher = Launcher(self)

        process = launcher.run("diff")

        self.assert_stops_agent(process, "could not find the version manager mise", "just doctor")

    def test_stops_agent_with_action_and_exit_code_when_hook_crashes(self):
        launcher = Launcher(self)
        launcher.install_mise(launcher.bin_dir)
        launcher.set_mise_answer("exec", stdout="", stderr="Traceback: KeyError\n", exit_code=3)

        process = launcher.run("implementation")

        self.assert_stops_agent(process, "`implementation`", "exit code 3", "just doctor")

    def test_keeps_stop_message_valid_json_when_error_output_has_quotes_and_control_characters(self):
        launcher = Launcher(self)
        launcher.install_mise(launcher.bin_dir)
        launcher.set_mise_answer("exec", stdout="", stderr='KeyError: "baseline"\\ \t\x1b[31m\r\n', exit_code=1)

        process = launcher.run("diff")

        response = self.assert_stops_agent(process)
        self.assertIn('KeyError: "baseline"\\', response["stopReason"])

    def test_doctor_exits_with_code_of_environment_check(self):
        launcher = Launcher(self)
        launcher.install_mise(launcher.bin_dir)
        launcher.set_mise_answer("exec", stdout="openspec: not found\n", exit_code=1)

        process = launcher.run("doctor")

        self.assertEqual(process.returncode, 1)
        self.assertEqual(process.stdout, "openspec: not found\n")
        [call] = launcher.mise_calls("exec")
        self.assertEqual(call["arguments"][-2:], [str(HOOK_PATH), "doctor"])

    def test_doctor_exits_with_code_1_when_pinned_version_is_missing(self):
        launcher = Launcher(self)
        launcher.install_mise(launcher.bin_dir)
        launcher.set_mise_answer("ls", stdout=f"{MISSING_OPENSPEC_LINE}\n")

        process = launcher.run("doctor")

        self.assertEqual(process.returncode, 1)
        self.assertIn(MISSING_OPENSPEC_LINE, process.stderr)
        self.assertEqual(launcher.mise_calls("exec"), [])

    def test_doctor_exits_with_code_1_when_version_manager_is_missing(self):
        launcher = Launcher(self)

        process = launcher.run("doctor")

        self.assertEqual(process.returncode, 1)
        self.assertIn("could not find the version manager mise", process.stderr)


class MissingToolTest(unittest.TestCase):
    def test_stops_agent_naming_openspec_and_doctor_when_openspec_is_missing(self):
        repository = HookRepository(self, missing_tools=("openspec",))
        repository.submit_prompt("Write a spec for gadgets")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertIs(response["continue"], False)
        self.assertIn("could not find `openspec`", response["stopReason"])
        self.assertIn("just doctor", response["stopReason"])
        self.assertIn("just doctor", response["systemMessage"])
        self.assertEqual(repository.reviewer_calls(), [])


class DoctorTest(unittest.TestCase):
    def test_reports_path_of_every_tool_and_exits_with_code_0(self):
        repository = HookRepository(self, is_path_isolated=True)
        bin_dir = repository.root.parent / "bin"

        process = repository.run_doctor()

        self.assertEqual(process.returncode, 0, process.stdout)
        self.assertEqual(
            process.stdout.splitlines(),
            [
                f"python: {sys.executable}",
                f"git: {repository.root.parent / 'system-bin' / 'git'}",
                f"openspec: {bin_dir / 'openspec'}",
                f"claude: {bin_dir / 'claude'}",
            ],
        )

    def test_reports_missing_reviewer_cli_and_exits_with_nonzero_code(self):
        repository = HookRepository(self, missing_tools=("claude",))

        process = repository.run_doctor()

        self.assertNotEqual(process.returncode, 0)
        self.assertIn("claude: not found", process.stdout.splitlines())


class RunChecksTest(unittest.TestCase):
    def setUp(self):
        temporary_dir = tempfile.TemporaryDirectory(prefix="review-gate-checks-test-")
        self.addCleanup(temporary_dir.cleanup)
        self.root = Path(temporary_dir.name)

    def read_pid(self, name):
        return int((self.root / name).read_text())

    def test_stops_check_with_every_child_process_when_check_exceeds_its_limit(self):
        check = review_gate.Check(".", (sys.executable, "-c", CHECK_WITH_CHILD_SOURCE, "child.pid"))

        failure = review_gate.run_checks(self.root, [check], check_timeout_seconds=1, total_timeout_seconds=10)

        self.assertIn("did not finish within its limit of 1 seconds per check", failure)
        self.assertTrue(wait_until_process_gone(self.read_pid("child.pid")))

    def test_stops_running_check_and_names_total_limit_when_checks_together_exceed_it(self):
        checks = [
            review_gate.Check(".", (sys.executable, "-c", "import time; time.sleep(1.5)")),
            review_gate.Check(".", (sys.executable, "-c", CHECK_WITH_CHILD_SOURCE, "child.pid")),
        ]

        failure = review_gate.run_checks(self.root, checks, check_timeout_seconds=2, total_timeout_seconds=3)

        self.assertIn("did not finish within the total limit of 3 seconds for all checks", failure)
        self.assertIn("child.wait()", failure)
        self.assertTrue(wait_until_process_gone(self.read_pid("child.pid")))

    def test_names_exceeded_limit_when_child_outside_process_group_keeps_output_open(self):
        check = review_gate.Check(".", (sys.executable, "-c", CHECK_WITH_ESCAPED_CHILD_SOURCE, "child.pid"))
        self.addCleanup(lambda: is_process_gone(self.read_pid("child.pid")) or os.kill(self.read_pid("child.pid"), signal.SIGKILL))

        with mock.patch.object(review_gate, "KILLED_CHECK_OUTPUT_TIMEOUT_SECONDS", 0.5):
            failure = review_gate.run_checks(self.root, [check], check_timeout_seconds=1, total_timeout_seconds=10)

        self.assertIn("did not finish within its limit of 1 seconds per check", failure)
        self.assertIn("End of output:\n(no output)", failure)


class FullReviewBudgetTest(unittest.TestCase):
    def get_hook_timeouts(self, gate):
        config = json.loads((HOOK_PATH.parent.parent / ".claude" / "settings.json").read_text())
        return [
            hook["timeout"]
            for groups in config["hooks"].values()
            for group in groups
            for hook in group["hooks"]
            if hook["command"].endswith(f'run.sh\" {gate}')
        ]

    def test_budget_fits_checks_and_two_reviewer_waves(self):
        for full_review in review_gate.FULL_REVIEWS.values():
            self.assertGreaterEqual(
                review_gate.FULL_REVIEW_BUDGET_SECONDS,
                review_gate.CHECKS_TOTAL_TIMEOUT_SECONDS + 2 * full_review.reviewer.timeout_seconds,
            )

    def test_budget_ends_before_full_review_hook_timeouts(self):
        for gate in review_gate.FULL_REVIEWS:
            [timeout_seconds] = self.get_hook_timeouts(gate)
            self.assertLess(review_gate.FULL_REVIEW_BUDGET_SECONDS, timeout_seconds, gate)


class ChangeVerificationParsingTest(unittest.TestCase):
    def setUp(self):
        temporary_dir = tempfile.TemporaryDirectory(prefix="review-gate-verification-test-")
        self.addCleanup(temporary_dir.cleanup)
        self.change_dir = Path(temporary_dir.name) / CHANGE_NAME
        self.change_dir.mkdir()

    def parse(self, section):
        (self.change_dir / "tasks.md").write_text(f"# Tasks\n\n- [x] 1.1 Build\n{section}")
        return review_gate.parse_change_verification(self.change_dir)

    def assert_rejected(self, section, reason):
        with self.assertRaises(review_gate.ChangeVerificationError) as context:
            self.parse(section)
        message = str(context.exception)
        self.assertIn(reason, message)
        self.assertIn(f"`{CHANGE_NAME}/tasks.md`", message)
        self.assertIn(review_gate.ALLOWED_VERIFICATION_FORMS, message)

    def test_rejects_full_backend_suite(self):
        self.assert_rejected(verification_section("just test"), "`just test` is not an allowed command")

    def test_rejects_shell_operator_as_forbidden_character(self):
        self.assert_rejected(
            verification_section("just test-package ./internal/features/intervals/... && rm -rf /tmp/x"),
            "the argument `&&`",
        )

    def test_rejects_argument_with_forbidden_character(self):
        self.assert_rejected(
            verification_section("just test-package ./internal/a;rm"),
            "the argument `./internal/a;rm` in `just test-package ./internal/a;rm` has a forbidden character",
        )

    def test_runs_backend_package_tests_in_backend_directory(self):
        checks = self.parse(verification_section("just test-package ./internal/features/intervals/..."))

        self.assertEqual(checks, [review_gate.Check("backend", ("just", "test-package", "./internal/features/intervals/..."))])

    def test_accepts_every_allowed_form_and_skips_empty_and_comment_lines(self):
        checks = self.parse(
            verification_section(
                "# backend",
                "just test-package ./internal/features/intervals/... ./internal/features/backups/...",
                "",
                "just --justfile agent/verification/Justfile test",
                "pnpm --dir frontend test src/features/widgets --reporter=dot",
            )
        )

        self.assertEqual(
            checks,
            [
                review_gate.Check(
                    "backend", ("just", "test-package", "./internal/features/intervals/...", "./internal/features/backups/...")
                ),
                review_gate.Check(".", ("just", "--justfile", "agent/verification/Justfile", "test")),
                review_gate.Check(".", ("pnpm", "--dir", "frontend", "test", "src/features/widgets", "--reporter=dot")),
            ],
        )

    def test_rejects_backend_package_tests_without_package(self):
        self.assert_rejected(verification_section("just test-package"), "does not match the form `just test-package <package>...`")

    def test_rejects_package_path_without_leading_dot_slash(self):
        self.assert_rejected(verification_section("just test-package internal/features/intervals/..."), "must start with `./`")

    def test_rejects_arguments_after_verification_agent_form(self):
        self.assert_rejected(
            verification_section("just --justfile agent/verification/Justfile test ./internal/..."),
            "does not match the form `just --justfile agent/verification/Justfile test`",
        )

    def test_rejects_section_with_only_bash_fenced_block(self):
        self.assert_rejected(verification_section("pnpm --dir frontend test", fence="bash"), "the section has no ```sh code block")

    def test_rejects_unclosed_block(self):
        self.assert_rejected("\n## Change verification\n\n```sh\npnpm --dir frontend test\n", "the ```sh code block is not closed")

    def test_rejects_block_without_commands(self):
        self.assert_rejected(verification_section("# nothing yet", ""), "the ```sh code block holds no command")

    def test_returns_no_commands_without_section(self):
        self.assertEqual(self.parse(""), [])


class PlanGateTest(unittest.TestCase):
    def exit_plan_mode(self, repository, gate, plan):
        payload = {
            "hook_event_name": "PreToolUse",
            "session_id": "session-1",
            "tool_name": "ExitPlanMode",
            "tool_input": {"plan": plan},
        }
        return repository.run_hook(gate, payload)

    def test_allows_plan_when_reviewer_approves(self):
        repository = HookRepository(self)
        repository.submit_prompt("Plan the widgets")

        response = self.exit_plan_mode(repository, "plan", "# Plan\n\nAdd widgets.\n")

        self.assertEqual(response, {})
        [call] = repository.reviewer_calls()
        self.assertIn("## Plan review", call["prompt"])
        self.assertIn("Add widgets.", call["prompt"])

    def test_denies_plan_with_reviewer_reason_when_reviewer_rejects(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"ok": False, "reason": "The plan keeps a compatibility alias."})
        repository.submit_prompt("Plan the widgets")

        response = self.exit_plan_mode(repository, "necessity", "# Plan\n")

        output = response["hookSpecificOutput"]
        self.assertEqual(output["permissionDecision"], "deny")
        self.assertIn("The plan keeps a compatibility alias.", output["permissionDecisionReason"])
        self.assertIn("Revise the plan and show it again", output["permissionDecisionReason"])

    def test_denies_exit_plan_mode_without_plan_text(self):
        repository = HookRepository(self)
        repository.submit_prompt("Plan the widgets")

        response = self.exit_plan_mode(repository, "architecture", "")

        self.assertEqual(response["hookSpecificOutput"]["permissionDecision"], "deny")
        self.assertEqual(repository.reviewer_calls(), [])


class DiffGateTest(unittest.TestCase):
    def test_accepts_turn_when_reviewer_approves(self):
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        response = repository.stop("diff")

        self.assertEqual(response, {})
        self.assertEqual(len(repository.reviewer_calls()), 1)

    def test_blocks_turn_with_reviewer_reason_when_reviewer_rejects(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"ok": False, "reason": "No test for an empty list."})
        repository.submit_prompt("Add a widget")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        response = repository.stop("diff")

        self.assertEqual(response["decision"], "block")
        self.assertIn("No test for an empty list.", response["reason"])
        self.assertIn("a summary of the final state, without retelling this review", response["reason"])
        self.assertNotIn("Fix the result", response["reason"])

    def test_diff_reviewer_prompt_checks_agent_instructions(self):
        repository = HookRepository(self)
        repository.submit_prompt("Fix the docs")
        repository.write("AGENTS.md", "# AGENTS.md\n")

        repository.stop("diff")

        [call] = repository.reviewer_calls()
        self.assertIn("Check edits to agent instructions per the same section.", call["prompt"])

    def test_reviewer_prompt_maps_paths_to_module_docs(self):
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        repository.stop("diff")

        [call] = repository.reviewer_calls()
        self.assertIn("| `agent/**` | `backend/AGENTS.md` |", call["prompt"])

    def test_runs_claude_reviewer_on_opus_with_low_effort(self):
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        repository.stop("diff")

        [call] = repository.calls("claude")
        self.assertIn("--model opus --effort low", " ".join(call["arguments"]))
        self.assertEqual(call["effort_env"], "low")

    def test_stops_agent_after_fourth_rejection(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"ok": False, "reason": "No test for an empty list."})
        repository.submit_prompt("Add a widget")

        responses = []
        for attempt in range(4):
            repository.write("src/widgets.py", f"WIDGETS = [{attempt}]\n")
            responses.append(repository.stop("diff"))

        self.assertEqual([response.get("decision") for response in responses[:3]], ["block", "block", "block"])
        self.assertIs(responses[3]["continue"], False)


class ArchitectureGateTest(unittest.TestCase):
    def test_architecture_prompt_checks_http_contract_against_api_design_skill(self):
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


class FullReviewSchemaTest(unittest.TestCase):
    def test_reviewer_schemas_allow_exactly_the_categories_of_their_full_review(self):
        for full_review in (review_gate.IMPLEMENTATION_REVIEW, review_gate.PROPOSAL_REVIEW):
            with self.subTest(full_review.gate):
                schema = json.loads(full_review.reviewer.schema_path.read_text())
                category_enum = schema["properties"]["findings"]["items"]["properties"]["category"]["enum"]
                self.assertEqual(tuple(category_enum), full_review.categories)


class NamingGateTest(unittest.TestCase):
    def test_runs_reviewer_when_turn_changes_backend_go_file(self):
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")

        response = repository.stop("naming")

        self.assertEqual(response, {})
        [call] = repository.reviewer_calls()
        self.assertIn("## Naming review", call["prompt"])

    def test_runs_reviewer_when_turn_changes_agent_go_file(self):
        repository = HookRepository(self)
        repository.submit_prompt("Add a heartbeat field")
        repository.write("agent/verification/internal/heartbeat/heartbeat.go", "package heartbeat\n")

        repository.stop("naming")

        self.assertEqual(len(repository.reviewer_calls()), 1)

    def test_runs_reviewer_when_turn_changes_frontend_tsx_file(self):
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("frontend/src/pages/widgets/ui/WidgetsPage.tsx", "export {};\n")

        repository.stop("naming")

        self.assertEqual(len(repository.reviewer_calls()), 1)

    def test_runs_reviewer_when_turn_changes_website_tsx_file(self):
        repository = HookRepository(self)
        repository.submit_prompt("Add a page")
        repository.write("website/app/[lang]/pricing/page.tsx", "export {};\n")

        repository.stop("naming")

        self.assertEqual(len(repository.reviewer_calls()), 1)

    def test_blocks_turn_with_reviewer_reason_when_reviewer_rejects(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"ok": False, "reason": "Active() is a getter, name it GetRunningVerificationIDs()."})
        repository.submit_prompt("Add a widget")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")

        response = repository.stop("naming")

        self.assertEqual(response["decision"], "block")
        self.assertIn("GetRunningVerificationIDs", response["reason"])

    def test_skips_turn_that_changes_only_markdown(self):
        repository = HookRepository(self)
        repository.submit_prompt("Fix the docs")
        repository.write("backend/AGENTS.md", "# AGENTS.md\n")

        response = repository.stop("naming")

        self.assertEqual(response, {})
        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_turn_that_changes_only_frontend_config(self):
        repository = HookRepository(self)
        repository.submit_prompt("Bump Vite")
        repository.write("frontend/vite.config.ts", "export default {};\n")

        repository.stop("naming")

        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_markdown_turn_when_go_file_was_changed_before_request(self):
        repository = HookRepository(self)
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")
        repository.submit_prompt("Fix the docs")
        repository.write("backend/AGENTS.md", "# AGENTS.md\n")

        repository.stop("naming")

        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_turn_that_changes_only_ready_proposal(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("naming")

        self.assertEqual(response, {})
        self.assertEqual(repository.reviewer_calls(), [])

    def test_runs_claude_reviewer_on_opus_with_low_effort(self):
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")

        repository.stop("naming")

        [call] = repository.calls("claude")
        self.assertIn("--model opus --effort low", " ".join(call["arguments"]))

class ImplementationGateStartTest(unittest.TestCase):
    def test_runs_full_review_when_agent_completes_last_task(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual(len(repository.reviewer_calls()), 4)

    def test_skips_review_when_tasks_remain_unchecked(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_review_when_workspace_is_unchanged_during_turn(self):
        repository = HookRepository(self)
        repository.complete_tasks()
        repository.submit_prompt("What is left?")

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_review_in_new_session_when_workspace_matches_accepted_state(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue", session_id="first")
        repository.submit_prompt("Look at the tasks", session_id="second")
        repository.complete_tasks()
        repository.stop("implementation", session_id="first")
        repository.clear_calls()

        response = repository.stop("implementation", session_id="second")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_runs_review_again_when_files_change_after_accepted_review(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.complete_tasks()
        repository.stop("implementation")
        repository.clear_calls()
        repository.submit_prompt("Rename it")
        repository.write("src/widgets.py", "WIDGETS = ()\n")

        repository.stop("implementation")

        self.assertEqual(len(repository.reviewer_calls()), 4)

    def test_skips_review_when_prompt_has_openspec_planning_command(self):
        repository = HookRepository(self)
        repository.submit_prompt("/opsx:propose add widgets")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_archived_changes(self):
        repository = HookRepository(self)
        repository.git("rm", "-r", "--quiet", f"openspec/changes/{CHANGE_NAME}")
        repository.git("commit", "--quiet", "-m", "archive")
        repository.submit_prompt("Archive it")
        repository.write_change(spec_names=("widgets",), tasks="- [x] 1.1 Build\n", name=f"archive/2026-10-01-{CHANGE_NAME}")

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_returns_empty_response_without_payload(self):
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


class ImplementationGateChecksTest(ShortReplyAssertions, unittest.TestCase):
    def test_runs_only_backend_linter_when_change_touches_backend(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")
        repository.complete_tasks()

        repository.stop("implementation")

        [call] = repository.check_calls()
        self.assertEqual(call["name"], "golangci-lint")
        self.assertEqual(call["arguments"], ["run", "./cmd/...", "./internal/..."])
        self.assertEqual(Path(call["cwd"]), repository.root / "backend")

    def test_runs_agent_linter_in_verification_agent_directory(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("agent/verification/internal/heartbeat/heartbeat.go", "package heartbeat\n")
        repository.complete_tasks()

        repository.stop("implementation")

        [call] = repository.check_calls()
        self.assertEqual(call["arguments"], ["run", "./cmd/...", "./internal/...", "./e2e/..."])
        self.assertEqual(Path(call["cwd"]), repository.root / "agent" / "verification")

    def test_runs_only_frontend_linter_when_change_touches_frontend(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("frontend/src/pages/widgets/ui/WidgetsPage.tsx", "export {};\n")
        repository.complete_tasks()

        repository.stop("implementation")

        self.assertEqual([call["arguments"] for call in repository.check_calls()], [["--dir", "frontend", "lint"]])

    def test_runs_website_lint_and_build_when_change_touches_website(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("website/app/[lang]/pricing/page.tsx", "export {};\n")
        repository.complete_tasks()

        repository.stop("implementation")

        self.assertEqual(
            sorted(call["arguments"] for call in repository.calls("npm")),
            [["--prefix", "website", "run", "build"], ["--prefix", "website", "run", "lint"]],
        )

    def test_runs_no_checks_when_change_touches_no_module(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("docs/widgets.md", "# Widgets\n")
        repository.complete_tasks()

        repository.stop("implementation")

        self.assertEqual(repository.check_calls(), [])
        self.assertEqual(len(repository.reviewer_calls()), 4)

    def test_blocks_turn_with_output_tail_when_linter_fails(self):
        repository = HookRepository(self)
        repository.set_stub("golangci-lint", {"exit_code": 1, "output": "service.go:12:2: ineffectual assignment to err\n"})
        repository.submit_prompt("Continue")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn("`golangci-lint run ./cmd/... ./internal/...` in `backend`", response["reason"])
        self.assertIn("service.go:12:2: ineffectual assignment to err", response["reason"])
        self.assert_asks_for_short_reply(response["reason"])
        self.assertEqual(repository.reviewer_calls(), [])

    def test_never_runs_formatters_and_runs_just_only_from_change_verification_block(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")
        repository.write("frontend/src/pages/widgets/ui/WidgetsPage.tsx", "export {};\n")
        repository.complete_tasks(verification_section=verification_section("just test-package ./internal/features/widgets/..."))

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual([call["arguments"] for call in repository.calls("just")], [["test-package", "./internal/features/widgets/..."]])
        arguments = [" ".join(call["arguments"]) for call in repository.check_calls()]
        self.assertEqual(len(arguments), 3)
        self.assertFalse(any("fmt" in argument or "format" in argument for argument in arguments))
        self.assertEqual(len(repository.reviewer_calls()), 4)

    def test_blocks_turn_with_command_and_output_tail_when_package_test_fails(self):
        repository = HookRepository(self)
        repository.set_stub("just", {"exit_code": 1, "output": "--- FAIL: Test_CreateInterval_WhenOverlapping_ReturnsConflict\n"})
        repository.submit_prompt("Continue")
        repository.write("backend/internal/features/intervals/service.go", "package intervals\n")
        repository.complete_tasks(verification_section=verification_section("just test-package ./internal/features/intervals/..."))

        response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn("`just test-package ./internal/features/intervals/...` in `backend` exited with code 1", response["reason"])
        self.assertIn("--- FAIL: Test_CreateInterval_WhenOverlapping_ReturnsConflict", response["reason"])
        self.assert_asks_for_short_reply(response["reason"])
        self.assertEqual(repository.reviewer_calls(), [])

    def test_runs_backend_package_tests_in_backend_directory_after_linter(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("backend/internal/features/intervals/service.go", "package intervals\n")
        repository.complete_tasks(verification_section=verification_section("just test-package ./internal/features/intervals/..."))

        repository.stop("implementation")

        [linter_call] = repository.calls("golangci-lint")
        [test_call] = repository.calls("just")
        self.assertEqual(test_call["arguments"], ["test-package", "./internal/features/intervals/..."])
        self.assertEqual(Path(test_call["cwd"]), repository.root / "backend")
        self.assertLess(linter_call["started_at"], test_call["started_at"])

    def test_runs_command_named_by_two_changes_once(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.complete_tasks(verification_section=verification_section("pnpm --dir frontend test"))
        repository.write_change(spec_names=("gizmos",), tasks="- [x] 1.1 Build\n" + verification_section("pnpm --dir frontend test"), name="add-gizmos")

        repository.stop("implementation")

        self.assertEqual([call["arguments"] for call in repository.calls("pnpm")], [["--dir", "frontend", "test"]])

    def test_starts_reviewers_without_test_run_when_change_has_no_verification_section(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("frontend/src/pages/widgets/ui/WidgetsPage.tsx", "export {};\n")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual([call["arguments"] for call in repository.check_calls()], [["--dir", "frontend", "lint"]])
        self.assertEqual(len(repository.reviewer_calls()), 4)

    def test_blocks_turn_before_any_check_when_verification_section_is_malformed(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("backend/internal/features/intervals/service.go", "package intervals\n")
        repository.complete_tasks(verification_section=verification_section("just test"))

        response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn("`just test` is not an allowed command", response["reason"])
        self.assertIn(review_gate.ALLOWED_VERIFICATION_FORMS, response["reason"])
        self.assertIn("No check or reviewer ran.", response["reason"])
        self.assertEqual(repository.check_calls(), [])
        self.assertEqual(repository.reviewer_calls(), [])

    def test_runs_hook_tests_when_change_touches_hooks(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write(".agent-hooks/sample_test.py", PASSING_HOOK_TEST)
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual(len(repository.reviewer_calls()), 4)

    def test_blocks_turn_when_hook_tests_fail(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write(".agent-hooks/sample_test.py", FAILING_HOOK_TEST)
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn("gate returned block for an accepted turn", response["reason"])
        self.assertEqual(repository.reviewer_calls(), [])


class ImplementationGateReviewTest(ShortReplyAssertions, unittest.TestCase):
    def test_runs_reviewer_per_spec_and_for_architecture_docs_and_tests_on_claude_with_medium_effort(self):
        repository = HookRepository(self)
        repository.write_change(spec_names=("widgets", "gadgets"), tasks="- [x] 1.1 Build\n")
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        repository.stop("implementation")

        calls = repository.calls("claude")
        self.assertEqual(len(calls), 5)
        for call in calls:
            self.assertIn("--model opus --effort medium", " ".join(call["arguments"]))
            self.assertEqual(call["effort_env"], "medium")
        roles = sorted(line for call in calls for line in call["prompt"].splitlines() if line.startswith("Role:"))
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

    def test_implementation_architecture_reviewer_prompt_checks_layout_contract_and_exempts_old_code(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        repository.stop("implementation")

        [call] = [call for call in repository.reviewer_calls() if "\nRole: architecture\n" in call["prompt"]]
        prompt = call["prompt"]
        architecture_role = prompt[prompt.index("### Role: architecture") : prompt.index("### Role: documentation and tasks")]
        self.assertIn("Read `docs/architecture.md` completely", architecture_role)
        self.assertIn("including parts committed or written in earlier requests", architecture_role)
        self.assertIn(
            "When the change edits an HTTP contract file under `backend/api/`, read `.agents/skills/api-design/SKILL.md`",
            architecture_role,
        )
        self.assertIn(
            'Go code under `backend/` and `agent/` outside the layout block in "Backend layout" of `docs/architecture.md`',
            architecture_role,
        )
        self.assertIn("The exemption does not cover the frontend.", architecture_role)

    def test_implementation_docs_reviewer_prompt_checks_agent_instructions(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        repository.stop("implementation")

        [call] = [call for call in repository.reviewer_calls() if "\nRole: documentation and tasks\n" in call["prompt"]]
        prompt = call["prompt"]
        docs_role = prompt[prompt.index("### Role: documentation and tasks") : prompt.index("### Role: tests")]
        self.assertIn("Check the edits to agent instructions the change made", docs_role)

    def test_tests_reviewer_prompt_requires_failing_tests_reproduced_bugs_waits_and_covered_packages(self):
        repository = HookRepository(self)
        repository.write_change(spec_names=("widgets", "gadgets"), tasks="- [x] 1.1 Build\n")
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        repository.stop("implementation")

        self.assertEqual(len(repository.reviewer_calls()), 5)
        [call] = [call for call in repository.reviewer_calls() if "\nRole: tests\n" in call["prompt"]]
        prompt = call["prompt"]
        tests_role = prompt[prompt.index("### Role: tests") : prompt.index("### Role: repeat review")]
        self.assertIn("`.agents/skills/testing/SKILL.md`", tests_role)
        self.assertIn("The test fails without the behavior it pins.", tests_role)
        self.assertIn("A bug fix has a test that reproduces the bug and fails on the code before the fix.", tests_role)
        self.assertIn("waits for a condition up to a limit instead of a fixed pause", tests_role)
        self.assertIn(
            "run the tests of every backend package, verification agent package and frontend test file in which the change added or changed tests",
            tests_role,
        )
        self.assertIn("such as the hook tests in `.agent-hooks/`, count as covered", tests_role)

    def test_blocks_turn_with_budget_reason_when_third_wave_of_reviewers_does_not_fit(self):
        repository = HookRepository(self)
        repository.set_stub("claude", {"sleep_seconds": THIRD_WAVE_REVIEWER_SECONDS})
        repository.write_change(spec_names=("widgets", *(f"spec{index}" for index in range(10))), tasks="- [x] 1.1 Build\n")
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop_in_process("implementation", THIRD_WAVE_BUDGET_SECONDS, THIRD_WAVE_REVIEWER_TIMEOUT_SECONDS)

        self.assertEqual(response["decision"], "block")
        self.assertIn(review_gate.BUDGET_EXHAUSTED_ERROR, response["reason"])
        self.assertEqual(len(repository.reviewer_calls()), 12)

    def test_runs_two_waves_of_reviewers_after_checks_use_their_total_limit(self):
        repository = HookRepository(self)
        repository.set_stub("golangci-lint", {"sleep_seconds": TWO_WAVES_CHECKS_TIMEOUT_SECONDS - 0.1})
        repository.set_stub("claude", {"sleep_seconds": TWO_WAVES_REVIEWER_TIMEOUT_SECONDS - 0.2})
        repository.write_change(spec_names=("widgets", *(f"spec{index}" for index in range(4))), tasks="- [x] 1.1 Build\n")
        repository.submit_prompt("Continue")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")
        repository.complete_tasks()

        response = repository.stop_in_process(
            "implementation",
            TWO_WAVES_BUDGET_SECONDS,
            TWO_WAVES_REVIEWER_TIMEOUT_SECONDS,
            checks_timeout_seconds=TWO_WAVES_CHECKS_TIMEOUT_SECONDS,
        )

        self.assertEqual(response, {})
        self.assertEqual(len(repository.calls("golangci-lint")), 1)
        self.assertEqual(len(repository.reviewer_calls()), 8)

    def test_blocks_turn_and_requires_agent_to_resolve_findings_when_reviewer_finds_bug(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [finding("bug", "Race between the check and the write.")]})
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn("[bug] src/widgets.py:10: Race between the check and the write.", response["reason"])
        self.assertIn("without asking the user", response["reason"])
        self.assert_asks_for_short_reply(response["reason"])

    def test_blocks_turn_and_asks_to_move_code_or_change_architecture_doc_when_reviewer_finds_architecture_violation(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [finding("architecture", "The service imports another module's repository.")]})
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn("[architecture] src/widgets.py:10: The service imports another module's repository.", response["reason"])
        self.assertIn(
            "On an architecture finding, move the code to follow docs/architecture.md, "
            "or change docs/architecture.md in the same change when the rule itself is wrong.",
            response["reason"],
        )
        self.assert_asks_for_short_reply(response["reason"])

    def test_marks_minor_findings_optional_when_turn_is_blocked(self):
        repository = HookRepository(self)
        repository.set_reviewer_result(
            {"findings": [finding("docs", "The README does not mention the command."), finding("minor", "Long function.")]}
        )
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertIn("[minor] (optional) src/widgets.py:10: Long function.", response["reason"])

    def test_accepts_change_and_shows_minor_findings_to_user(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [finding("minor", "Long function.")]})
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")
        repository.clear_calls()
        repeated_response = repository.stop("implementation")

        self.assertNotIn("decision", response)
        self.assertIn("Long function.", response["systemMessage"])
        self.assertEqual(repeated_response, {})
        self.assertEqual(repository.calls(), [])

    def test_blocks_turn_when_reviewer_answer_does_not_match_schema(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [{"category": "typo", "file": "a", "summary": "b"}]})
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn("does not match the schema", response["reason"])

    def test_blocks_turn_with_failure_reason_when_reviewer_crashes(self):
        repository = HookRepository(self)
        repository.set_reviewer_failure("rate limit exceeded")
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn("rate limit exceeded", response["reason"])


class ImplementationGateRepeatTest(SummaryAssertions, unittest.TestCase):
    def block_with_bug(self, repository):
        repository.set_reviewer_result({"findings": [finding("bug", "Race between the check and the write.")]})
        repository.submit_prompt("Continue")
        repository.write("src/widgets.py", "WIDGETS = []\n")
        repository.write("src/gadgets.py", "GADGETS = []\n")
        repository.complete_tasks()
        repository.stop("implementation")
        repository.clear_calls()

    def test_repeat_runs_one_reviewer_with_previous_findings_and_paths_changed_after_review(self):
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

    def test_accepted_repeat_clears_findings(self):
        repository = HookRepository(self)
        self.block_with_bug(repository)
        repository.set_reviewer_result({"findings": []})
        repository.write("src/widgets.py", "WIDGETS = ()\n")

        repository.stop("implementation")
        repository.write("src/widgets.py", "WIDGETS = (1,)\n")
        repository.clear_calls()
        repository.stop("implementation")

        self.assertEqual(len(repository.reviewer_calls()), 4)

    def test_asks_for_summary_without_review_once_after_repeat_accepts_change(self):
        repository = HookRepository(self)
        self.block_with_bug(repository)
        repository.set_reviewer_result({"findings": [finding("minor", "Long function.")]})
        repository.write("src/widgets.py", "WIDGETS = ()\n")

        response = repository.stop("implementation")
        repository.clear_calls()
        summary_response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn("accepted the change", response["reason"])
        self.assertIn("summary of the implementation", response["reason"])
        self.assert_asks_for_summary_without_review(response["reason"])
        self.assertIn("Long function.", response["reason"])
        self.assertEqual(summary_response, {})
        self.assertEqual(repository.calls(), [])

    def test_stops_agent_with_remaining_findings_after_third_block(self):
        repository = HookRepository(self)
        self.block_with_bug(repository)
        repository.write("src/widgets.py", "WIDGETS = ()\n")
        second_response = repository.stop("implementation")
        repository.write("src/widgets.py", "WIDGETS = (1,)\n")

        third_response = repository.stop("implementation")

        self.assertEqual(second_response["decision"], "block")
        self.assertIs(third_response["continue"], False)
        self.assertIn("Race between the check and the write.", third_response["stopReason"])


class NecessityDuringImplementationReviewTest(unittest.TestCase):
    def block_with_spec_mismatch(self, repository):
        repository.set_reviewer_result({"findings": [finding("spec_mismatch", "The spec requires 409, the code answers 400.")]})
        repository.submit_prompt("Continue")
        repository.complete_tasks()
        repository.stop("implementation")
        repository.clear_calls()

    def test_skips_necessity_when_agent_fixes_delta_spec_after_finding(self):
        repository = HookRepository(self)
        self.block_with_spec_mismatch(repository)
        repository.write(f"openspec/changes/{CHANGE_NAME}/specs/widgets/spec.md", "# widgets\n\nAnswers 400.\n")

        response = repository.stop("necessity")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_runs_necessity_when_prompt_has_openspec_planning_command(self):
        repository = HookRepository(self)
        self.block_with_spec_mismatch(repository)
        repository.submit_prompt("/opsx:propose a new change")
        repository.write(f"openspec/changes/{CHANGE_NAME}/specs/widgets/spec.md", "# widgets\n\nAnswers 400.\n")

        repository.stop("necessity")

        self.assertEqual(len(repository.reviewer_calls()), 1)


class ProposalGateStartTest(unittest.TestCase):
    def test_runs_full_review_when_agent_writes_all_artifacts_without_command(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec for gadgets")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(len(repository.reviewer_calls()), 6)

    def test_skips_review_without_calling_openspec_when_tasks_are_missing(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec for gadgets")
        repository.write(f"openspec/changes/{PROPOSAL_NAME}/proposal.md", "# Proposal\n")

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_review_without_calling_openspec_when_tasks_file_has_no_tasks(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec for gadgets")
        repository.write_proposal(tasks="")

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_review_when_openspec_reports_incomplete_artifacts(self):
        repository = HookRepository(self)
        repository.set_stub("openspec", {"status": {"output": json.dumps({"isPlanningComplete": False})}})
        repository.submit_prompt("Write a spec for gadgets")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_review_when_openspec_status_fails(self):
        repository = HookRepository(self)
        repository.set_stub("openspec", {"status": {"exit_code": 1, "output": "openspec: unknown change"}})
        repository.submit_prompt("Write a spec for gadgets")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_review_without_calling_openspec_when_task_is_checked(self):
        repository = HookRepository(self)
        repository.submit_prompt("Fix the widgets spec")
        repository.write(f"openspec/changes/{CHANGE_NAME}/specs/widgets/spec.md", "# widgets\n\nAnswers 409.\n")

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_review_when_change_files_are_unchanged_during_turn(self):
        repository = HookRepository(self)
        repository.write_proposal()
        repository.submit_prompt("What is in the proposal?")

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_review_in_new_session_when_files_match_accepted_state(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec", session_id="first")
        repository.submit_prompt("Look at the spec", session_id="second")
        repository.write_proposal()
        repository.stop("proposal", session_id="first")
        repository.clear_calls()

        response = repository.stop("proposal", session_id="second")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_runs_review_again_when_design_changes_after_accepted_review(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()
        repository.stop("proposal")
        repository.clear_calls()
        repository.submit_prompt("Fix the design")
        repository.write(f"openspec/changes/{PROPOSAL_NAME}/design.md", "# Design\n\nIn-memory cache.\n")

        repository.stop("proposal")

        self.assertEqual(len(repository.reviewer_calls()), 6)

    def test_skips_archived_changes(self):
        repository = HookRepository(self)
        repository.submit_prompt("Archive it")
        repository.write_change(spec_names=("gadgets",), tasks="- [ ] 1.1 Build\n", name=f"archive/2026-10-01-{PROPOSAL_NAME}")

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])


class ProposalGateValidationTest(unittest.TestCase):
    def test_blocks_turn_with_validation_output_when_requirement_has_no_scenario(self):
        repository = HookRepository(self)
        repository.set_stub(
            "openspec",
            {"validate": {"exit_code": 1, "output": '✗ [ERROR] gadgets/spec.md: ADDED "Gadgets" must include at least one scenario\n'}},
        )
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn(f"`openspec validate {PROPOSAL_NAME} --strict`", response["reason"])
        self.assertIn("must include at least one scenario", response["reason"])
        self.assertEqual(repository.reviewer_calls(), [])

    def test_blocks_turn_without_running_command_when_verification_section_has_forbidden_command(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal(tasks="- [ ] 1.1 Build\n" + verification_section("just test"))

        response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn("`just test` is not an allowed command", response["reason"])
        self.assertIn(review_gate.ALLOWED_VERIFICATION_FORMS, response["reason"])
        self.assertEqual(repository.calls("just"), [])
        self.assertEqual([call["arguments"][0] for call in repository.calls("openspec")], ["status"])
        self.assertEqual(repository.reviewer_calls(), [])


class ProposalGateReviewTest(ShortReplyAssertions, SummaryAssertions, unittest.TestCase):
    def test_runs_reviewer_per_spec_and_one_per_proposal_role_on_claude_with_medium_effort(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal(spec_names=("gadgets", "gizmos"))

        repository.stop("proposal")

        calls = repository.calls("claude")
        self.assertEqual(len(calls), 7)
        for call in calls:
            self.assertIn("--model opus --effort medium", " ".join(call["arguments"]))
            self.assertEqual(call["effort_env"], "medium")
        roles = sorted(line for call in calls for line in call["prompt"].splitlines() if line.startswith("Role:"))
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

    def test_runs_seven_reviewers_of_two_spec_proposal_at_most_six_at_a_time(self):
        reviewer_seconds = 1
        repository = HookRepository(self)
        repository.set_stub("claude", {"sleep_seconds": reviewer_seconds})
        repository.submit_prompt("Write a spec")
        repository.write_proposal(spec_names=("gadgets", "gizmos"))

        repository.stop("proposal")

        reviewer_starts_ns = sorted(call["started_at"] for call in repository.reviewer_calls())
        self.assertEqual(len(reviewer_starts_ns), 7)
        self.assertEqual(review_gate.MAX_PARALLEL_REVIEWERS, 6)
        first_wave_spread_seconds = (reviewer_starts_ns[5] - reviewer_starts_ns[0]) / 1e9
        seventh_reviewer_delay_seconds = (reviewer_starts_ns[6] - reviewer_starts_ns[0]) / 1e9
        self.assertLess(first_wave_spread_seconds, reviewer_seconds)
        self.assertGreaterEqual(seventh_reviewer_delay_seconds, reviewer_seconds)

    def get_proposal_role_section(self, repository, role, next_role):
        [call] = [call for call in repository.reviewer_calls() if f"\nRole: {role}\n" in call["prompt"]]
        prompt = call["prompt"]
        return prompt[prompt.index(f"### Role: {role}") : prompt.index(f"### Role: {next_role}")]

    def test_necessity_reviewer_prompt_returns_simplify_or_scope(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        repository.stop("proposal")

        necessity_role = self.get_proposal_role_section(repository, "necessity", "architecture")
        self.assertIn('"The best code is unwritten"', necessity_role)
        self.assertIn("Return a `simplify` finding when the user loses nothing", necessity_role)
        self.assertIn("a `scope` finding when the user loses a feature, screen, scenario or setting", necessity_role)

    def test_architecture_reviewer_prompt_checks_design_and_operations_and_exempts_old_code(self):
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
        self.assertIn(
            'Go code under `backend/` and `agent/` outside the layout block in "Backend layout" of `docs/architecture.md`',
            architecture_role,
        )
        self.assertIn("The exemption does not cover the frontend.", architecture_role)

    def test_names_reviewer_prompt_checks_naming_rules_data_section_and_glossary(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        repository.stop("proposal")

        names_role = self.get_proposal_role_section(repository, "names", "proposal and design")
        self.assertIn('the sections "Naming" and "Functions, methods and types" of `AGENTS.md`', names_role)
        self.assertIn('the table naming rules in the "Data" section of `docs/architecture.md`', names_role)
        self.assertIn("is a `mismatch` finding that names the rule and suggests a name", names_role)
        self.assertIn("`docs/glossary.md` does not define, while the change does not add it, is a `gap` finding", names_role)

    def test_tasks_reviewer_prompt_requires_every_scenario_covered_by_task_with_test(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        repository.stop("proposal")

        [call] = [call for call in repository.reviewer_calls() if "\nRole: tasks\n" in call["prompt"]]
        self.assertIn("every scenario of the delta specs is covered by a task with a test", call["prompt"])
        self.assertIn("the task order can be followed", call["prompt"])

    def test_tasks_reviewer_prompt_requires_verification_section_for_tested_packages_except_hook_tests(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        repository.stop("proposal")

        [call] = [call for call in repository.reviewer_calls() if "\nRole: tasks\n" in call["prompt"]]
        prompt = call["prompt"]
        tasks_role = prompt[prompt.index("### Role: tasks") : prompt.index("### Role: repeat review")]
        self.assertIn(
            "tasks add or change tests in a backend package, a verification agent package or the frontend has a `## Change verification` section",
            tasks_role,
        )
        self.assertIn("run the tests of every such package the tasks' verifications name", tasks_role)
        self.assertIn("a package the section leaves out, is a `tasks` finding that names the package", tasks_role)
        self.assertIn("such as the hook tests in `.agent-hooks/`, need no section", tasks_role)

    def test_blocks_turn_with_budget_reason_when_third_wave_of_reviewers_does_not_fit(self):
        repository = HookRepository(self)
        repository.set_stub("claude", {"sleep_seconds": THIRD_WAVE_REVIEWER_SECONDS})
        repository.submit_prompt("Write a spec")
        repository.write_proposal(spec_names=tuple(f"spec{index}" for index in range(11)))

        response = repository.stop_in_process("proposal", THIRD_WAVE_BUDGET_SECONDS, THIRD_WAVE_REVIEWER_TIMEOUT_SECONDS)

        self.assertEqual(response["decision"], "block")
        self.assertIn(review_gate.BUDGET_EXHAUSTED_ERROR, response["reason"])
        self.assertEqual(len(repository.reviewer_calls()), 12)

    def test_runs_two_waves_of_reviewers_after_validation_uses_checks_total_limit(self):
        repository = HookRepository(self)
        repository.set_stub("openspec", {"validate": {"sleep_seconds": TWO_WAVES_CHECKS_TIMEOUT_SECONDS - 0.1}})
        repository.set_stub("claude", {"sleep_seconds": TWO_WAVES_REVIEWER_TIMEOUT_SECONDS - 0.2})
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

    def test_blocks_turn_and_requires_agent_to_resolve_simplify_finding_without_asking_user(self):
        repository = HookRepository(self)
        repository.set_reviewer_result(
            {"findings": [finding("simplify", "A field in an existing table replaces the new table.", file="design.md:12")]}
        )
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn("[simplify] design.md:12: A field in an existing table replaces the new table.", response["reason"])
        self.assertIn("without asking the user", response["reason"])
        self.assert_asks_for_short_reply(response["reason"])

    def test_asks_for_summary_with_scope_finding_without_requiring_fix(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [finding("scope", "The settings screen can be dropped.", file="proposal.md:5")]})
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("proposal")
        repository.clear_calls()
        summary_response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn("Full proposal review accepted the proposal.", response["reason"])
        self.assertIn("[scope] (the user decides) proposal.md:5: The settings screen can be dropped.", response["reason"])
        self.assertIn("summary of the proposal", response["reason"])
        self.assert_asks_for_summary_without_review(response["reason"])
        self.assertIn("what the user loses and what they save", response["reason"])
        self.assertEqual(summary_response, {})
        self.assertEqual(repository.calls(), [])

    def test_accepts_proposal_and_shows_minor_findings_to_user(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [finding("minor", "Long paragraph.", file="design.md:3")]})
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("proposal")
        repository.clear_calls()
        repeated_response = repository.stop("proposal")

        self.assertNotIn("decision", response)
        self.assertIn("Long paragraph.", response["systemMessage"])
        self.assertEqual(repeated_response, {})
        self.assertEqual(repository.calls(), [])

    def test_blocks_turn_when_reviewer_uses_implementation_category(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [finding("bug", "Race.")]})
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn("does not match the schema", response["reason"])


class ProposalGateRepeatTest(ShortReplyAssertions, SummaryAssertions, unittest.TestCase):
    def block_with_gap(self, repository):
        repository.set_reviewer_result({"findings": [finding("gap", "The answer for an empty list is missing.", file="specs/gadgets/spec.md:8")]})
        repository.submit_prompt("Write a spec")
        repository.write_proposal()
        repository.stop("proposal")
        repository.clear_calls()

    def test_blocks_turn_with_gap_and_asks_for_short_reply_without_summary(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [finding("gap", "The answer for an empty list is missing.", file="specs/gadgets/spec.md:8")]})
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn("[gap] specs/gadgets/spec.md:8: The answer for an empty list is missing.", response["reason"])
        self.assertIn("without asking the user", response["reason"])
        self.assert_asks_for_short_reply(response["reason"])

    def test_repeat_runs_one_reviewer_with_previous_findings_and_files_changed_after_review(self):
        repository = HookRepository(self)
        self.block_with_gap(repository)
        repository.write(f"openspec/changes/{PROPOSAL_NAME}/specs/gadgets/spec.md", "# gadgets\n\nEmpty list.\n")

        repository.stop("proposal")

        [call] = repository.reviewer_calls()
        self.assertIn("Role: repeat review", call["prompt"])
        self.assertIn("The answer for an empty list is missing.", call["prompt"])
        changed_paths = call["prompt"].split("Paths changed after the previous review:")[1]
        self.assertIn(f"- openspec/changes/{PROPOSAL_NAME}/specs/gadgets/spec.md", changed_paths)
        self.assertNotIn("design.md", changed_paths)

    def test_asks_for_summary_without_review_once_after_repeat_accepts_proposal(self):
        repository = HookRepository(self)
        self.block_with_gap(repository)
        repository.set_reviewer_result({"findings": []})
        repository.write(f"openspec/changes/{PROPOSAL_NAME}/specs/gadgets/spec.md", "# gadgets\n\nEmpty list.\n")

        response = repository.stop("proposal")
        repository.clear_calls()
        summary_response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn("accepted the proposal", response["reason"])
        self.assertIn("summary of the proposal", response["reason"])
        self.assert_asks_for_summary_without_review(response["reason"])
        self.assertEqual(summary_response, {})
        self.assertEqual(repository.calls(), [])

    def test_lists_scope_option_in_summary_without_review_when_repeat_accepts_proposal(self):
        repository = HookRepository(self)
        self.block_with_gap(repository)
        repository.set_reviewer_result({"findings": [finding("scope", "The settings screen can be dropped.", file="proposal.md:5")]})
        repository.write(f"openspec/changes/{PROPOSAL_NAME}/specs/gadgets/spec.md", "# gadgets\n\nEmpty list.\n")

        response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn("[scope] (the user decides) proposal.md:5: The settings screen can be dropped.", response["reason"])
        self.assertIn("List every scope finding separately: what the user loses and what they save", response["reason"])
        self.assert_asks_for_summary_without_review(response["reason"])

    def test_stops_agent_with_remaining_findings_after_third_block(self):
        repository = HookRepository(self)
        self.block_with_gap(repository)
        repository.write(f"openspec/changes/{PROPOSAL_NAME}/design.md", "# Design\n\nSecond attempt.\n")
        second_response = repository.stop("proposal")
        repository.write(f"openspec/changes/{PROPOSAL_NAME}/design.md", "# Design\n\nThird attempt.\n")

        third_response = repository.stop("proposal")

        self.assertEqual(second_response["decision"], "block")
        self.assertIs(third_response["continue"], False)
        self.assertIn("Full proposal review", third_response["stopReason"])
        self.assertIn("The answer for an empty list is missing.", third_response["stopReason"])


class QuickGatesDuringProposalReviewTest(unittest.TestCase):
    def test_skips_quick_checks_when_turn_changes_only_ready_proposal(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        responses = [repository.stop(gate) for gate in ("necessity", "architecture", "diff")]

        self.assertEqual(responses, [{}, {}, {}])
        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_quick_checks_in_summary_turn_after_accepted_review(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [finding("scope", "The settings screen can be dropped.")]})
        repository.submit_prompt("Write a spec")
        repository.write_proposal()
        repository.stop("proposal")
        repository.clear_calls()

        responses = [repository.stop(gate, message="Proposal summary.") for gate in ("necessity", "architecture", "diff")]

        self.assertEqual(responses, [{}, {}, {}])
        self.assertEqual(repository.reviewer_calls(), [])

    def test_runs_necessity_when_agent_wrote_only_proposal(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write(f"openspec/changes/{PROPOSAL_NAME}/proposal.md", "# Proposal\n")

        repository.stop("necessity", message="Is the settings screen needed?")

        [call] = repository.reviewer_calls()
        self.assertIn("Under review: OpenSpec proposal", call["prompt"])

    def test_runs_quick_checks_without_calling_openspec_when_code_changes_too(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec and start")
        repository.write_proposal()
        repository.write("src/gadgets.py", "GADGETS = []\n")

        response = repository.stop("diff")

        self.assertEqual(response, {})
        self.assertEqual(len(repository.reviewer_calls()), 1)
        self.assertEqual(repository.calls("openspec"), [])


if __name__ == "__main__":
    unittest.main()
