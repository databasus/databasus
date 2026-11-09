import hashlib
import json
import re
import subprocess
from dataclasses import dataclass
from pathlib import Path, PurePosixPath
from urllib.parse import urlsplit


@dataclass(frozen=True)
class ClientFile:
    path: str
    member: str
    sha256: str
    mode: int


@dataclass(frozen=True)
class ClientArchive:
    bundle: str
    version: str
    build_family: str
    architecture: str
    url: str
    sha256: str
    files: tuple[ClientFile, ...]


def get_string(record: dict[str, object], field: str) -> str:
    value = record[field]
    if not isinstance(value, str) or not value:
        raise ValueError(f"client archive requires {field}")
    return value


def get_sha256(record: dict[str, object], field: str = "sha256") -> str:
    checksum = get_string(record, field)
    if not re.fullmatch(r"[a-f0-9]{64}", checksum):
        raise ValueError("client record requires an expected SHA256")
    return checksum


def get_client_archives(root: Path) -> tuple[ClientArchive, ...]:
    document = json.loads((root / "tools/dependencies/clients.json").read_text())
    if (
        not isinstance(document, dict)
        or document.get("schema") != 1
        or not isinstance(document.get("archives"), list)
    ):
        raise ValueError("client source records are missing or malformed")

    archives: list[ClientArchive] = []
    for record in document["archives"]:
        if (
            not isinstance(record, dict)
            or not isinstance(record.get("files"), list)
            or not record["files"]
        ):
            raise ValueError("client archive requires recorded files")

        bundle, architecture = get_string(record, "bundle"), get_string(record, "architecture")
        if architecture not in {"x64", "arm"}:
            raise ValueError("client archive has an unsupported architecture")

        engine = bundle.split("-")[0]
        prefix = f"assets/tools/{architecture}/{engine}/"
        if engine != "mongodb":
            prefix += bundle + "/"

        files: list[ClientFile] = []
        for file in record["files"]:
            if (
                not isinstance(file, dict)
                or type(file.get("mode")) is not int
                or file["mode"] not in {0o644, 0o755}
            ):
                raise ValueError("client file requires a recorded executable or library mode")

            path, member = get_string(file, "path"), get_string(file, "member")
            if not path.startswith(prefix) or any(
                PurePosixPath(value).is_absolute() or ".." in PurePosixPath(value).parts
                for value in (path, member)
            ):
                raise ValueError("client file path differs from its bundle")

            files.append(ClientFile(path, member, get_sha256(file), file["mode"]))

        url = get_string(record, "url")
        source = urlsplit(url)
        if (
            (
                source.scheme != "https"
                and not (source.scheme == "http" and source.hostname in {"127.0.0.1", "::1"})
            )
            or not source.hostname
            or source.username
            or source.password
        ):
            raise ValueError("client source requires a credential-free HTTPS URL")

        archives.append(
            ClientArchive(
                bundle,
                get_string(record, "version"),
                get_string(record, "build_family"),
                architecture,
                url,
                get_sha256(record),
                tuple(files),
            )
        )

    return tuple(archives)


def get_file_sha256(path: Path) -> str:
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def verify_client_bundles(root: Path) -> None:
    archives = get_client_archives(root)
    recorded_paths: set[str] = set()
    builds: dict[str, set[tuple[str, str]]] = {}
    architectures: dict[str, set[str]] = {}

    index = subprocess.check_output(
        ["git", "ls-files", "--stage", "-z", "--", "assets/tools"], cwd=root, timeout=30
    )
    indexed_modes = {
        entry.split(b"\t", 1)[1].decode(): entry.split(b" ", 1)[0]
        for entry in index.split(b"\0")
        if entry
    }

    for archive in archives:
        builds.setdefault(archive.bundle, set()).add((archive.version, archive.build_family))
        architectures.setdefault(archive.bundle, set()).add(archive.architecture)

        for file in archive.files:
            if file.path in recorded_paths:
                raise ValueError(f"{file.path}: duplicate client file record")

            recorded_paths.add(file.path)
            path = root / file.path
            if path.is_symlink() or not path.is_file() or get_file_sha256(path) != file.sha256:
                raise ValueError(f"{file.path}: missing client file or SHA256 mismatch")

            if path.stat().st_mode & 0o777 != file.mode:
                raise ValueError(f"{file.path}: client file permissions differ from the record")

            if file.mode == 0o755 and indexed_modes.get(file.path, b"100755") != b"100755":
                raise ValueError(f"{file.path}: Git index requires executable permission")

    for bundle, selected in architectures.items():
        supported = {"x64"} if bundle == "mysql-5.7" else {"x64", "arm"}
        if selected != supported or len(builds[bundle]) != 1:
            raise ValueError(f"{bundle}: partial architecture refresh or mixed client builds")

    shipped_paths = {
        str(path.relative_to(root))
        for architecture in ("x64", "arm")
        for path in (root / "assets/tools" / architecture).rglob("*")
        if path.is_file() or path.is_symlink()
    }
    if shipped_paths != recorded_paths:
        raise ValueError(
            "client file inventory differs: " + ", ".join(sorted(shipped_paths ^ recorded_paths))
        )
