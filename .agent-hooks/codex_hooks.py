import json
import os
import selectors
import subprocess
import tempfile
import time
from pathlib import Path
from typing import BinaryIO, NotRequired, TypedDict, TypeGuard

from hook_configuration import get_hook_configuration


class NativeHook(TypedDict):
    sourcePath: str
    eventName: str
    command: str
    enabled: bool
    trustStatus: str


class NativeHookEntry(TypedDict):
    hooks: list[NativeHook]
    errors: NotRequired[object]


def is_native_hook(value: object) -> TypeGuard[NativeHook]:
    return (
        isinstance(value, dict)
        and all(
            isinstance(value.get(key), str)
            for key in ("sourcePath", "eventName", "command", "trustStatus")
        )
        and isinstance(value.get("enabled"), bool)
    )


def is_native_hook_entry(value: object) -> TypeGuard[NativeHookEntry]:
    if not isinstance(value, dict):
        return False
    hooks = value.get("hooks")
    return isinstance(hooks, list) and all(is_native_hook(hook) for hook in hooks)


def get_last_line(error_output: BinaryIO) -> str:
    error_output.seek(0)
    lines = error_output.read().decode(errors="replace").splitlines()
    return lines[-1] if lines else "no error output"


def get_project_hooks(executable: str, root: Path) -> tuple[list[NativeHookEntry], bool]:
    requests = [
        {
            "id": 1,
            "method": "initialize",
            "params": {
                "clientInfo": {"name": "databasus-doctor", "version": "1"},
                "capabilities": {"experimentalApi": True},
            },
        },
        {"method": "initialized"},
        {"id": 2, "method": "hooks/list", "params": {"cwds": [str(root)]}},
        {"id": 3, "method": "config/read", "params": {"cwd": str(root), "includeLayers": False}},
    ]
    with (
        tempfile.TemporaryFile() as error_output,
        subprocess.Popen(
            [executable, "app-server"],
            cwd=root,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=error_output,
        ) as process,
    ):
        try:
            assert process.stdin is not None
            assert process.stdout is not None
            process.stdin.write(
                ("\n".join(json.dumps(request) for request in requests) + "\n").encode()
            )
            process.stdin.flush()

            deadline = time.monotonic() + 10
            pending = b""
            answers: dict[int, dict[str, object]] = {}
            with selectors.DefaultSelector() as selector:
                selector.register(process.stdout, selectors.EVENT_READ)
                while time.monotonic() < deadline:
                    if not selector.select(max(0, deadline - time.monotonic())):
                        break
                    chunk = os.read(process.stdout.fileno(), 65536)
                    if not chunk:
                        raise ValueError(
                            f"app-server closed before answering hooks/list: {get_last_line(error_output)}"
                        )
                    pending += chunk
                    if len(pending) > 1024 * 1024:
                        raise ValueError("app-server response exceeded the inspection limit")

                    while b"\n" in pending:
                        line, pending = pending.split(b"\n", 1)
                        response: object = json.loads(line)
                        if not isinstance(response, dict):
                            raise ValueError("app-server response must be an object")
                        response_id = response.get("id")
                        if response_id in (2, 3):
                            response_result = response["result"]
                            if not isinstance(response_result, dict):
                                raise ValueError("app-server result must be an object")
                            answers[int(response_id)] = response_result
                        if 2 in answers and 3 in answers:
                            configuration = answers[3]["config"]
                            if not isinstance(configuration, dict):
                                raise ValueError("app-server configuration must be an object")
                            features = configuration.get("features") or {}
                            if not isinstance(features, dict):
                                raise ValueError("app-server features must be an object")
                            is_enabled = features.get("hooks", True)
                            entries = answers[2]["data"]
                            if not isinstance(is_enabled, bool) or not isinstance(entries, list):
                                raise ValueError("app-server returned invalid hook configuration")
                            parsed_entries = []
                            for entry in entries:
                                if not is_native_hook_entry(entry):
                                    raise ValueError("app-server returned an invalid hook entry")
                                parsed_entries.append(entry)
                            return parsed_entries, is_enabled

            raise ValueError("hooks/list did not answer within 10 seconds")
        finally:
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()


def get_project_hook_sources(root: Path) -> set[Path]:
    worktrees = subprocess.run(
        ["git", "worktree", "list", "--porcelain", "-z"],
        cwd=root,
        capture_output=True,
        check=True,
        timeout=10,
    )
    main_checkout_record = worktrees.stdout.split(b"\0", 1)[0]
    main_checkout = Path(os.fsdecode(main_checkout_record.removeprefix(b"worktree ")))

    return {(checkout / ".codex/hooks.json").resolve() for checkout in (root, main_checkout)}


def check_project_hooks(executable: str, root: Path) -> bool:
    configuration_path = root / ".codex/hooks.json"
    if not configuration_path.is_file():
        return True
    try:
        configuration = json.loads(configuration_path.read_text())
        configured_events = get_hook_configuration(configuration["hooks"])
        configured_hooks = [
            (event[0].lower() + event[1:], hook["command"])
            for event, groups in configured_events.items()
            for group in groups
            for hook in group["hooks"]
        ]

        entries, is_enabled = get_project_hooks(executable, root)
        if not is_enabled:
            print("Codex project hooks: features.hooks is disabled")
            return False
        if not configured_hooks or any(entry.get("errors") for entry in entries):
            raise ValueError("project hook configuration is empty or invalid")

        project_hook_sources = get_project_hook_sources(root)
        loaded_hooks = {
            (hook["eventName"], hook["command"]): hook
            for entry in entries
            for hook in entry["hooks"]
            if Path(hook["sourcePath"]).resolve() in project_hook_sources
        }

        failures = []
        for event, command in configured_hooks:
            hook = loaded_hooks.get((event, command))
            if hook is None:
                failures.append(f"{event}: not loaded")
            elif not hook["enabled"]:
                failures.append(f"{event}: disabled")
            elif hook["trustStatus"] != "trusted":
                failures.append(f"{event}: {hook['trustStatus']}")

        if failures:
            print("Codex project hooks: " + "; ".join(failures))
            print(
                "Open Codex in this project and use /hooks to inspect and approve its hooks; also check project trust and features.hooks."
            )
            return False
        print(f"Codex project hooks: {len(configured_hooks)} trusted and enabled")
        return True
    except (OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError) as error:
        print(
            f"Codex project hooks: could not inspect hooks/list ({type(error).__name__}: {error})"
        )
        print(
            "Inside the Codex sandbox, `codex app-server` cannot write its home directory, so run this check outside the sandbox."
        )
        return False
