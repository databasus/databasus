from pathlib import Path

from docker_wrapper import install_docker_wrapper_and_get_path
from leftover_check import is_image_present
from probe_run import INTERRUPT_READY, assert_failed_and_released, start_probe_run
from public_command import get_running_image


class TestNamedImage:
    def test_tests_a_named_image_without_building_and_leaves_it_in_place(self, probes_image: str, tmp_path: Path):
        build_marker = tmp_path / "docker-build-was-called"
        path_recording_builds = install_docker_wrapper_and_get_path(tmp_path, f'*" build "*) touch "{build_marker}";;')
        run = start_probe_run(probes_image, "interrupt", PATH=path_recording_builds)
        run.wait_for_output(INTERRUPT_READY)

        assert get_running_image(run.get_project_name(), "databasus") == probes_image

        run.terminate_supervisor()
        output = assert_failed_and_released(run, "E2E run interrupted")

        assert f"E2E image under test: {probes_image}" in output
        assert not build_marker.exists()
        assert is_image_present(probes_image)
