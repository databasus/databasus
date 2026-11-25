import json
import re
import tempfile
import unittest
from dataclasses import dataclass, field
from pathlib import Path

from development_environment_check import (
    COMMAND_TIMEOUT_SECONDS,
    CommandOutput,
    CommandStatus,
    LocalMachineProbe,
    Outcome,
    PrerequisiteCheckResult,
    format_report,
    get_exit_code,
    run_checks,
)

REPOSITORY_ROOT = Path(__file__).resolve().parent.parent
FAKE_ROOT = Path("/repository")
GIB_IN_KIB = 1024 * 1024
BUNDLE_PROGRAMS = (
    FAKE_ROOT / "assets/tools/x64/postgresql/postgresql-17/bin/pg_dump",
    FAKE_ROOT / "assets/tools/x64/mariadb/mariadb-12.3/bin/mariadb",
)
CLEAN_LDD_OUTPUT = CommandOutput(
    CommandStatus.COMPLETED,
    stdout="\tlinux-vdso.so.1 (0x00007ffd)\n\tlibc.so.6 => /lib/x86_64-linux-gnu/libc.so.6 (0x7f)\n",
)
READ_ONLY_COMMANDS = {
    ("docker", "info"),
    ("docker", "compose", "version"),
    ("ldd",),
    ("mise", "ls"),
    ("mise", "tasks"),
    ("findmnt",),
    ("git", "config", "--get-all"),
    ("git", "rev-parse", "--git-path"),
}


def completed(stdout: str = "", exit_code: int = 0, stderr: str = "") -> CommandOutput:
    return CommandOutput(CommandStatus.COMPLETED, exit_code, stdout, stderr)


def get_memory_info(total_memory_gib: float) -> str:
    return f"MemTotal:       {int(total_memory_gib * GIB_IN_KIB)} kB\nMemFree: 1 kB\n"


@dataclass
class FakeMachineProbe:
    files: dict[str, str] = field(
        default_factory=lambda: {
            "/proc/sys/net/bridge/bridge-nf-call-iptables": "1\n",
            "/proc/sys/fs/inotify/max_user_instances": "1024\n",
            "/proc/sys/fs/inotify/max_user_watches": "1048576\n",
            "/etc/os-release": 'NAME="Ubuntu"\nID=ubuntu\nID_LIKE=debian\n',
            "/proc/meminfo": get_memory_info(31.3),
        }
    )
    commands: dict[tuple[str, ...], CommandOutput] = field(
        default_factory=lambda: {
            ("docker", "info", "--format", "{{json .}}"): completed(
                json.dumps(
                    {"ServerVersion": "29.0.0", "SecurityOptions": ["name=seccomp,profile=builtin"]}
                )
            ),
            ("docker", "compose", "version"): completed("Docker Compose version v2.40.0\n"),
            ("mise", "ls", "--current", "--missing", "--no-truncate"): completed(),
            ("mise", "tasks", "--hidden", "--json"): completed(
                json.dumps(
                    [
                        {"name": "lint", "tools": {}},
                        {"name": "hooks:migrate", "tools": {"lefthook": "2.2.1"}},
                    ]
                )
            ),
            ("mise", "ls", "--installed", "--json"): completed(
                json.dumps({"lefthook": [{"version": "2.1.14"}, {"version": "2.2.1"}]})
            ),
            ("findmnt", "-no", "FSTYPE", "/"): completed("ext4\n"),
            ("git", "config", "--get-all", "hook.hk-pre-commit.command"): completed(
                'test "${HK:-1}" = "0" || mise x -- hk run pre-commit --from-hook\n'
            ),
            ("git", "rev-parse", "--git-path", "hooks/pre-commit"): completed(
                ".git/hooks/pre-commit\n"
            ),
        }
    )
    ldd_outputs: dict[Path, CommandOutput] = field(default_factory=dict)
    cpu_count: int | None = 4
    architecture: str = "x86_64"
    executables: dict[Path, list[Path]] = field(
        default_factory=lambda: {FAKE_ROOT / "assets/tools/x64": list(BUNDLE_PROGRAMS)}
    )
    run_commands: list[list[str]] = field(default_factory=list)

    def get_repository_root(self) -> Path:
        return FAKE_ROOT

    def read_file(self, path: Path) -> str | None:
        return self.files.get(str(path))

    def run_command(self, arguments: list[str]) -> CommandOutput:
        self.run_commands.append(arguments)
        if arguments[0] == "ldd":
            return self.ldd_outputs.get(Path(arguments[-1]), CLEAN_LDD_OUTPUT)

        return self.commands.get(tuple(arguments), CommandOutput(CommandStatus.NOT_FOUND))

    def get_cpu_count(self) -> int | None:
        return self.cpu_count

    def get_machine_architecture(self) -> str:
        return self.architecture

    def list_executable_files(self, directory: Path) -> list[Path] | None:
        return self.executables.get(directory)


def get_result(probe: FakeMachineProbe, name: str) -> PrerequisiteCheckResult:
    return next(result for result in run_checks(probe) if result.name == name)


def get_guide_anchors() -> set[str]:
    guide = (REPOSITORY_ROOT / "docs/how-to-setup-vm.md").read_text()
    headings = re.findall(r"^#+ (.+)$", guide, flags=re.MULTILINE)

    return {re.sub(r"[^\w\- ]", "", heading.lower()).replace(" ", "-") for heading in headings}


def break_every_prerequisite(probe: FakeMachineProbe) -> None:
    probe.files.clear()
    probe.commands.clear()
    probe.cpu_count = 1


class DevelopmentEnvironmentReportTest(unittest.TestCase):
    def test_reports_every_check_and_exits_with_1_when_one_required_check_fails(self) -> None:
        probe = FakeMachineProbe()
        del probe.files["/proc/sys/net/bridge/bridge-nf-call-iptables"]

        results = run_checks(probe)
        report = format_report(results)

        self.assertEqual(get_exit_code(results), 1)
        self.assertEqual(len(report.splitlines()), 12)
        self.assertEqual(
            [result.outcome for result in results], [Outcome.FAILED] + [Outcome.PASSED] * 10
        )
        self.assertIn(
            "Bridge netfilter: failed, the br_netfilter module is not loaded",
            report.splitlines()[1],
        )
        self.assertIn(
            "See docs/how-to-setup-vm.md#bridge-netfilter-for-cloudnativepg-tests", report
        )

    def test_exits_with_0_only_when_every_check_passes(self) -> None:
        results = run_checks(FakeMachineProbe())

        self.assertEqual(get_exit_code(results), 0)
        self.assertEqual({result.outcome for result in results}, {Outcome.PASSED})

    def test_exits_with_1_when_only_the_system_and_the_commit_hook_are_missing(self) -> None:
        probe = FakeMachineProbe()
        probe.files["/etc/os-release"] = "ID=fedora\n"
        del probe.commands[("git", "config", "--get-all", "hook.hk-pre-commit.command")]

        results = run_checks(probe)

        self.assertEqual(get_exit_code(results), 1)
        self.assertEqual(
            {result.name for result in results if result.outcome is Outcome.FAILED},
            {"Ubuntu", "Commit hook"},
        )

    def test_names_only_sections_of_the_setup_guide_in_hints(self) -> None:
        probe = FakeMachineProbe()
        break_every_prerequisite(probe)
        probe.architecture = "riscv64"
        guide_anchors = get_guide_anchors()

        for result in run_checks(probe):
            with self.subTest(check=result.name):
                self.assertNotEqual(result.outcome, Outcome.PASSED)
                self.assertIn(result.guide_section, guide_anchors)


class KernelSettingsCheckTest(unittest.TestCase):
    def test_fails_when_the_bridge_setting_is_missing(self) -> None:
        probe = FakeMachineProbe()
        del probe.files["/proc/sys/net/bridge/bridge-nf-call-iptables"]

        bridge_result = get_result(probe, "Bridge netfilter")

        self.assertEqual(bridge_result.outcome, Outcome.FAILED)
        self.assertIn("not loaded", bridge_result.detail)

    def test_fails_and_says_the_setting_is_off_when_the_bridge_setting_is_0(self) -> None:
        probe = FakeMachineProbe()
        probe.files["/proc/sys/net/bridge/bridge-nf-call-iptables"] = "0\n"

        bridge_result = get_result(probe, "Bridge netfilter")

        self.assertEqual(bridge_result.outcome, Outcome.FAILED)
        self.assertIn("is off", bridge_result.detail)

    def test_fails_with_current_and_required_value_when_instances_keep_the_kernel_default(
        self,
    ) -> None:
        probe = FakeMachineProbe()
        probe.files["/proc/sys/fs/inotify/max_user_instances"] = "128\n"

        inotify_result = get_result(probe, "inotify")

        self.assertEqual(inotify_result.outcome, Outcome.FAILED)
        self.assertIn(
            "max_user_instances is 128, the tests need at least 1024", inotify_result.detail
        )

    def test_fails_when_watches_are_below_the_requirement(self) -> None:
        probe = FakeMachineProbe()
        probe.files["/proc/sys/fs/inotify/max_user_watches"] = "8192\n"

        inotify_result = get_result(probe, "inotify")

        self.assertEqual(inotify_result.outcome, Outcome.FAILED)
        self.assertIn("max_user_watches is 8192", inotify_result.detail)

    def test_passes_with_exactly_the_required_and_with_higher_limits(self) -> None:
        for instances, watches in (("1024", "1048576"), ("8192", "2097152")):
            with self.subTest(instances=instances, watches=watches):
                probe = FakeMachineProbe()
                probe.files["/proc/sys/fs/inotify/max_user_instances"] = instances
                probe.files["/proc/sys/fs/inotify/max_user_watches"] = watches

                self.assertEqual(get_result(probe, "inotify").outcome, Outcome.PASSED)


class DockerCheckTest(unittest.TestCase):
    docker_info_command = ("docker", "info", "--format", "{{json .}}")

    def test_passes_with_a_rootful_daemon(self) -> None:
        docker_result = get_result(FakeMachineProbe(), "Docker")

        self.assertEqual(docker_result.outcome, Outcome.PASSED)

    def test_fails_and_asks_for_rootful_docker_with_a_rootless_daemon(self) -> None:
        probe = FakeMachineProbe()
        probe.commands[self.docker_info_command] = completed(
            json.dumps({"SecurityOptions": ["name=seccomp,profile=builtin", "name=rootless"]})
        )

        docker_result = get_result(probe, "Docker")

        self.assertEqual(docker_result.outcome, Outcome.FAILED)
        self.assertIn("the tests need rootful Docker", docker_result.detail)

    def test_fails_without_the_docker_command(self) -> None:
        probe = FakeMachineProbe()
        probe.commands[self.docker_info_command] = CommandOutput(CommandStatus.NOT_FOUND)

        docker_result = get_result(probe, "Docker")

        self.assertEqual(docker_result.outcome, Outcome.FAILED)
        self.assertIn("the `docker` command is not installed", docker_result.detail)

    def test_fails_when_docker_info_exits_with_an_error(self) -> None:
        probe = FakeMachineProbe()
        probe.commands[self.docker_info_command] = completed(
            exit_code=1,
            stderr="Cannot connect to the Docker daemon at unix:///var/run/docker.sock\n",
        )

        docker_result = get_result(probe, "Docker")

        self.assertEqual(docker_result.outcome, Outcome.FAILED)
        self.assertIn(
            "exited with code 1: Cannot connect to the Docker daemon", docker_result.detail
        )

    def test_reports_every_other_check_after_docker_times_out(self) -> None:
        probe = FakeMachineProbe()
        probe.commands[self.docker_info_command] = CommandOutput(CommandStatus.TIMED_OUT)

        results = run_checks(probe)

        self.assertEqual(len(results), 11)
        self.assertEqual(
            [result.name for result in results if result.outcome is not Outcome.PASSED],
            ["Docker"],
        )
        self.assertIn("did not answer within 10 seconds", results[2].detail)

    def test_fails_when_docker_info_reports_server_errors(self) -> None:
        probe = FakeMachineProbe()
        probe.commands[self.docker_info_command] = completed(
            json.dumps({"ServerErrors": ["Cannot connect to the Docker daemon"]})
        )

        docker_result = get_result(probe, "Docker")

        self.assertEqual(docker_result.outcome, Outcome.FAILED)
        self.assertIn("Cannot connect to the Docker daemon", docker_result.detail)

    def test_fails_and_names_the_plugin_when_compose_is_missing(self) -> None:
        probe = FakeMachineProbe()
        probe.commands[("docker", "compose", "version")] = completed(
            exit_code=1, stderr="docker: unknown command: docker compose\n"
        )

        compose_result = get_result(probe, "Docker Compose")

        self.assertEqual(compose_result.outcome, Outcome.FAILED)
        self.assertIn("the Docker Compose plugin is missing", compose_result.detail)


class ClientToolBundlesCheckTest(unittest.TestCase):
    def test_passes_with_clean_ldd_output(self) -> None:
        bundles_result = get_result(FakeMachineProbe(), "Client tool bundles")

        self.assertEqual(bundles_result.outcome, Outcome.PASSED)

    def test_names_the_program_and_library_when_a_library_is_not_found(self) -> None:
        probe = FakeMachineProbe()
        probe.ldd_outputs[BUNDLE_PROGRAMS[1]] = completed(
            "\tlibedit.so.2 => not found\n\tlibc.so.6 => /lib64/libc.so.6 (0x7f)\n"
        )

        bundles_result = get_result(probe, "Client tool bundles")

        self.assertEqual(bundles_result.outcome, Outcome.FAILED)
        self.assertIn(
            "assets/tools/x64/mariadb/mariadb-12.3/bin/mariadb needs libedit.so.2",
            bundles_result.detail,
        )
        self.assertNotIn("pg_dump", bundles_result.detail)

    def test_names_the_program_and_symbol_when_a_library_is_too_old(self) -> None:
        probe = FakeMachineProbe()
        probe.ldd_outputs[BUNDLE_PROGRAMS[0]] = completed(
            "\tlibpq.so.5 => /lib/x86_64-linux-gnu/libpq.so.5 (0x7f)\n"
            "undefined symbol: PQchangePassword\t(./psql)\n"
        )

        bundles_result = get_result(probe, "Client tool bundles")

        self.assertEqual(bundles_result.outcome, Outcome.FAILED)
        self.assertIn(
            "assets/tools/x64/postgresql/postgresql-17/bin/pg_dump needs the symbol PQchangePassword",
            bundles_result.detail,
        )
        self.assertNotIn("mariadb", bundles_result.detail)

    def test_passes_with_a_statically_linked_program(self) -> None:
        probe = FakeMachineProbe()
        probe.ldd_outputs[BUNDLE_PROGRAMS[0]] = completed(
            exit_code=1, stderr="\tnot a dynamic executable\n"
        )

        self.assertEqual(get_result(probe, "Client tool bundles").outcome, Outcome.PASSED)

    def test_fails_and_names_an_unknown_architecture(self) -> None:
        probe = FakeMachineProbe()
        probe.architecture = "riscv64"

        bundles_result = get_result(probe, "Client tool bundles")

        self.assertEqual(bundles_result.outcome, Outcome.FAILED)
        self.assertIn("riscv64", bundles_result.detail)

    def test_reads_the_arm_bundles_on_aarch64(self) -> None:
        probe = FakeMachineProbe()
        probe.architecture = "aarch64"
        arm_program = FAKE_ROOT / "assets/tools/arm/mongodb/bin/mongodump"
        probe.executables = {FAKE_ROOT / "assets/tools/arm": [arm_program]}

        get_result(probe, "Client tool bundles")

        self.assertIn(["ldd", "-r", str(arm_program)], probe.run_commands)


class PinnedToolsCheckTest(unittest.TestCase):
    missing_tools_command = ("mise", "ls", "--current", "--missing", "--no-truncate")

    def test_passes_when_no_tool_is_missing(self) -> None:
        self.assertEqual(get_result(FakeMachineProbe(), "Pinned tools").outcome, Outcome.PASSED)

    def test_names_each_missing_tool_and_the_locked_installation(self) -> None:
        probe = FakeMachineProbe()
        probe.commands[self.missing_tools_command] = completed(
            "go     1.26.3 (missing)  ~/databasus/mise.toml  1.26.3\n"
            "node   24.11.0 (missing) ~/databasus/mise.toml  24.11.0\n"
        )

        tools_result = get_result(probe, "Pinned tools")

        self.assertEqual(tools_result.outcome, Outcome.FAILED)
        self.assertIn("not installed: go, node", tools_result.detail)
        self.assertIn("mise install --locked --include-task-tools", tools_result.detail)

    def test_names_a_missing_task_tool_and_its_task(self) -> None:
        probe = FakeMachineProbe()
        probe.commands[("mise", "ls", "--installed", "--json")] = completed(
            json.dumps({"lefthook": [{"version": "2.1.14"}]})
        )

        tools_result = get_result(probe, "Pinned tools")

        self.assertEqual(tools_result.outcome, Outcome.FAILED)
        self.assertIn("lefthook@2.2.1 of the task hooks:migrate", tools_result.detail)
        self.assertIn("mise install --locked --include-task-tools", tools_result.detail)

    def test_fails_when_mise_fails(self) -> None:
        for failing_command in (
            self.missing_tools_command,
            ("mise", "tasks", "--hidden", "--json"),
            ("mise", "ls", "--installed", "--json"),
        ):
            with self.subTest(command=failing_command):
                probe = FakeMachineProbe()
                probe.commands[failing_command] = completed(exit_code=1, stderr="mise ERROR\n")

                tools_result = get_result(probe, "Pinned tools")

                self.assertEqual(tools_result.outcome, Outcome.FAILED)
                self.assertIn("exited with code 1", tools_result.detail)


class MachineCheckTest(unittest.TestCase):
    def test_fails_on_btrfs_with_2_cpus_and_16_gib_naming_each_minimum(self) -> None:
        probe = FakeMachineProbe()
        probe.commands[("findmnt", "-no", "FSTYPE", "/")] = completed("btrfs\n")
        probe.cpu_count = 2
        probe.files["/proc/meminfo"] = get_memory_info(15.6)

        ext4_result = get_result(probe, "ext4")
        cpu_result = get_result(probe, "CPUs")
        memory_result = get_result(probe, "Memory")

        self.assertEqual(
            {ext4_result.outcome, cpu_result.outcome, memory_result.outcome}, {Outcome.FAILED}
        )
        self.assertIn("the tests need an ext4 root filesystem", ext4_result.detail)
        self.assertIn("fewer than the minimum of 4 logical CPUs", cpu_result.detail)
        self.assertIn("15.6 GiB", memory_result.detail)
        self.assertIn("less than the minimum of 32 GiB of memory", memory_result.detail)

    def test_passes_with_an_ext4_root_and_exactly_4_cpus(self) -> None:
        probe = FakeMachineProbe()

        self.assertEqual(get_result(probe, "ext4").outcome, Outcome.PASSED)
        self.assertEqual(get_result(probe, "CPUs").outcome, Outcome.PASSED)
        self.assertEqual(get_result(probe, "Memory").outcome, Outcome.PASSED)

    def test_passes_with_exactly_30_gib_and_fails_with_29_9_gib(self) -> None:
        for total_memory_gib, expected_outcome in ((30, Outcome.PASSED), (29.9, Outcome.FAILED)):
            with self.subTest(total_memory_gib=total_memory_gib):
                probe = FakeMachineProbe()
                probe.files["/proc/meminfo"] = get_memory_info(total_memory_gib)

                self.assertEqual(get_result(probe, "Memory").outcome, expected_outcome)

    def test_fails_and_says_the_tooling_targets_ubuntu_on_fedora(self) -> None:
        probe = FakeMachineProbe()
        probe.files["/etc/os-release"] = 'NAME="Fedora Linux"\nID=fedora\n'

        ubuntu_result = get_result(probe, "Ubuntu")

        self.assertEqual(ubuntu_result.outcome, Outcome.FAILED)
        self.assertIn("the project's tooling targets Ubuntu", ubuntu_result.detail)


class CommitHookCheckTest(unittest.TestCase):
    configured_hook_command = ("git", "config", "--get-all", "hook.hk-pre-commit.command")

    def test_passes_with_the_hook_hk_writes_to_git_config(self) -> None:
        self.assertEqual(get_result(FakeMachineProbe(), "Commit hook").outcome, Outcome.PASSED)

    def test_passes_with_the_hook_script_hk_writes_for_older_git(self) -> None:
        probe = FakeMachineProbe()
        del probe.commands[self.configured_hook_command]
        probe.files[str(FAKE_ROOT / ".git/hooks/pre-commit")] = (
            '#!/bin/sh\ntest "${HK:-1}" = "0" || exec hk run pre-commit --from-hook "$@"\n'
        )

        self.assertEqual(get_result(probe, "Commit hook").outcome, Outcome.PASSED)

    def test_fails_with_the_installation_command_when_no_hook_is_installed(self) -> None:
        probe = FakeMachineProbe()
        del probe.commands[self.configured_hook_command]

        hook_result = get_result(probe, "Commit hook")

        self.assertEqual(hook_result.outcome, Outcome.FAILED)
        self.assertIn("hk install --mise", hook_result.detail)

    def test_fails_when_the_hook_script_belongs_to_another_tool(self) -> None:
        probe = FakeMachineProbe()
        del probe.commands[self.configured_hook_command]
        probe.files[str(FAKE_ROOT / ".git/hooks/pre-commit")] = "#!/bin/sh\npre-commit run\n"

        self.assertEqual(get_result(probe, "Commit hook").outcome, Outcome.FAILED)


class ReadOnlyCheckTest(unittest.TestCase):
    def test_runs_only_read_only_commands_whatever_the_outcome(self) -> None:
        healthy_probe = FakeMachineProbe()
        broken_probe = FakeMachineProbe()
        break_every_prerequisite(broken_probe)

        for probe in (healthy_probe, broken_probe):
            run_checks(probe)
            for arguments in probe.run_commands:
                with self.subTest(command=arguments):
                    self.assertTrue(
                        any(
                            tuple(arguments[: len(prefix)]) == prefix
                            for prefix in READ_ONLY_COMMANDS
                        )
                    )


class LocalMachineProbeTest(unittest.TestCase):
    def setUp(self) -> None:
        directory = tempfile.TemporaryDirectory(prefix="machine-probe-")
        self.addCleanup(directory.cleanup)
        self.repository_root = Path(directory.name)

    def test_waits_10_seconds_for_a_command_by_default(self) -> None:
        self.assertEqual(COMMAND_TIMEOUT_SECONDS, 10)
        self.assertEqual(LocalMachineProbe(self.repository_root).command_timeout_seconds, 10)

    def test_reports_a_timeout_when_a_command_hangs(self) -> None:
        probe = LocalMachineProbe(self.repository_root, command_timeout_seconds=0.1)

        hung_command = probe.run_command(["sleep", "5"])

        self.assertEqual(hung_command.status, CommandStatus.TIMED_OUT)

    def test_reports_a_missing_command(self) -> None:
        probe = LocalMachineProbe(self.repository_root)

        missing_command = probe.run_command(["databasus-command-that-does-not-exist"])

        self.assertEqual(missing_command.status, CommandStatus.NOT_FOUND)

    def test_reports_the_exit_code_and_output_of_a_completed_command(self) -> None:
        probe = LocalMachineProbe(self.repository_root)

        finished_command = probe.run_command(["sh", "-c", "pwd; echo failed >&2; exit 3"])

        self.assertEqual(finished_command.status, CommandStatus.COMPLETED)
        self.assertEqual(finished_command.exit_code, 3)
        self.assertEqual(
            Path(finished_command.stdout.strip()).resolve(), self.repository_root.resolve()
        )
        self.assertEqual(finished_command.stderr, "failed\n")


if __name__ == "__main__":
    unittest.main()
