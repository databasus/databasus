import hashlib
import io
import json
import os
import subprocess
import tarfile
from pathlib import Path

import pytest
from recovery_image import get_archived_image_id

SUITE_DIRECTORY = Path(__file__).parent.parent


@pytest.mark.parametrize("config_path", ["config.json", "blobs/sha256/image-config"])
def test_uses_the_config_digest_shared_by_legacy_and_containerd_engines(
    tmp_path: Path, config_path: str
) -> None:
    config = b'{"architecture":"arm64","os":"linux","rootfs":{"type":"layers","diff_ids":[]}}'
    manifest = json.dumps([{"Config": config_path, "RepoTags": None, "Layers": []}]).encode()
    archive_path = tmp_path / "postgres.tar"
    with tarfile.open(archive_path, "w") as archive:
        for filename, content in [("manifest.json", manifest), (config_path, config)]:
            member = tarfile.TarInfo(filename)
            member.size = len(content)
            archive.addfile(member, io.BytesIO(content))

    assert get_archived_image_id(archive_path) == f"sha256:{hashlib.sha256(config).hexdigest()}"


@pytest.mark.parametrize("manifest", [[], [{}, {}], [{"Config": 42}]])
def test_rejects_an_archive_without_one_unambiguous_image_config(
    tmp_path: Path, manifest: object
) -> None:
    archive_path = tmp_path / "postgres.tar"
    content = json.dumps(manifest).encode()
    with tarfile.open(archive_path, "w") as archive:
        member = tarfile.TarInfo("manifest.json")
        member.size = len(content)
        archive.addfile(member, io.BytesIO(content))

    with pytest.raises(ValueError, match="exactly one image config"):
        get_archived_image_id(archive_path)


@pytest.mark.parametrize("should_load_fail", [False, True])
def test_loads_the_supplied_image_and_never_contacts_a_registry(
    tmp_path: Path, should_load_fail: bool
) -> None:
    commands = tmp_path / "commands"
    readiness = tmp_path / "ready"
    binary_directory = tmp_path / "bin"
    binary_directory.mkdir()
    for name in ("apt-get", "rm", "chown", "chgrp", "chmod", "sleep"):
        executable = binary_directory / name
        executable.write_text("#!/bin/sh\nexit 0\n")
        executable.chmod(0o755)
    docker = binary_directory / "docker"
    docker.write_text(
        '#!/bin/sh\nprintf "%s\\n" "$*" >> "$RECOVERY_COMMANDS"\n'
        'case "$*" in\n'
        '  "info" | "image inspect sha256:fixture") exit 0;;\n'
        f'  "image load --input /postgres.tar") exit {1 if should_load_fail else 0};;\n'
        '  *) echo "unexpected registry access or image" >&2; exit 1;;\n'
        "esac\n"
    )
    docker.chmod(0o755)
    script = tmp_path / "prepare-recovery.sh"
    script.write_text(
        (SUITE_DIRECTORY / "prepare_recovery.sh")
        .read_text()
        .replace("/tmp/recovery-ready", str(readiness))
    )

    completed = subprocess.run(
        ["sh", str(script)],
        env={
            **os.environ,
            "PATH": f"{binary_directory}:{os.environ['PATH']}",
            "E2E_POSTGRES_IMAGE": "sha256:fixture",
            "RECOVERY_COMMANDS": str(commands),
        },
        capture_output=True,
        text=True,
    )

    assert (completed.returncode != 0) == should_load_fail, completed.stderr
    assert readiness.exists() != should_load_fail
    invoked_commands = commands.read_text().splitlines()
    assert invoked_commands[:2] == ["info", "image load --input /postgres.tar"]
    assert ("image inspect sha256:fixture" in invoked_commands) != should_load_fail
