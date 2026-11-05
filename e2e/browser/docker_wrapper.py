import os
import shutil
import stat
from pathlib import Path


def install_docker_wrapper_and_get_path(directory: Path, case_branch: str) -> str:
    real_docker = shutil.which("docker")
    wrapper = directory / "docker"
    wrapper.write_text(f'#!/bin/sh\ncase " $* " in {case_branch} esac\nexec {real_docker} "$@"\n')
    wrapper.chmod(wrapper.stat().st_mode | stat.S_IEXEC)

    return f"{directory}:{os.environ['PATH']}"
