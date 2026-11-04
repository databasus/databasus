import re
from pathlib import Path

from compose_project import remove_dead_run_project
from docker_wrapper import install_docker_wrapper_and_get_path
from probe_run import assert_failed_and_released, start_probe_run
from public_command import PublicRun
from supervisor_file import remove_run_directory

PROBE_PASSWORD = "synthetic-account-password"
PROBE_DATABASE_PASSWORD = "synthetic-database-password"
BEARER_TOKEN = "synthetic-bearer-marker"
STARTUP_FAILING_IMAGE = "postgres:18@sha256:3a4264520e121176619f54ff867d12774bc81e95e7b84db58d63bccdb6144b81"


class TestFailingRuns:
    def test_refuses_an_unknown_probe_because_no_scenario_would_run(self):
        finished_run = PublicRun(environment={"E2E_PROBE": "mistyped"}).finish()

        assert finished_run.exit_code != 0
        assert "E2E refuses the unknown probe mistyped" in finished_run.output

    def test_fails_with_the_diagnostic_and_removes_everything_when_a_scenario_fails(self, probes_image: str):
        assert_failed_and_released(
            start_probe_run(probes_image, "scenario"),
            "deliberate E2E scenario failure after readiness",
        )

    def test_shows_the_refusal_when_a_form_sends_a_wrong_database_password(self, probes_image: str):
        output = assert_failed_and_released(
            start_probe_run(probes_image, "refused-form"),
            "the application refused the connection form",
        )

        assert re.search(r"the application refused the connection form: .*password authentication failed", output), output

    def test_prints_the_log_end_and_removes_everything_when_the_application_never_becomes_ready(self):
        output = assert_failed_and_released(
            PublicRun(environment={"DATABASUS_IMAGE": STARTUP_FAILING_IMAGE}),
            "E2E start-up failed",
        )

        assert "superuser password" in output

    def test_masks_secrets_in_every_stream_a_failed_run_prints(self, probes_image: str):
        output = assert_failed_and_released(
            start_probe_run(
                probes_image,
                "secrets",
                E2E_PROBE_PASSWORD=PROBE_PASSWORD,
                E2E_PROBE_DATABASE_PASSWORD=PROBE_DATABASE_PASSWORD,
            ),
            "deliberate E2E secrets failure",
        )

        for stream in ("probe-stream-pytest", "probe-stream-databases", "probe-stream-application"):
            assert stream in output, f"{stream} did not reach the output:\n{output}"
        assert "synthetic-local-value" not in output
        assert PROBE_PASSWORD not in output
        assert PROBE_DATABASE_PASSWORD not in output
        assert BEARER_TOKEN not in output

    def test_fails_and_names_the_remains_when_a_resource_cannot_be_removed_after_every_scenario_passed(
        self, probes_image: str, tmp_path: Path
    ):
        path_with_refusal = install_docker_wrapper_and_get_path(tmp_path, '*" down "*) echo "removal refused"; exit 1;;')

        run = start_probe_run(
            probes_image,
            "passing",
            PATH=path_with_refusal,
            E2E_PROBE_PASSWORD=PROBE_PASSWORD,
            E2E_PROBE_DATABASE_PASSWORD=PROBE_DATABASE_PASSWORD,
        )
        project_name = run.get_project_name()

        try:
            finished_run = run.finish()

            assert finished_run.exit_code != 0
            assert "1 passed" in finished_run.output, finished_run.output
            assert f"E2E cleanup failed, remaining: container {project_name}-" in finished_run.output
            assert "probe-cleanup-databases" in finished_run.output
            assert "probe-cleanup-application" in finished_run.output
            assert PROBE_PASSWORD not in finished_run.output
            assert PROBE_DATABASE_PASSWORD not in finished_run.output
            assert BEARER_TOKEN not in finished_run.output
        finally:
            remove_dead_run_project(project_name)
            remove_run_directory(run.get_run_id())

    def test_removes_everything_when_the_run_fails_outside_a_scenario(
        self, probes_image: str, tmp_path: Path
    ):
        path_without_port = install_docker_wrapper_and_get_path(tmp_path, '*" port browser "*) echo "port refused" >&2; exit 1;;')

        assert_failed_and_released(
            start_probe_run(probes_image, "scenario", PATH=path_without_port),
            "E2E run failed",
        )
