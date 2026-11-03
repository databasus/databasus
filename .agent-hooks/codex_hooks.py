import json
import os
from pathlib import Path
import selectors
import subprocess
import tempfile
import time


def get_last_line(error_output):
    error_output.seek(0)
    lines = error_output.read().decode(errors="replace").splitlines()
    return lines[-1] if lines else "no error output"


def get_project_hooks(executable, root):
    requests = [
        {"id": 1, "method": "initialize", "params": {
            "clientInfo": {"name": "databasus-doctor", "version": "1"},
            "capabilities": {"experimentalApi": True},
        }},
        {"method": "initialized"},
        {"id": 2, "method": "hooks/list", "params": {"cwds": [str(root)]}},
        {"id": 3, "method": "config/read", "params": {"cwd": str(root), "includeLayers": False}},
    ]
    with tempfile.TemporaryFile() as error_output, subprocess.Popen(
        [executable, "app-server"], cwd=root, stdin=subprocess.PIPE,
        stdout=subprocess.PIPE, stderr=error_output,
    ) as process:
        try:
            process.stdin.write(("\n".join(json.dumps(request) for request in requests) + "\n").encode())
            process.stdin.flush()

            deadline = time.monotonic() + 10
            pending = b""
            answers = {}
            with selectors.DefaultSelector() as selector:
                selector.register(process.stdout, selectors.EVENT_READ)
                while time.monotonic() < deadline:
                    if not selector.select(max(0, deadline - time.monotonic())):
                        break
                    chunk = os.read(process.stdout.fileno(), 65536)
                    if not chunk:
                        raise ValueError(f"app-server closed before answering hooks/list: {get_last_line(error_output)}")
                    pending += chunk
                    if len(pending) > 1024 * 1024:
                        raise ValueError("app-server response exceeded the inspection limit")

                    while b"\n" in pending:
                        line, pending = pending.split(b"\n", 1)
                        response = json.loads(line)
                        if response.get("id") in (2, 3):
                            answers[response["id"]] = response["result"]
                        if 2 in answers and 3 in answers:
                            features = answers[3]["config"].get("features") or {}
                            return answers[2]["data"], features.get("hooks", True)

            raise ValueError("hooks/list did not answer within 10 seconds")
        finally:
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()


def check_project_hooks(executable, root):
    configuration_path = root / ".codex/hooks.json"
    if not configuration_path.is_file():
        return True
    try:
        configuration = json.loads(configuration_path.read_text())
        configured_hooks = [
            (event[0].lower() + event[1:], hook["command"])
            for event, groups in configuration["hooks"].items()
            for group in groups for hook in group["hooks"]
        ]

        entries, is_enabled = get_project_hooks(executable, root)
        if not is_enabled:
            print("Codex project hooks: features.hooks is disabled")
            return False
        if not configured_hooks or any(entry.get("errors") for entry in entries):
            raise ValueError("project hook configuration is empty or invalid")

        loaded_hooks = {
            (hook["eventName"], hook["command"]): hook
            for entry in entries for hook in entry["hooks"]
            if Path(hook["sourcePath"]) == configuration_path
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
            print("Open Codex in this project and use /hooks to inspect and approve its hooks; also check project trust and features.hooks.")
            return False
        print(f"Codex project hooks: {len(configured_hooks)} trusted and enabled")
        return True
    except (OSError, ValueError, KeyError, TypeError) as error:
        print(f"Codex project hooks: could not inspect hooks/list ({type(error).__name__}: {error})")
        print("Inside the Codex sandbox, `codex app-server` cannot write its home directory, so run this check outside the sandbox.")
        return False
