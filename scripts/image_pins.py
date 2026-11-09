import json
import re
from pathlib import Path

PLATFORMS = ("linux-x64", "linux-arm64")
DIGEST = r"sha256:[a-f0-9]{64}"


def verify_image_pins(root: Path) -> None:
    document = json.loads((root / "backend/internal/lib/testimages/images.json").read_text())
    images = document["images"]
    verification = (root / "backend/cmd/verification-agent/images/postgres/Dockerfile").read_text()
    for major in range(12, 19):
        if (
            f"FROM {images[f'postgres:{major}']['reference']} AS postgres-{major}"
            not in verification
        ):
            raise ValueError(f"verification PostgreSQL {major}: base and catalog digests disagree")

    for name, record in images.items():
        if not re.fullmatch(r"[^\s@]+@" + DIGEST, record["reference"]):
            raise ValueError(f"{name}: image requires a digest pin")

        platforms = set(PLATFORMS) if name not in {"mysql:5.7", "mariadb:5.5"} else {"linux-x64"}
        if set(record["platforms"]) != platforms or any(
            not re.fullmatch(DIGEST, digest) for digest in record["platforms"].values()
        ):
            raise ValueError(f"{name}: image platform digests are incomplete or malformed")

    dockerfiles = (
        "Dockerfile",
        "frontend/browser/Dockerfile",
        "backend/cmd/verification-agent/images/postgres/Dockerfile",
        "backend/internal/lib/testcontainers/testdata/ftp/Dockerfile",
        "backend/internal/lib/testcontainers/testdata/sftp/Dockerfile",
        "backend/internal/lib/testcontainers/testdata/ssh_bastion/Dockerfile",
        "backend/internal/modules/backups/e2e/postgresql/testdata/mtls/Dockerfile",
        "backend/internal/modules/backups/e2e/postgresql/testdata/ssl/Dockerfile",
        "backend/internal/modules/backups/physicalapitests/testdata/mtls/Dockerfile",
        "e2e/docker-storage/nfs/Dockerfile",
    )
    for path in dockerfiles:
        content = (root / path).read_text()
        references = re.findall(r"^[ \t]*FROM\s+(?:--platform=\S+\s+)?(\S+)", content, re.M | re.I)
        references.extend(re.findall(r"^# syntax=(\S+)", content, re.M))
        stages = {
            stage.lower()
            for stage in re.findall(r"^[ \t]*FROM\s+.*?\s+AS\s+(\S+)", content, re.M | re.I)
        }
        for reference in references:
            if reference.lower() in stages:
                continue

            if (
                path == "backend/cmd/verification-agent/images/postgres/Dockerfile"
                and reference == "postgres-${PG_MAJOR}"
            ):
                continue

            if not re.fullmatch(r"[^\s@]+@" + DIGEST, reference):
                raise ValueError(f"{path}: third-party image requires a digest pin")

    for path in ("docker-compose.yml", "e2e/browser/compose.yaml"):
        for reference in re.findall(
            r'^\s*image:\s*[\'"]?([^\s\'"#]+)', (root / path).read_text(), re.M
        ):
            if reference.startswith(("databasus-", "${")):
                continue

            if not re.fullmatch(r"[^\s@]+@" + DIGEST, reference):
                raise ValueError(f"{path}: third-party image requires a digest pin")
