from pathlib import Path

import pytest

from docker_wrapper import install_docker_wrapper_and_get_path
from public_command import PublicRun


class TestCiWithoutImage:
    @pytest.mark.parametrize("command", [("mise", "run", "e2e:browser"), ("mise", "run", "e2e:lifecycle")])
    def test_refuses_names_the_setting_and_builds_nothing_when_ci_names_no_image(
        self, command: tuple[str, ...], monkeypatch: pytest.MonkeyPatch, tmp_path: Path
    ):
        monkeypatch.delenv("DATABASUS_IMAGE", raising=False)
        build_marker = tmp_path / "docker-build-was-called"
        path_recording_builds = install_docker_wrapper_and_get_path(tmp_path, f'*" build "*) touch "{build_marker}";;')

        finished_run = PublicRun(command=command, environment={"CI": "1", "PATH": path_recording_builds}).finish()

        assert finished_run.exit_code != 0
        assert "DATABASUS_IMAGE" in finished_run.output
        assert not build_marker.exists()
