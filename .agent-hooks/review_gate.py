#!/usr/bin/env python3
"""Review hooks for Claude Code, wired in .claude/settings.json.

Usage: review_gate.py <action>
       review_gate.py doctor

Claude Code starts it through run.sh, which runs it with the Python and OpenSpec versions pinned
in mise.toml and stops the agent with the reason when the hook cannot start or crashes.

Actions:
  session-start  SessionStart: tells the agent to load the communication skill in a fresh context.
  snapshot       UserPromptSubmit: remembers the request and the working tree it started from.
  plan           Reviews a plan before the user sees it.
  necessity      Checks a plan or an OpenSpec proposal: is the task needed, is there a much simpler option.
  diff           Reviews a finished change.
  architecture   Checks a plan or a change against docs/architecture.md and the architecture rules of the module docs.
  naming         Checks names in code the request changed against the naming rules of AGENTS.md
                 and the module docs. Runs when a .go, .ts or .tsx file under backend/, agent/,
                 frontend/src/ or website/ changed.
  implementation Full review of an OpenSpec change whose tasks are all checked: first the read-only
                 checks of the touched modules, then the commands of the "Change verification"
                 section of tasks.md, then parallel reviewers: one per delta spec, and one each for
                 architecture, documentation and tasks, and tests; after a block a repeat over the
                 previous findings only.
  proposal       Full review of a finished OpenSpec proposal with no checked task: first the format
                 check of the "Change verification" section without running it, then
                 openspec validate --strict, then parallel reviewers: one per delta spec, and one each
                 for necessity, architecture, names, proposal and design, and tasks. While it runs,
                 necessity, diff, architecture and naming skip a turn that only wrote the artifacts.
  doctor         Reports whether Python, git, openspec and a reviewer CLI are found, one line per tool,
                 and exits with a non-zero code when one is missing. `just doctor` runs it.

Both full reviews run within FULL_REVIEW_BUDGET_SECONDS from the hook's start: the checks share
one total limit, and a reviewer starts only when the budget still holds its whole limit.

Every review runs a separate read-only process of the same agent.
The review prompt lives in .agent-hooks/<action>-prompt.md, the shared part in common-prompt.md:
they say what the reviewer reads and checks.
A review that could not run never accepts the result, and a missing openspec stops the agent.
"""

import argparse
from concurrent.futures import ThreadPoolExecutor
from collections.abc import Callable
from dataclasses import dataclass, replace
import hashlib
import json
import os
from pathlib import Path
import re
import shlex
import shutil
import signal
import subprocess
import sys
import time


AUTOMATIC_REVIEW_PREFIX = "AUTOMATIC REVIEW:"
MAX_REVISIONS = 3
MAX_FULL_REVIEW_REVISIONS = 2
MAX_PARALLEL_REVIEWERS = 6
CHECK_TIMEOUT_SECONDS = 900
CHECKS_TOTAL_TIMEOUT_SECONDS = 1800
KILLED_CHECK_OUTPUT_TIMEOUT_SECONDS = 10
# Both full review hooks have a 7200-second client timeout. The budget leaves 600 seconds for Git,
# openspec status, prompts and result handling, and still fits the checks' total limit plus two
# waves of reviewers, so the client never kills a review without a message.
FULL_REVIEW_BUDGET_SECONDS = 6600
BUDGET_EXHAUSTED_ERROR = "the review time budget ran out before this reviewer started"
CHECK_OUTPUT_LINES = 80
OPENSPEC_TIMEOUT_SECONDS = 60
REVIEWER_ACTIVE_ENV = "AGENT_REVIEW_HOOK_ACTIVE"
HOOKS_DIR = Path(__file__).resolve().parent
GATES = ("plan", "necessity", "diff", "architecture", "naming", "implementation", "proposal")
GATES_YIELDING_TO_PROPOSAL_REVIEW = ("necessity", "diff", "architecture", "naming")
CODE_DIRS = ("backend/", "agent/", "frontend/src/", "website/")
CODE_SUFFIXES = (".go", ".ts", ".tsx")
TASK_PATTERN = re.compile(r"^\s*[-*] \[([ xX])\]", re.MULTILINE)


class MissingToolError(Exception):
    def __init__(self, tool):
        super().__init__(tool)
        self.tool = tool


@dataclass(frozen=True)
class Reviewer:
    claude_model: str
    claude_effort: str
    schema_path: Path
    timeout_seconds: int


# The quick reviewers run on the same model as the agent that writes the code, with low effort, so
# they answer fast. The full proposal and implementation reviews run once per change and look for
# what the quick checks miss, so their effort is higher.
QUICK_REVIEWER = Reviewer(
    claude_model="opus",
    claude_effort="low",
    schema_path=HOOKS_DIR / "review-result.schema.json",
    timeout_seconds=540,
)
IMPLEMENTATION_REVIEWER = Reviewer(
    claude_model="opus",
    claude_effort="medium",
    schema_path=HOOKS_DIR / "implementation-result.schema.json",
    timeout_seconds=1800,
)
PROPOSAL_REVIEWER = replace(IMPLEMENTATION_REVIEWER, schema_path=HOOKS_DIR / "proposal-result.schema.json")
OPENSPEC_PLANNING_COMMANDS = (
    "/opsx:propose",
    "/opsx:update",
)
# The repeat review reads the reply after a block, so the reply names the rejected findings,
# and the summary for the user waits until the review accepts the result.
FULL_REVIEW_REPLY_INSTRUCTION = (
    "Then finish the turn again with one short line about your fixes that names every rejected finding "
    "with the reason. Do not write the summary for the user yet: the hook asks for it once the review accepts the result. "
    "The repeat review checks the previous findings and the new edits."
)
HUMANIZER_CONTEXT = (
    "Mandatory first step: load the communication skill (.agents/skills/humanizer/SKILL.md) "
    "before any other action and apply it to all prose you write. "
    "Project rules: AGENTS.md."
)


@dataclass(frozen=True)
class Check:
    directory: str
    arguments: tuple


# The hook only runs checks that leave files untouched: a formatter would rewrite files a parallel
# agent is editing. Tests run only through the change verification commands a change names, and
# their forms leave out `just test`, which resets the test containers other agents share.
MODULE_CHECKS = (
    ("backend/", (Check("backend", ("golangci-lint", "run", "./cmd/...", "./internal/...")),)),
    (
        "agent/",
        (Check("agent/verification", ("golangci-lint", "run", "./cmd/...", "./internal/...", "./e2e/...")),),
    ),
    ("frontend/", (Check(".", ("pnpm", "--dir", "frontend", "lint")),)),
    (
        "website/",
        (
            Check(".", ("npm", "--prefix", "website", "run", "lint")),
            Check(".", ("npm", "--prefix", "website", "run", "build")),
        ),
    ),
    # Under test PATH holds stubs, so the hook tests run on this interpreter rather than python3 from PATH.
    (".agent-hooks/", (Check(".", (sys.executable, "-m", "unittest", "discover", "-s", ".agent-hooks", "-p", "*_test.py")),)),
)


class ChangeVerificationError(Exception):
    pass


@dataclass(frozen=True)
class VerificationForm:
    prefix: tuple
    directory: str
    usage: str
    min_arguments: int
    max_arguments: int | None
    is_package_list: bool


CHANGE_VERIFICATION_HEADING = "## Change verification"
VERIFICATION_FORMS = (
    VerificationForm(
        prefix=("just", "test-package"),
        directory="backend",
        usage="just test-package <package>...",
        min_arguments=1,
        max_arguments=None,
        is_package_list=True,
    ),
    VerificationForm(
        prefix=("just", "--justfile", "agent/verification/Justfile", "test"),
        directory=".",
        usage="just --justfile agent/verification/Justfile test",
        min_arguments=0,
        max_arguments=0,
        is_package_list=False,
    ),
    VerificationForm(
        prefix=("pnpm", "--dir", "frontend", "test"),
        directory=".",
        usage="pnpm --dir frontend test [<argument>...]",
        min_arguments=0,
        max_arguments=None,
        is_package_list=False,
    ),
)
# The commands run outside the agent's permission prompts, and both runners pass the arguments
# through a shell: `just test-package` interpolates them into a bash recipe. The set leaves out
# every shell operator, redirection and substitution character, so `./internal/a;rm` is refused
# although it holds no separate operator word.
VERIFICATION_ARGUMENT_PATTERN = re.compile(r"[A-Za-z0-9._/=:@-]+")
ALLOWED_VERIFICATION_FORMS = (
    "Allowed forms, one command per line in one ```sh block:\n"
    "- `just test-package <package>...`, every package starting with `./`, runs in `backend/`\n"
    "- `just --justfile agent/verification/Justfile test`, with nothing after it\n"
    "- `pnpm --dir frontend test [<argument>...]`\n"
    "Arguments may hold only letters, digits and `.`, `/`, `_`, `-`, `=`, `:`, `@`."
)


def read_change_verification_block(tasks_text):
    lines = tasks_text.splitlines()
    try:
        start = next(index for index, line in enumerate(lines) if line.strip() == CHANGE_VERIFICATION_HEADING)
    except StopIteration:
        return None

    # A shell comment inside a code block looks like a heading, so only a heading outside blocks ends the section.
    is_inside_other_block = False
    for index in range(start + 1, len(lines)):
        line = lines[index].strip()
        if line == "```sh" and not is_inside_other_block:
            return read_code_block(lines[index + 1 :])
        if line.startswith("```"):
            is_inside_other_block = not is_inside_other_block
        elif not is_inside_other_block and re.match(r"#{1,2} ", line):
            break
    raise ChangeVerificationError("the section has no ```sh code block")


def read_code_block(lines):
    for index, line in enumerate(lines):
        if line.strip() == "```":
            return lines[:index]
    raise ChangeVerificationError("the ```sh code block is not closed")


def parse_verification_command(line):
    try:
        words = shlex.split(line)
    except ValueError as error:
        raise ChangeVerificationError(f"`{line}` cannot be split into words: {error}")

    for form in VERIFICATION_FORMS:
        if tuple(words[: len(form.prefix)]) != form.prefix:
            continue

        arguments = words[len(form.prefix) :]
        for argument in arguments:
            if not VERIFICATION_ARGUMENT_PATTERN.fullmatch(argument):
                raise ChangeVerificationError(f"the argument `{argument}` in `{line}` has a forbidden character")
        if len(arguments) < form.min_arguments or (form.max_arguments is not None and len(arguments) > form.max_arguments):
            raise ChangeVerificationError(f"`{line}` does not match the form `{form.usage}`")
        if form.is_package_list and not all(argument.startswith("./") for argument in arguments):
            raise ChangeVerificationError(f"every package in `{line}` must start with `./`")
        return Check(form.directory, tuple(words))
    raise ChangeVerificationError(f"`{line}` is not an allowed command")


def parse_change_verification(change_dir):
    try:
        tasks_text = (change_dir / "tasks.md").read_text()
    except OSError:
        return []

    try:
        block = read_change_verification_block(tasks_text)
        if block is None:
            return []
        commands = [line.strip() for line in block if line.strip() and not line.strip().startswith("#")]
        if not commands:
            raise ChangeVerificationError("the ```sh code block holds no command")
        return [parse_verification_command(line) for line in commands]
    except ChangeVerificationError as error:
        raise ChangeVerificationError(
            f"The \"Change verification\" section of `{change_dir.name}/tasks.md` is not valid: {error}.\n\n"
            f"{ALLOWED_VERIFICATION_FORMS}"
        ) from None


def get_change_verification_checks(targets):
    checks = []
    for target in targets:
        for check in parse_change_verification(target.change_dir):
            if check not in checks:
                checks.append(check)
    return checks


def read_payload():
    try:
        value = json.load(sys.stdin)
    except (json.JSONDecodeError, OSError):
        return {}
    return value if isinstance(value, dict) else {}


def get_field(payload, *names):
    for name in names:
        value = payload.get(name)
        if value not in (None, ""):
            return value
    return None


def emit(value):
    json.dump(value, sys.stdout, ensure_ascii=False)
    sys.stdout.write("\n")


def run_git(root, *args):
    return subprocess.run(
        ["git", *args],
        cwd=root,
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    ).stdout


def find_root(payload):
    cwd = Path(get_field(payload, "workspaceRoot", "cwd") or Path.cwd()).resolve()
    try:
        return Path(run_git(cwd, "rev-parse", "--show-toplevel").decode().strip())
    except (OSError, subprocess.CalledProcessError):
        return HOOKS_DIR.parent


def get_untracked_paths(root):
    try:
        output = run_git(root, "ls-files", "--others", "--exclude-standard", "-z")
    except (OSError, subprocess.CalledProcessError):
        return []
    return [path for path in output.decode(errors="replace").split("\0") if path]


def get_changed_paths(root):
    paths = set(get_untracked_paths(root))
    try:
        output = run_git(root, "diff", "--name-only", "-z", "HEAD", "--")
    except (OSError, subprocess.CalledProcessError):
        output = run_git(root, "diff", "--name-only", "-z", "--cached", "--")
    paths.update(path for path in output.decode(errors="replace").split("\0") if path)
    return sorted(paths)


def get_workspace_fingerprint(root):
    digest = hashlib.sha256()
    try:
        digest.update(run_git(root, "rev-parse", "HEAD"))
        digest.update(run_git(root, "diff", "--binary", "HEAD", "--"))
    except (OSError, subprocess.CalledProcessError):
        digest.update(run_git(root, "diff", "--binary", "--cached", "--"))
        digest.update(run_git(root, "diff", "--binary", "--"))

    for relative_path in get_untracked_paths(root):
        path = root / relative_path
        digest.update(relative_path.encode())
        if path.is_symlink():
            digest.update(os.readlink(path).encode())
            continue
        if not path.is_file():
            continue
        try:
            with path.open("rb") as source:
                for chunk in iter(lambda: source.read(1024 * 1024), b""):
                    digest.update(chunk)
        except OSError:
            digest.update(b"unreadable")
    return digest.hexdigest()


def get_state_dir(root):
    try:
        git_dir = Path(run_git(root, "rev-parse", "--absolute-git-dir").decode().strip())
    except (OSError, subprocess.CalledProcessError):
        git_dir = root / ".git"
    return git_dir / "agent-review-hooks"


def get_session_key(payload):
    session_id = get_field(payload, "session_id", "sessionId") or "unknown"
    return re.sub(r"[^A-Za-z0-9_.-]", "_", str(session_id))


def load_state(path):
    try:
        value = json.loads(path.read_text())
    except (OSError, json.JSONDecodeError):
        return {}
    return value if isinstance(value, dict) else {}


def save_state(path, state):
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_suffix(".tmp")
    temporary.write_text(json.dumps(state, ensure_ascii=False))
    temporary.replace(path)


class Session:
    def __init__(self, root, payload):
        self.root = root
        self.payload = payload
        self.state_dir = get_state_dir(root)
        self.key = get_session_key(payload)

    @property
    def request_path(self):
        return self.state_dir / f"{self.key}.json"

    def gate_path(self, gate):
        return self.state_dir / f"{self.key}.{gate}.json"

    def load_request(self):
        return load_state(self.request_path)


def start_session():
    emit(
        {
            "hookSpecificOutput": {
                "hookEventName": "SessionStart",
                "additionalContext": HUMANIZER_CONTEXT,
            }
        }
    )


def snapshot(session):
    prompt = str(get_field(session.payload, "prompt", "userPrompt") or "")
    if prompt.startswith(AUTOMATIC_REVIEW_PREFIX) and session.load_request():
        return

    baseline_paths = get_changed_paths(session.root)
    save_state(
        session.request_path,
        {
            "baseline": get_workspace_fingerprint(session.root),
            "baseline_paths": baseline_paths,
            "baseline_files": get_file_hashes(session.root, baseline_paths),
            "openspec_files": get_openspec_file_hashes(session.root),
            "prompt": prompt,
        },
    )
    for gate in GATES:
        try:
            session.gate_path(gate).unlink()
        except FileNotFoundError:
            pass


def has_command(request, commands):
    prompt = str(request.get("prompt") or "").lower()
    return any(command in prompt for command in commands)


def get_active_changes(root):
    changes_dir = root / "openspec" / "changes"
    if not changes_dir.is_dir():
        return []
    return sorted(path for path in changes_dir.iterdir() if path.is_dir() and path.name != "archive")


def get_change_state_path(root, gate, change_name):
    return get_state_dir(root) / gate / f"{change_name}.json"


def get_changes_with_findings(root):
    return {
        change_dir.name
        for change_dir in get_active_changes(root)
        if load_state(get_change_state_path(root, "implementation", change_dir.name)).get("findings")
    }


def get_openspec_proposal_hashes(root, excluded_changes=frozenset()):
    hashes = {}
    for change_dir in get_active_changes(root):
        if change_dir.name in excluded_changes:
            continue

        # tasks.md is left out: implementing the change also checks its tasks.
        paths = [change_dir / "proposal.md", change_dir / "design.md", *change_dir.glob("specs/**/*.md")]
        for path in paths:
            try:
                hashes[str(path.relative_to(root))] = hashlib.sha256(path.read_bytes()).hexdigest()
            except OSError:
                continue
    return hashes


# The agent may write an OpenSpec proposal without a command in the request, for example
# after "write this up as a spec", so the turn is also recognized by changed artifacts.
# Edits to a change that has unaccepted full review findings follow those findings
# and do not count as a new proposal.
def is_openspec_planning(request, root):
    if has_command(request, OPENSPEC_PLANNING_COMMANDS):
        return True

    baseline = request.get("openspec_files")
    if not isinstance(baseline, dict):
        return False

    current = get_openspec_proposal_hashes(root, get_changes_with_findings(root))
    return any(baseline.get(path) != digest for path, digest in current.items())


def read_plan(tool_input):
    plan = str(tool_input.get("plan") or "").strip()
    if plan:
        return plan
    try:
        return Path(str(tool_input.get("planFilePath") or "")).read_text().strip()
    except OSError:
        return ""


def build_review_prompt(session, gate, subject, candidate, paths):
    request = session.load_request()
    instructions = "\n\n".join(
        [
            (HOOKS_DIR / "common-prompt.md").read_text(),
            (HOOKS_DIR / f"{gate}-prompt.md").read_text(),
        ]
    )
    baseline_paths = "\n".join(f"- {path}" for path in request.get("baseline_paths") or []) or "- none"
    changed_paths = "\n".join(f"- {path}" for path in paths) or "- none"
    # This builds a reviewer prompt, not an HTTP response.
    # nosemgrep: python.flask.security.audit.directly-returned-format-string.directly-returned-format-string
    return f"""{instructions}

Under review: {subject}
Repository root: {session.root}

Latest user request:
<user_request>
{request.get("prompt") or "(not captured)"}
</user_request>

Candidate:
<candidate>
{candidate}
</candidate>

Paths changed before this request:
{baseline_paths}

Currently changed paths:
{changed_paths}
"""


def load_reviewer_output(text):
    value = json.loads(text)
    if isinstance(value, dict) and isinstance(value.get("structured_output"), dict):
        value = value["structured_output"]
    if not isinstance(value, dict):
        raise ValueError("the reviewer returned a non-object result")
    return value


def parse_review_result(value):
    if not isinstance(value.get("ok"), bool) or not isinstance(value.get("reason"), str):
        raise ValueError("the reviewer result does not match the schema")
    return value


def get_reviewer_environment():
    environment = os.environ.copy()
    environment[REVIEWER_ACTIVE_ENV] = "1"
    for name in ("CLAUDECODE", "CLAUDE_CODE_SESSION_ID"):
        environment.pop(name, None)
    return environment


def run_claude_reviewer(root, prompt, model, effort, schema_path, timeout_seconds):
    command = [
        "claude",
        "--print",
        "--model",
        model,
        "--effort",
        effort,
        "--permission-mode",
        "plan",
        # The reviewer loads this repository's hooks too, and its own turn must never be gated.
        "--settings",
        '{"disableAllHooks": true}',
        "--tools",
        "Read,Glob,Grep,Bash",
        "--allowed-tools",
        "Read,Glob,Grep,Bash(git status *),Bash(git diff *),Bash(git log *),Bash(git show *),"
        "Bash(git ls-files *),Bash(git rev-parse *)",
        "--no-session-persistence",
        "--prompt-suggestions",
        "false",
        "--output-format",
        "json",
        "--json-schema",
        schema_path.read_text(),
    ]
    environment = get_reviewer_environment()
    environment["CLAUDE_CODE_EFFORT_LEVEL"] = effort

    process = subprocess.run(
        command,
        cwd=root,
        input=prompt,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        env=environment,
        timeout=timeout_seconds,
    )
    if process.returncode != 0:
        raise RuntimeError(process.stderr.strip() or process.stdout.strip() or f"claude exited with code {process.returncode}")
    return load_reviewer_output(process.stdout)


def run_reviewer(root, prompt, reviewer):
    return run_claude_reviewer(
        root, prompt, reviewer.claude_model, reviewer.claude_effort, reviewer.schema_path, reviewer.timeout_seconds
    )


def describe_reviewer_error(error):
    message = str(error).strip()[-1200:] or "unknown reviewer error"
    return f"The automatic reviewer could not run: {message}"


def review(root, prompt):
    try:
        result = parse_review_result(run_reviewer(root, prompt, QUICK_REVIEWER))
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired) as error:
        return False, describe_reviewer_error(error)

    if result["ok"]:
        return True, ""
    return False, result["reason"].strip() or "The reviewer rejected the result without a reason."


def gate_plan_tool(session, gate):
    gate_path = session.gate_path(gate)
    state = load_state(gate_path)
    plan = read_plan(get_field(session.payload, "tool_input", "toolInput") or {})
    if plan:
        prompt = build_review_prompt(session, gate, "plan", plan, get_changed_paths(session.root))
        is_accepted, reason = review(session.root, prompt)
    else:
        is_accepted, reason = False, "ExitPlanMode carried no plan text, so the plan could not be reviewed."

    if is_accepted:
        save_state(gate_path, {"blocks": 0})
        emit({})
        return

    blocks = int(state.get("blocks") or 0) + 1
    save_state(gate_path, {"blocks": blocks})
    if blocks > MAX_REVISIONS:
        emit(
            {
                "continue": False,
                "stopReason": f"Automatic plan review ({gate}) still fails after {MAX_REVISIONS} revisions: {reason}",
            }
        )
        return

    denial = f"{AUTOMATIC_REVIEW_PREFIX} {gate}\n{reason}\nRevise the plan and show it again."
    emit(
        {
            "hookSpecificOutput": {
                "hookEventName": "PreToolUse",
                "permissionDecision": "deny",
                "permissionDecisionReason": denial,
            }
        }
    )


# A plan reaches the reviewer through ExitPlanMode, which has its own check, so a finished turn
# never carries one.
def get_stop_subject(gate, is_openspec_turn, is_workspace_changed, blocks, is_code_changed=False):
    if gate == "naming":
        return "finished change" if is_code_changed or blocks > 0 else None

    if gate == "necessity":
        return "OpenSpec proposal" if is_openspec_turn and is_workspace_changed else None
    if gate == "plan":
        return None
    if is_workspace_changed or blocks > 0:
        return "finished change"
    return None


def gate_stop(session, gate):
    request = session.load_request()
    if gate in GATES_YIELDING_TO_PROPOSAL_REVIEW and is_proposal_review_turn(session.root, request):
        emit({})
        return

    gate_path = session.gate_path(gate)
    state = load_state(gate_path)
    fingerprint = get_workspace_fingerprint(session.root)
    blocks = int(state.get("blocks") or 0)
    subject = get_stop_subject(
        gate,
        is_openspec_planning(request, session.root),
        request.get("baseline") != fingerprint,
        blocks,
        gate == "naming" and is_code_changed_since_request(session.root, request),
    )

    if subject is None or (state.get("accepted") == fingerprint and blocks == 0):
        emit({})
        return

    candidate = str(get_field(session.payload, "last_assistant_message", "lastAssistantMessage") or "(not available)")
    prompt = build_review_prompt(session, gate, subject, candidate, get_changed_paths(session.root))
    is_accepted, reason = review(session.root, prompt)

    if is_accepted:
        save_state(gate_path, {"blocks": 0, "accepted": fingerprint})
        emit({})
        return

    blocks += 1
    save_state(gate_path, {"blocks": blocks})
    if blocks > MAX_REVISIONS:
        emit(
            {
                "continue": False,
                "stopReason": f"Automatic review ({gate}) still fails after {MAX_REVISIONS} revisions: {reason}",
                "systemMessage": f"Automatic review ({gate}) reached its revision limit. The result was not accepted.",
            }
        )
        return

    if subject == "plan":
        instruction = "Revise the plan and show it again; the review will repeat."
    else:
        instruction = (
            f"Fix the {subject}, then finish the turn again with a new reply written for the user: "
            "a summary of the final state, without retelling this review, its findings or your fixes. "
            "The review will repeat."
        )
    emit({"decision": "block", "reason": f"{AUTOMATIC_REVIEW_PREFIX} {gate}\n{reason}\n{instruction}"})


def get_task_marks(change_dir):
    try:
        return TASK_PATTERN.findall((change_dir / "tasks.md").read_text())
    except OSError:
        return []


def is_implemented(change_dir):
    marks = get_task_marks(change_dir)
    return bool(marks) and " " not in marks


def is_unstarted(change_dir):
    marks = get_task_marks(change_dir)
    return bool(marks) and all(mark == " " for mark in marks)


# The CLI decides whether the artifacts are complete: it knows the change's schema, an optional
# design.md and skip_specs. A CLI that cannot start raises MissingToolError, because skipping
# the proposal review would let an unreviewed proposal pass silently.
def is_planning_complete(root, change_name):
    try:
        process = subprocess.run(
            ["openspec", "status", "--change", change_name, "--json"],
            cwd=root,
            text=True,
            errors="replace",
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            timeout=OPENSPEC_TIMEOUT_SECONDS,
        )
    except OSError as error:
        raise MissingToolError("openspec") from error
    except subprocess.TimeoutExpired:
        return False
    try:
        status = json.loads(process.stdout)
    except json.JSONDecodeError:
        return False
    return process.returncode == 0 and isinstance(status, dict) and status.get("isPlanningComplete") is True


def get_file_hashes(root, relative_paths):
    hashes = {}
    for relative_path in relative_paths:
        try:
            hashes[relative_path] = hashlib.sha256((root / relative_path).read_bytes()).hexdigest()
        except OSError:
            hashes[relative_path] = "missing"
    return hashes


def get_paths_changed_since(reviewed_files, current_files):
    paths = set(reviewed_files) | set(current_files)
    return sorted(path for path in paths if reviewed_files.get(path) != current_files.get(path))


def get_change_file_hashes(root, change_dir):
    paths = sorted(path for path in change_dir.rglob("*") if path.is_file())
    return get_file_hashes(root, [str(path.relative_to(root)) for path in paths])


def get_openspec_file_hashes(root):
    hashes = {}
    for change_dir in get_active_changes(root):
        hashes.update(get_change_file_hashes(root, change_dir))
    return hashes


def is_code_path(path):
    return path.startswith(CODE_DIRS) and path.endswith(CODE_SUFFIXES)


# Changed code is found by file hashes, as in is_proposal_review_turn: someone else's changed
# file that was already in the tree before the request does not start the naming reviewer.
def is_code_changed_since_request(root, request):
    current_files = get_file_hashes(root, get_changed_paths(root))
    changed_paths = get_paths_changed_since(request.get("baseline_files") or {}, current_files)
    return any(is_code_path(path) for path in changed_paths)


# The conditions go from cheap to expensive: openspec status runs last.
def is_proposal_review_turn(root, request):
    current_files = get_file_hashes(root, get_changed_paths(root))
    change_names = set()
    for path in get_paths_changed_since(request.get("baseline_files") or {}, current_files):
        parts = Path(path).parts
        if len(parts) < 4 or parts[:2] != ("openspec", "changes") or parts[2] == "archive":
            return False
        change_names.add(parts[2])

    changes_dir = root / "openspec" / "changes"
    return bool(change_names) and all(
        is_unstarted(changes_dir / name) and is_planning_complete(root, name) for name in sorted(change_names)
    )


def get_output_tail(output):
    if isinstance(output, bytes):
        output = output.decode(errors="replace")
    return "\n".join((output or "").rstrip().splitlines()[-CHECK_OUTPUT_LINES:]) or "(no output)"


# `just` starts bash, bash starts `go test`, and `go test` starts test binaries that hold a test slot
# database and containers. Each check runs in its own process group, so a timeout kills them all.
def stop_process_group(process):
    try:
        os.killpg(process.pid, signal.SIGKILL)
    except ProcessLookupError:
        pass
    try:
        return process.communicate(timeout=KILLED_CHECK_OUTPUT_TIMEOUT_SECONDS)[0]
    except subprocess.TimeoutExpired as error:
        # A descendant that left the group still holds the pipe, so the output read so far is all there is.
        # TimeoutExpired carries it as bytes even for a text-mode process; get_output_tail decodes it.
        process.stdout.close()
        process.wait()
        return error.output


def run_checks(root, checks, check_timeout_seconds=CHECK_TIMEOUT_SECONDS, total_timeout_seconds=CHECKS_TOTAL_TIMEOUT_SECONDS):
    deadline = time.monotonic() + total_timeout_seconds
    for check in checks:
        command = f"`{' '.join(check.arguments)}` in `{check.directory}`"
        remaining_seconds = max(deadline - time.monotonic(), 0)
        if check_timeout_seconds <= remaining_seconds:
            timeout_seconds, exceeded_limit = check_timeout_seconds, f"its limit of {check_timeout_seconds} seconds per check"
        else:
            timeout_seconds, exceeded_limit = remaining_seconds, f"the total limit of {total_timeout_seconds} seconds for all checks"

        try:
            # The hooks run on Python 3.6 or newer.
            # nosemgrep: python.lang.compatibility.python36.python36-compatibility-Popen1
            process = subprocess.Popen(
                list(check.arguments),
                cwd=root / check.directory,
                text=True,
                errors="replace",
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                start_new_session=True,
            )
        except OSError as error:
            return f"{command} could not start: {error}"

        try:
            output = process.communicate(timeout=timeout_seconds)[0]
        except subprocess.TimeoutExpired:
            output = stop_process_group(process)
            return f"{command} did not finish within {exceeded_limit} and was stopped with every process it started.\n\nEnd of output:\n{get_output_tail(output)}"

        if process.returncode != 0:
            return f"{command} exited with code {process.returncode}.\n\nEnd of output:\n{get_output_tail(output)}"
    return None


def get_module_checks(root):
    changed_paths = get_changed_paths(root)
    return [
        check
        for prefix, checks in MODULE_CHECKS
        if any(path.startswith(prefix) for path in changed_paths)
        for check in checks
    ]


@dataclass(frozen=True)
class ReviewTarget:
    change_dir: Path
    state: dict
    accepted: str
    files: dict


@dataclass(frozen=True)
class FullReview:
    gate: str
    title: str
    subject: str
    reviewer: Reviewer
    categories: tuple
    non_blocking_categories: tuple
    summary_categories: tuple
    category_notes: dict
    first_roles: tuple
    files_title: str
    select_targets: Callable
    get_checks: Callable
    block_instruction: str
    optional_findings_title: str
    summary_instruction: str


@dataclass(frozen=True)
class ReviewJob:
    change_name: str
    prompt: str


def format_paths(paths):
    return "\n".join(f"- {path}" for path in paths) or "- none"


def format_findings(full_review, findings):
    order = {category: index for index, category in enumerate(full_review.categories)}
    lines = []
    for finding in sorted(findings, key=lambda finding: order[finding["category"]]):
        note = full_review.category_notes.get(finding["category"], "")
        lines.append(f"- [{finding['category']}]{note} {finding['file']}: {finding['summary']}")
    return "\n".join(lines)


def build_full_review_prompt(session, full_review, change_dir, role, details):
    request = session.load_request()
    instructions = "\n\n".join(
        [
            (HOOKS_DIR / "common-prompt.md").read_text(),
            (HOOKS_DIR / f"{full_review.gate}-prompt.md").read_text(),
        ]
    )
    candidate = str(get_field(session.payload, "last_assistant_message", "lastAssistantMessage") or "(not available)")
    # This builds a reviewer prompt, not an HTTP response.
    # nosemgrep: python.flask.security.audit.directly-returned-format-string.directly-returned-format-string
    return f"""{instructions}

OpenSpec change: {change_dir.relative_to(session.root)}
Role: {role}
Repository root: {session.root}

Latest user request:
<user_request>
{request.get("prompt") or "(not captured)"}
</user_request>

Agent response:
<candidate>
{candidate}
</candidate>

{details}
"""


def build_review_jobs(session, full_review, target):
    change_dir = target.change_dir
    findings = target.state.get("findings")
    if findings:
        changed_since_review = get_paths_changed_since(target.state.get("reviewed_files") or {}, target.files)
        details = (
            f"Previous review findings:\n{format_findings(full_review, findings)}\n\n"
            f"Paths changed after the previous review:\n{format_paths(changed_since_review)}"
        )
        prompt = build_full_review_prompt(session, full_review, change_dir, "repeat review", details)
        return [ReviewJob(change_dir.name, prompt)]

    details = f"{full_review.files_title}:\n{format_paths(sorted(target.files))}"
    spec_paths = sorted(change_dir.glob("specs/**/spec.md"))
    roles = [f"delta spec `{path.relative_to(session.root)}`" for path in spec_paths]
    roles.extend(full_review.first_roles)
    return [ReviewJob(change_dir.name, build_full_review_prompt(session, full_review, change_dir, role, details)) for role in roles]


def parse_findings(full_review, value):
    findings = value.get("findings")
    if not isinstance(findings, list):
        raise ValueError("the reviewer result does not match the schema")

    for finding in findings:
        if (
            not isinstance(finding, dict)
            or finding.get("category") not in full_review.categories
            or not isinstance(finding.get("file"), str)
            or not isinstance(finding.get("summary"), str)
        ):
            raise ValueError("the reviewer result does not match the schema")
    return [{"category": finding["category"], "file": finding["file"], "summary": finding["summary"]} for finding in findings]


def run_review_job(root, full_review, job, deadline):
    if deadline - time.monotonic() < full_review.reviewer.timeout_seconds:
        return [], describe_reviewer_error(BUDGET_EXHAUSTED_ERROR)
    try:
        return parse_findings(full_review, run_reviewer(root, job.prompt, full_review.reviewer)), None
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired) as error:
        return [], describe_reviewer_error(error)


def block_full_review(session, full_review, report, instruction):
    gate_path = session.gate_path(full_review.gate)
    blocks = int(load_state(gate_path).get("blocks") or 0) + 1
    save_state(gate_path, {"blocks": blocks})
    if blocks > MAX_FULL_REVIEW_REVISIONS:
        emit(
            {
                "continue": False,
                "stopReason": f"{full_review.title} still fails after {MAX_FULL_REVIEW_REVISIONS} repeats.\n\n{report}",
                "systemMessage": f"{full_review.title} reached its repeat limit. The result was not accepted.",
            }
        )
        return

    emit({"decision": "block", "reason": f"{AUTOMATIC_REVIEW_PREFIX} {full_review.gate}\n{report}\n\n{instruction}"})


def gate_full_review(session, full_review, deadline):
    root = session.root
    targets = full_review.select_targets(session)
    if not targets:
        emit({})
        return

    try:
        checks = full_review.get_checks(root, targets)
    except ChangeVerificationError as error:
        block_full_review(
            session,
            full_review,
            f"{error}\n\nNo check or reviewer ran.",
            "Fix the section. " + FULL_REVIEW_REPLY_INSTRUCTION,
        )
        return

    checks_timeout_seconds = min(CHECKS_TOTAL_TIMEOUT_SECONDS, max(deadline - time.monotonic(), 0))
    check_failure = run_checks(root, checks, total_timeout_seconds=checks_timeout_seconds)
    if check_failure:
        block_full_review(
            session,
            full_review,
            f"{check_failure}\n\nNo reviewer ran.",
            "Fix the failure. " + FULL_REVIEW_REPLY_INSTRUCTION,
        )
        return

    jobs = [job for target in targets for job in build_review_jobs(session, full_review, target)]
    with ThreadPoolExecutor(max_workers=MAX_PARALLEL_REVIEWERS) as executor:
        results = list(executor.map(lambda job: run_review_job(root, full_review, job, deadline), jobs))

    reports = []
    has_blocking = False
    has_summary_findings = False
    for target in targets:
        change_name = target.change_dir.name
        change_results = [result for job, result in zip(jobs, results) if job.change_name == change_name]
        findings = [finding for change_findings, _ in change_results for finding in change_findings]
        errors = [error for _, error in change_results if error]
        is_blocking = bool(errors) or any(
            finding["category"] not in full_review.non_blocking_categories for finding in findings
        )

        state_path = get_change_state_path(root, full_review.gate, change_name)
        if is_blocking and not errors:
            save_state(state_path, {"findings": findings, "reviewed_files": target.files})
        elif not is_blocking:
            save_state(state_path, {"accepted": target.accepted, "findings": [], "reviewed_files": {}})

        has_blocking = has_blocking or is_blocking
        has_summary_findings = has_summary_findings or any(
            finding["category"] in full_review.summary_categories for finding in findings
        )
        lines = [format_findings(full_review, findings), *errors]
        if any(lines):
            reports.append(f"Change `{change_name}`:\n" + "\n".join(line for line in lines if line))

    report = "\n\n".join(reports)
    if has_blocking:
        block_full_review(session, full_review, f"{full_review.title} did not accept the {full_review.subject}.\n\n{report}", full_review.block_instruction)
        return

    gate_path = session.gate_path(full_review.gate)
    previous_blocks = int(load_state(gate_path).get("blocks") or 0)
    save_state(gate_path, {"blocks": 0})
    # After fix rounds the agent's last response covers only the fixes,
    # so it writes the summary of the whole work in a separate turn.
    if previous_blocks or has_summary_findings:
        optional_findings = f"\n\n{full_review.optional_findings_title}:\n\n{report}" if report else ""
        emit(
            {
                "decision": "block",
                "reason": f"{AUTOMATIC_REVIEW_PREFIX} {full_review.gate}\n"
                f"{full_review.title} accepted the {full_review.subject}.{optional_findings}\n\n{full_review.summary_instruction}",
            }
        )
        return

    if report:
        emit({"systemMessage": f"{full_review.title} accepted the {full_review.subject}. {full_review.optional_findings_title}:\n\n{report}"})
        return
    emit({})


def select_implementation_targets(session):
    root = session.root
    request = session.load_request()
    fingerprint = get_workspace_fingerprint(root)
    if "baseline" not in request or request["baseline"] == fingerprint or has_command(request, OPENSPEC_PLANNING_COMMANDS):
        return []

    change_dirs = [change_dir for change_dir in get_active_changes(root) if is_implemented(change_dir)]
    states = {change_dir: load_state(get_change_state_path(root, "implementation", change_dir.name)) for change_dir in change_dirs}
    change_dirs = [change_dir for change_dir in change_dirs if states[change_dir].get("accepted") != fingerprint]
    if not change_dirs:
        return []

    files = get_file_hashes(root, get_changed_paths(root))
    return [ReviewTarget(change_dir, states[change_dir], fingerprint, files) for change_dir in change_dirs]


def select_proposal_targets(session):
    root = session.root
    baseline = session.load_request().get("openspec_files")
    if not isinstance(baseline, dict):
        return []

    targets = []
    for change_dir in get_active_changes(root):
        files = get_change_file_hashes(root, change_dir)
        prefix = f"{change_dir.relative_to(root)}/"
        baseline_files = {path: digest for path, digest in baseline.items() if path.startswith(prefix)}
        if files == baseline_files or not is_unstarted(change_dir):
            continue

        accepted = hashlib.sha256(json.dumps(files, sort_keys=True).encode()).hexdigest()
        state = load_state(get_change_state_path(root, "proposal", change_dir.name))
        if state.get("accepted") == accepted or not is_planning_complete(root, change_dir.name):
            continue
        targets.append(ReviewTarget(change_dir, state, accepted, files))
    return targets


# Every section is parsed before the first check, so a malformed one blocks the turn before anything runs.
def get_implementation_checks(root, targets):
    verification_checks = get_change_verification_checks(targets)
    return [*get_module_checks(root), *verification_checks]


# The proposal review checks the section's format only: no code exists yet for its commands to test.
def get_proposal_checks(root, targets):
    for target in targets:
        parse_change_verification(target.change_dir)
    return [Check(".", ("openspec", "validate", target.change_dir.name, "--strict")) for target in targets]


IMPLEMENTATION_REVIEW = FullReview(
    gate="implementation",
    title="Full implementation review",
    subject="change",
    reviewer=IMPLEMENTATION_REVIEWER,
    categories=("bug", "spec_mismatch", "missing_test", "architecture", "docs", "minor"),
    non_blocking_categories=("minor",),
    summary_categories=(),
    category_notes={"minor": " (optional)"},
    first_roles=("architecture", "documentation and tasks", "tests"),
    files_title="Paths changed relative to HEAD",
    select_targets=select_implementation_targets,
    get_checks=get_implementation_checks,
    block_instruction="Resolve every finding yourself, without asking the user. Fix a bug in the code. "
    "On a spec mismatch, pick the right side and fix the other one, including the change's delta spec. "
    "On an architecture finding, move the code to follow docs/architecture.md, "
    "or change docs/architecture.md in the same change when the rule itself is wrong. "
    "Reject a false finding. Minor findings are optional. "
    "If a reviewer could not run, find the cause of the failure. " + FULL_REVIEW_REPLY_INSTRUCTION,
    optional_findings_title="Optional findings",
    summary_instruction="Do not change files any more. Write the user a summary of the implementation as if there had been no review: "
    "what was done for the change's tasks, which decisions were made, which checks passed, "
    "which minor findings you left, and the next step. "
    "Leave out review rounds, findings and the fixes made for them.",
)
PROPOSAL_REVIEW = FullReview(
    gate="proposal",
    title="Full proposal review",
    subject="proposal",
    reviewer=PROPOSAL_REVIEWER,
    categories=("gap", "mismatch", "tasks", "simplify", "scope", "minor"),
    non_blocking_categories=("scope", "minor"),
    summary_categories=("scope",),
    category_notes={"scope": " (the user decides)", "minor": " (optional)"},
    first_roles=("necessity", "architecture", "names", "proposal and design", "tasks"),
    files_title="Change files",
    select_targets=select_proposal_targets,
    get_checks=get_proposal_checks,
    block_instruction="Resolve every gap, mismatch, tasks and simplify finding yourself, without asking the user: "
    "fix the change's artifacts, and reject a false finding. "
    "Do not apply scope findings: cutting the feature needs the user's consent, and the user sees them in the summary. "
    "Minor findings are optional. If a reviewer could not run, find the cause of the failure. " + FULL_REVIEW_REPLY_INSTRUCTION,
    optional_findings_title="Findings that do not block the turn",
    summary_instruction="Do not change files any more. Write the user a summary of the proposal as if there had been no review: "
    "what is proposed and which decisions were made. "
    "Leave out review rounds, findings and the fixes made for them. "
    "List every scope finding separately: what the user loses and what they save, the decision is theirs. "
    "End with the next step.",
)
FULL_REVIEWS = {full_review.gate: full_review for full_review in (IMPLEMENTATION_REVIEW, PROPOSAL_REVIEW)}


def check_environment():
    tool_paths = {
        "python": sys.executable,
        "git": shutil.which("git"),
        "openspec": shutil.which("openspec"),
        "claude": shutil.which("claude"),
    }
    for tool, path in tool_paths.items():
        print(f"{tool}: {path or 'not found'}")

    return 0 if all(tool_paths.values()) else 1


def stop_for_missing_tool(error):
    message = (
        f"The review hooks could not find `{error.tool}`, so the review did not run. "
        "Run `mise install` to install the pinned tools and `just doctor` to check the environment."
    )
    emit({"continue": False, "stopReason": message, "systemMessage": message})


def main():
    deadline = time.monotonic() + FULL_REVIEW_BUDGET_SECONDS
    parser = argparse.ArgumentParser()
    parser.add_argument("action", choices=("session-start", "snapshot", "doctor", *GATES))
    arguments = parser.parse_args()
    if arguments.action == "doctor":
        sys.exit(check_environment())

    try:
        run_action(arguments.action, deadline)
    except MissingToolError as error:
        stop_for_missing_tool(error)


def run_action(action, deadline):
    payload = read_payload()
    if os.environ.get(REVIEWER_ACTIVE_ENV) == "1":
        emit({})
        return
    if action == "session-start":
        start_session()
        return

    session = Session(find_root(payload), payload)
    if action == "snapshot":
        snapshot(session)
        return

    if action in FULL_REVIEWS:
        gate_full_review(session, FULL_REVIEWS[action], deadline)
        return

    event = get_field(payload, "hook_event_name", "hookEventName")
    if event == "PreToolUse":
        gate_plan_tool(session, action)
        return
    gate_stop(session, action)


if __name__ == "__main__":
    main()
