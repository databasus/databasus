import json
import os
from pathlib import Path
import signal
import subprocess
import tempfile
import time


DISABLED_FEATURES = (
    "hooks", "apps", "enable_mcp_apps", "multi_agent", "multi_agent_v2",
    "multi_agent_v2_dynamic_tools",
)


def run_codex_command(request, arguments, deadline):
    remaining = deadline - time.monotonic()
    if remaining <= 0:
        raise subprocess.TimeoutExpired("codex", request.timeout_seconds)

    with subprocess.Popen(
        ["codex", *arguments], cwd=request.root, env=request.environment,
        stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
        text=True, start_new_session=True,
    ) as process:
        try:
            stdout, stderr = process.communicate(request.prompt if "exec" in arguments else "", timeout=remaining)
        except subprocess.TimeoutExpired:
            try:
                os.killpg(process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            process.wait()
            raise

    if process.returncode:
        # Metadata can contain server credentials; never include its output in diagnostics.
        detail = stderr.strip() if "exec" in arguments else "configuration metadata or restrictions unavailable"
        raise RuntimeError(f"codex exited with code {process.returncode}: {detail}")
    return stdout


def get_codex_restrictions(request, deadline):
    restrictions = ["-c", 'approval_policy="never"', "-c", 'sandbox_mode="read-only"']
    for feature in DISABLED_FEATURES:
        restrictions.extend(["--disable", feature])
    servers = json.loads(run_codex_command(request, [*restrictions, "mcp", "list", "--json"], deadline))
    plugins = json.loads(run_codex_command(request, [*restrictions, "plugin", "list", "--json"], deadline))

    if not isinstance(servers, list) or not isinstance(plugins, dict) or not isinstance(plugins.get("installed"), list):
        raise ValueError("codex returned invalid configuration metadata")
    server_names = []
    plugin_ids = []
    for server in servers:
        if not isinstance(server, dict) or not isinstance(server.get("name"), str) or not server["name"]:
            raise ValueError("codex returned an invalid MCP server identity")
        server_names.append(server["name"])
    for plugin in plugins["installed"]:
        if not isinstance(plugin, dict) or not isinstance(plugin.get("pluginId"), str) or not plugin["pluginId"]:
            raise ValueError("codex returned an invalid plugin identity")
        plugin_ids.append(plugin["pluginId"])

    for namespace, names in (("mcp_servers", server_names), ("plugins", plugin_ids)):
        entries = ",".join(f"{json.dumps(name)}={{enabled=false}}" for name in names)
        restrictions.extend(["-c", f"{namespace}={{{entries}}}"])
    restrictions.extend(["--disable", "plugins"])

    return restrictions


def run_codex_reviewer(request):
    deadline = time.monotonic() + request.timeout_seconds
    with tempfile.TemporaryDirectory(prefix="agent-review-codex-") as directory:
        answer_path = Path(directory) / "answer.json"
        restrictions = get_codex_restrictions(request, deadline)
        arguments = [
            *restrictions, "exec", "--sandbox", "read-only", "--ephemeral",
            "-c", f"model_reasoning_effort={json.dumps(request.effort)}",
            "--output-schema", str(request.schema_path),
            "--output-last-message", str(answer_path), "-",
        ]

        run_codex_command(request, arguments, deadline)
        if not answer_path.is_file() or not answer_path.read_text().strip():
            raise ValueError("codex returned no final answer file or an empty final answer")
        answer = json.loads(answer_path.read_text())
        if not isinstance(answer, dict):
            raise ValueError("codex returned a non-object final answer")

        return answer
