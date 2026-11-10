import re
from pathlib import Path


def verify_verification_extension_pins(root: Path) -> None:
    verification = (root / "backend/cmd/verification-agent/images/postgres/Dockerfile").read_text()
    extension_version = re.search(r"^ARG VCHORD_VERSION=([\d.]+)$", verification, re.M)
    if extension_version is None:
        raise ValueError("verification extension requires an exact version")

    checksums = (
        (root / "backend/cmd/verification-agent/images/postgres/vectorchord.sha256")
        .read_text()
        .splitlines()
    )
    filenames = {
        f"postgresql-{major}-vchord_{extension_version[1]}-1_{architecture}.deb"
        for major in range(14, 19)
        for architecture in ("amd64", "arm64")
    }
    if (
        len(checksums) != len(filenames)
        or {line.split("  ")[1] for line in checksums if re.fullmatch(r"[a-f0-9]{64}  \S+", line)}
        != filenames
    ):
        raise ValueError("verification extension checksums require all major/architecture records")
