import json
import os
import platform
import subprocess
import sys
from collections.abc import Callable
from dataclasses import dataclass
from enum import StrEnum
from pathlib import Path
from typing import Protocol

COMMAND_TIMEOUT_SECONDS = 10
SETUP_GUIDE = "docs/how-to-setup-vm.md"
REQUIRED_INOTIFY_INSTANCES = 1024
REQUIRED_INOTIFY_WATCHES = 1048576
MINIMUM_CPU_COUNT = 4
# A guest given 32 GiB reports about 31.3 GiB of MemTotal after the kernel's own reservation.
MINIMUM_MEMORY_KIB = 30 * 1024 * 1024
BUNDLE_ARCHITECTURES = {"x86_64": "x64", "amd64": "x64", "aarch64": "arm", "arm64": "arm"}
LOCKED_TOOL_INSTALLATION = "mise install --locked --include-task-tools"
COMMIT_HOOK_INSTALLATION = "hk install --mise"
# Both hook forms hk installs, the git config entry and the script shim, run this command.
HK_PRE_COMMIT_MARKER = "run pre-commit --from-hook"


class Outcome(StrEnum):
    PASSED = "passed"
    FAILED = "failed"


class CommandStatus(StrEnum):
    COMPLETED = "completed"
    NOT_FOUND = "not found"
    TIMED_OUT = "timed out"


@dataclass(frozen=True)
class CommandOutput:
    status: CommandStatus
    exit_code: int = 0
    stdout: str = ""
    stderr: str = ""


@dataclass(frozen=True)
class PrerequisiteCheckResult:
    name: str
    outcome: Outcome
    detail: str
    guide_section: str


class MachineProbe(Protocol):
    def get_repository_root(self) -> Path: ...

    def read_file(self, path: Path) -> str | None: ...

    def run_command(self, arguments: list[str]) -> CommandOutput: ...

    def get_cpu_count(self) -> int | None: ...

    def get_machine_architecture(self) -> str: ...

    def list_executable_files(self, directory: Path) -> list[Path] | None: ...


class LocalMachineProbe:
    def __init__(
        self, repository_root: Path, command_timeout_seconds: float = COMMAND_TIMEOUT_SECONDS
    ) -> None:
        self.repository_root = repository_root
        self.command_timeout_seconds = command_timeout_seconds

    def get_repository_root(self) -> Path:
        return self.repository_root

    def read_file(self, path: Path) -> str | None:
        try:
            return path.read_text()
        except OSError:
            return None

    def run_command(self, arguments: list[str]) -> CommandOutput:
        try:
            completed = subprocess.run(
                arguments,
                cwd=self.repository_root,
                stdin=subprocess.DEVNULL,
                capture_output=True,
                text=True,
                timeout=self.command_timeout_seconds,
                check=False,
            )
        except FileNotFoundError:
            return CommandOutput(CommandStatus.NOT_FOUND)
        except subprocess.TimeoutExpired:
            return CommandOutput(CommandStatus.TIMED_OUT)

        return CommandOutput(
            CommandStatus.COMPLETED, completed.returncode, completed.stdout, completed.stderr
        )

    def get_cpu_count(self) -> int | None:
        return os.cpu_count()

    def get_machine_architecture(self) -> str:
        return platform.machine()

    def list_executable_files(self, directory: Path) -> list[Path] | None:
        if not directory.is_dir():
            return None

        return sorted(
            path for path in directory.rglob("*") if path.is_file() and os.access(path, os.X_OK)
        )


def build_result(
    name: str, is_met: bool, detail: str, guide_section: str
) -> PrerequisiteCheckResult:
    outcome = Outcome.PASSED if is_met else Outcome.FAILED
    return PrerequisiteCheckResult(name, outcome, detail, guide_section)


def describe_unanswered_command(command: str, output: CommandOutput) -> str:
    if output.status is CommandStatus.NOT_FOUND:
        return f"the `{command.split()[0]}` command is not installed"
    if output.status is CommandStatus.TIMED_OUT:
        return f"`{command}` did not answer within {COMMAND_TIMEOUT_SECONDS} seconds"

    error_lines = output.stderr.strip().splitlines()
    error = f": {error_lines[-1]}" if error_lines else ""
    return f"`{command}` exited with code {output.exit_code}{error}"


def read_integer(probe: MachineProbe, path: Path) -> int | None:
    content = probe.read_file(path)
    if content is None:
        return None

    try:
        return int(content.strip())
    except ValueError:
        return None


def check_bridge_netfilter(probe: MachineProbe) -> PrerequisiteCheckResult:
    setting_path = Path("/proc/sys/net/bridge/bridge-nf-call-iptables")
    value = probe.read_file(setting_path)

    if value is None:
        detail = f"the br_netfilter module is not loaded: {setting_path} does not exist"
    elif value.strip() != "1":
        detail = (
            "the br_netfilter module is loaded, but the setting "
            f"net.bridge.bridge-nf-call-iptables is off, its value is {value.strip()}"
        )
    else:
        detail = "the kernel applies iptables rules to bridged traffic"

    return build_result(
        "Bridge netfilter",
        value is not None and value.strip() == "1",
        detail,
        "bridge-netfilter-for-cloudnativepg-tests",
    )


def check_inotify_limits(probe: MachineProbe) -> PrerequisiteCheckResult:
    limits = {
        "fs.inotify.max_user_instances": REQUIRED_INOTIFY_INSTANCES,
        "fs.inotify.max_user_watches": REQUIRED_INOTIFY_WATCHES,
    }
    problems = []
    for setting, required_value in limits.items():
        current_value = read_integer(probe, Path("/proc/sys") / setting.replace(".", "/"))
        if current_value is None:
            problems.append(
                f"{setting} could not be read, the tests need at least {required_value}"
            )
        elif current_value < required_value:
            problems.append(
                f"{setting} is {current_value}, the tests need at least {required_value}"
            )

    return build_result(
        "inotify",
        not problems,
        "; ".join(problems) or "the limits cover the CloudNativePG tests",
        "inotify-limits-for-cloudnativepg-tests",
    )


def check_docker(probe: MachineProbe) -> PrerequisiteCheckResult:
    command = "docker info --format '{{json .}}'"
    output = probe.run_command(["docker", "info", "--format", "{{json .}}"])

    if output.status is not CommandStatus.COMPLETED or output.exit_code != 0:
        detail = f"Docker does not answer: {describe_unanswered_command(command, output)}"
        return build_result("Docker", False, detail, "docker")

    try:
        docker_info = json.loads(output.stdout)
    except json.JSONDecodeError:
        detail = f"`{command}` printed output that is not JSON"
        return build_result("Docker", False, detail, "docker")

    server_errors = docker_info.get("ServerErrors") or []
    security_options = docker_info.get("SecurityOptions") or []
    if server_errors:
        detail = f"Docker does not answer: {'; '.join(map(str, server_errors))}"
        is_met = False
    elif "name=rootless" in security_options:
        detail = "the Docker daemon runs rootless, but the tests need rootful Docker"
        is_met = False
    else:
        detail = f"the rootful Docker daemon answers, version {docker_info.get('ServerVersion')}"
        is_met = True

    return build_result("Docker", is_met, detail, "docker")


def check_docker_compose(probe: MachineProbe) -> PrerequisiteCheckResult:
    command = "docker compose version"
    output = probe.run_command(["docker", "compose", "version"])
    is_met = output.status is CommandStatus.COMPLETED and output.exit_code == 0

    if is_met:
        detail = output.stdout.strip()
    else:
        detail = (
            f"the Docker Compose plugin is missing: {describe_unanswered_command(command, output)}"
        )

    return build_result("Docker Compose", is_met, detail, "docker")


def find_unresolved_dependencies(probe: MachineProbe, program: Path) -> list[str] | str:
    # -r also resolves symbols, so a library that is present but older than the program needs,
    # such as an old libpq5, is reported instead of failing only when the program starts.
    output = probe.run_command(["ldd", "-r", str(program)])
    if output.status is CommandStatus.COMPLETED and "not a dynamic executable" in (
        output.stdout + output.stderr
    ):
        return []
    if output.status is not CommandStatus.COMPLETED or output.exit_code != 0:
        return describe_unanswered_command(f"ldd -r {program}", output)

    unresolved_dependencies = []
    for line in (output.stdout + output.stderr).splitlines():
        if line.strip().endswith("not found"):
            unresolved_dependencies.append(line.split("=>")[0].strip())
        elif line.startswith("undefined symbol:"):
            symbol = line.removeprefix("undefined symbol:").split("\t")[0].strip()
            unresolved_dependencies.append(f"the symbol {symbol}")

    return unresolved_dependencies


def check_client_tool_bundles(probe: MachineProbe) -> PrerequisiteCheckResult:
    name = "Client tool bundles"
    guide_section = "client-libraries"
    machine_architecture = probe.get_machine_architecture()
    bundle_architecture = BUNDLE_ARCHITECTURES.get(machine_architecture)
    if bundle_architecture is None:
        detail = f"there are no client tool bundles for the architecture {machine_architecture}"
        return build_result(name, False, detail, guide_section)

    repository_root = probe.get_repository_root()
    bundle_directory = repository_root / "assets" / "tools" / bundle_architecture
    programs = probe.list_executable_files(bundle_directory)
    if not programs:
        detail = f"{bundle_directory.relative_to(repository_root)} holds no programs"
        return build_result(name, False, detail, guide_section)

    problems = []
    for program in programs:
        unresolved_dependencies = find_unresolved_dependencies(probe, program)
        relative_program = program.relative_to(repository_root)
        if isinstance(unresolved_dependencies, str):
            problems.append(f"{relative_program} could not be checked: {unresolved_dependencies}")
        elif unresolved_dependencies:
            problems.append(f"{relative_program} needs {', '.join(unresolved_dependencies)}")

    if problems:
        detail = "unresolved shared libraries or symbols: " + "; ".join(problems)
    else:
        detail = f"all {len(programs)} programs find their shared libraries and symbols"

    return build_result(name, not problems, detail, guide_section)


def run_mise_listing(probe: MachineProbe, arguments: list[str]) -> CommandOutput | str:
    output = probe.run_command(arguments)
    if output.status is not CommandStatus.COMPLETED or output.exit_code != 0:
        return describe_unanswered_command(" ".join(arguments), output)

    return output


def list_missing_configuration_tools(probe: MachineProbe) -> list[str] | str:
    output = run_mise_listing(probe, ["mise", "ls", "--current", "--missing", "--no-truncate"])
    if isinstance(output, str):
        return output

    return [line.split()[0] for line in output.stdout.splitlines() if line.strip()]


def list_missing_task_tools(probe: MachineProbe) -> list[str] | str:
    tasks_output = run_mise_listing(probe, ["mise", "tasks", "--hidden", "--json"])
    if isinstance(tasks_output, str):
        return tasks_output
    installed_output = run_mise_listing(probe, ["mise", "ls", "--installed", "--json"])
    if isinstance(installed_output, str):
        return installed_output

    try:
        tasks = json.loads(tasks_output.stdout)
        installed_tools = json.loads(installed_output.stdout)
    except json.JSONDecodeError:
        return "mise printed output that is not JSON"

    missing_tools = []
    for task in tasks:
        for tool, version in (task.get("tools") or {}).items():
            installed_versions = {
                installation.get("version") for installation in installed_tools.get(tool, [])
            }
            if version not in installed_versions:
                missing_tools.append(f"{tool}@{version} of the task {task.get('name')}")

    return missing_tools


def check_pinned_tools(probe: MachineProbe) -> PrerequisiteCheckResult:
    problems = []
    missing_tools = []
    for listing in (list_missing_configuration_tools(probe), list_missing_task_tools(probe)):
        if isinstance(listing, str):
            problems.append(f"the pinned tools could not be listed: {listing}")
        else:
            missing_tools.extend(listing)

    if missing_tools:
        problems.append(
            f"not installed: {', '.join(missing_tools)}; "
            f"install them with `{LOCKED_TOOL_INSTALLATION}`"
        )

    return build_result(
        "Pinned tools",
        not problems,
        "; ".join(problems) or "every tool pinned in mise.toml, its tasks included, is installed",
        "mise-and-the-checkout",
    )


def check_ubuntu(probe: MachineProbe) -> PrerequisiteCheckResult:
    os_release = probe.read_file(Path("/etc/os-release"))
    system_id = None
    for line in (os_release or "").splitlines():
        if line.startswith("ID="):
            system_id = line.removeprefix("ID=").strip().strip("\"'")

    if system_id == "ubuntu":
        detail = "the system is Ubuntu"
    elif system_id is None:
        detail = "/etc/os-release names no system, and the project's tooling targets Ubuntu"
    else:
        detail = f"the system is {system_id}, but the project's tooling targets Ubuntu"

    return build_result("Ubuntu", system_id == "ubuntu", detail, "guest-system")


def check_ext4_root(probe: MachineProbe) -> PrerequisiteCheckResult:
    command = "findmnt -no FSTYPE /"
    output = probe.run_command(["findmnt", "-no", "FSTYPE", "/"])

    if output.status is not CommandStatus.COMPLETED or output.exit_code != 0:
        detail = (
            f"the root filesystem could not be read: {describe_unanswered_command(command, output)}"
            ", and the tests need an ext4 root filesystem"
        )
        is_met = False
    else:
        filesystem = output.stdout.strip()
        is_met = filesystem == "ext4"
        detail = f"the root filesystem is {filesystem}"
        if not is_met:
            detail += ", but the tests need an ext4 root filesystem"

    return build_result("ext4", is_met, detail, "disk")


def check_cpu_count(probe: MachineProbe) -> PrerequisiteCheckResult:
    cpu_count = probe.get_cpu_count()

    if cpu_count is None:
        detail = f"the number of logical CPUs is unknown, the minimum is {MINIMUM_CPU_COUNT}"
    elif cpu_count < MINIMUM_CPU_COUNT:
        detail = (
            f"the machine has {cpu_count} logical CPUs, "
            f"fewer than the minimum of {MINIMUM_CPU_COUNT} logical CPUs"
        )
    else:
        detail = f"the machine has {cpu_count} logical CPUs"

    return build_result(
        "CPUs",
        cpu_count is not None and cpu_count >= MINIMUM_CPU_COUNT,
        detail,
        "cpu",
    )


def read_total_memory_kib(probe: MachineProbe) -> int | None:
    for line in (probe.read_file(Path("/proc/meminfo")) or "").splitlines():
        fields = line.split()
        if len(fields) >= 2 and fields[0] == "MemTotal:" and fields[1].isdigit():
            return int(fields[1])

    return None


def check_memory(probe: MachineProbe) -> PrerequisiteCheckResult:
    total_memory_kib = read_total_memory_kib(probe)

    if total_memory_kib is None:
        detail = "MemTotal could not be read from /proc/meminfo, the minimum is 32 GiB of memory"
    else:
        total_memory = f"{total_memory_kib / 1024 / 1024:.1f} GiB"
        if total_memory_kib < MINIMUM_MEMORY_KIB:
            detail = (
                f"the machine has {total_memory} of memory, "
                "less than the minimum of 32 GiB of memory"
            )
        else:
            detail = f"the machine has {total_memory} of memory"

    return build_result(
        "Memory",
        total_memory_kib is not None and total_memory_kib >= MINIMUM_MEMORY_KIB,
        detail,
        "memory",
    )


def is_commit_hook_installed(probe: MachineProbe) -> bool:
    configured_hook = probe.run_command(
        ["git", "config", "--get-all", "hook.hk-pre-commit.command"]
    )
    if configured_hook.status is CommandStatus.COMPLETED and (
        HK_PRE_COMMIT_MARKER in configured_hook.stdout
    ):
        return True

    hook_path = probe.run_command(["git", "rev-parse", "--git-path", "hooks/pre-commit"])
    if hook_path.status is not CommandStatus.COMPLETED or hook_path.exit_code != 0:
        return False

    hook_script = probe.read_file(probe.get_repository_root() / hook_path.stdout.strip())
    return hook_script is not None and HK_PRE_COMMIT_MARKER in hook_script


def check_commit_hook(probe: MachineProbe) -> PrerequisiteCheckResult:
    is_met = is_commit_hook_installed(probe)

    if is_met:
        detail = "hk runs on every commit"
    else:
        detail = (
            f"the hk commit hook is not installed; install it with `{COMMIT_HOOK_INSTALLATION}`"
        )

    return build_result("Commit hook", is_met, detail, "mise-and-the-checkout")


PREREQUISITE_CHECKS: tuple[Callable[[MachineProbe], PrerequisiteCheckResult], ...] = (
    check_bridge_netfilter,
    check_inotify_limits,
    check_docker,
    check_docker_compose,
    check_client_tool_bundles,
    check_pinned_tools,
    check_ubuntu,
    check_ext4_root,
    check_cpu_count,
    check_memory,
    check_commit_hook,
)


def run_checks(probe: MachineProbe) -> list[PrerequisiteCheckResult]:
    return [check(probe) for check in PREREQUISITE_CHECKS]


def format_report(results: list[PrerequisiteCheckResult]) -> str:
    lines = ["Development environment:"]
    for result in results:
        line = f"{result.name}: {result.outcome}, {result.detail}"
        if result.outcome is not Outcome.PASSED:
            line += f". See {SETUP_GUIDE}#{result.guide_section}"
        lines.append(line)

    return "\n".join(lines)


def get_exit_code(results: list[PrerequisiteCheckResult]) -> int:
    return int(any(result.outcome is Outcome.FAILED for result in results))


def main() -> int:
    results = run_checks(LocalMachineProbe(Path(__file__).resolve().parent.parent))
    print(format_report(results))

    return get_exit_code(results)


if __name__ == "__main__":
    sys.exit(main())
