import os
import subprocess
import sys
import tarfile
import tempfile
from dataclasses import dataclass
from pathlib import Path

from artifact_download import download_verified_artifact
from client_bundles import get_client_archives, get_file_sha256


@dataclass(frozen=True)
class MariaDBRefreshSelection:
    bundle: str
    architecture: str


def refresh_mariadb_bundle(root: Path, selection: MariaDBRefreshSelection) -> None:
    if selection.bundle not in {"10.6", "12.3", "13.0"} or selection.architecture not in {
        "amd64",
        "arm64",
    }:
        raise ValueError("usage: refresh-mariadb-bundle.sh <10.6|12.3|13.0> <amd64|arm64>")

    selected = [
        record
        for record in get_client_archives(root)
        if record.bundle == "mariadb-" + selection.bundle
        and record.architecture == {"amd64": "x64", "arm64": "arm"}[selection.architecture]
    ]
    if (
        len(selected) != 2
        or len({(record.version, record.build_family) for record in selected}) != 1
    ):
        raise ValueError("MariaDB refresh requires both sibling packages from one build")

    with tempfile.TemporaryDirectory(prefix="mariadb-refresh-") as workspace:
        staged: list[tuple[Path, Path, int]] = []
        for index, record in enumerate(selected):
            package = Path(workspace) / f"{index}.deb"
            download_verified_artifact(record.url, package, record.sha256)

            extracted = Path(workspace) / str(index)
            extracted.mkdir()
            subprocess.run(["ar", "x", str(package)], cwd=extracted, check=True)
            payloads = list(extracted.glob("data.tar.*"))
            if len(payloads) != 1:
                raise ValueError("MariaDB package requires one native data archive")

            # The data filter rejects paths and links outside the extraction directory.
            # nosemgrep: trailofbits.python.tarfile-extractall-traversal.tarfile-extractall-traversal
            with tarfile.open(payloads[0]) as payload:
                payload.extractall(extracted, filter="data")

            for file in record.files:
                member = extracted / file.member
                if (
                    not member.is_file()
                    or member.is_symlink()
                    or get_file_sha256(member) != file.sha256
                ):
                    raise ValueError("MariaDB archive member differs from its expected file hash")

                staged.append((member, root / file.path, file.mode))

        for member, target, mode in staged:
            target.parent.mkdir(parents=True, exist_ok=True)
            with tempfile.NamedTemporaryFile(dir=target.parent, delete=False) as stream:
                temporary = Path(stream.name)
                try:
                    stream.write(member.read_bytes())
                    stream.close()
                    temporary.chmod(mode)
                    os.replace(temporary, target)
                finally:
                    temporary.unlink(missing_ok=True)


def main() -> int:
    try:
        if len(sys.argv) != 3:
            raise ValueError("usage: refresh-mariadb-bundle.sh <10.6|12.3|13.0> <amd64|arm64>")

        refresh_mariadb_bundle(
            Path(__file__).resolve().parent.parent,
            MariaDBRefreshSelection(bundle=sys.argv[1], architecture=sys.argv[2]),
        )
        return 0
    except ValueError, OSError, subprocess.CalledProcessError:
        print(
            "MariaDB refresh failed; check source records, expected hashes and extraction tools",
            file=sys.stderr,
        )
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
