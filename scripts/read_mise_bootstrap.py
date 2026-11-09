import json
import os
import re
from pathlib import Path


def get_bootstrap(root: Path) -> tuple[str, dict[str, str]]:
    document = json.loads((root / "tools/dependencies/mise-bootstrap.json").read_text())
    version, platforms = document["version"], document["platforms"]
    if not isinstance(version, str) or not re.fullmatch(r"\d{4}\.\d+\.\d+", version):
        raise ValueError("mise requires an exact bootstrap version")

    if not isinstance(platforms, dict) or set(platforms) != {"linux-x64", "linux-arm64"}:
        raise ValueError("mise bootstrap requires both Linux architectures")

    checksums: dict[str, str] = {}
    for platform, checksum in platforms.items():
        if not isinstance(checksum, str) or not re.fullmatch(r"[a-f0-9]{64}", checksum):
            raise ValueError("mise bootstrap requires expected SHA256 checksums")

        checksums[platform] = checksum

    return version, checksums


def main() -> None:
    architecture = {"X64": "linux-x64", "ARM64": "linux-arm64"}[os.environ["RUNNER_ARCH"]]
    version, checksums = get_bootstrap(Path(__file__).resolve().parent.parent)

    with Path(os.environ["GITHUB_OUTPUT"]).open("a") as stream:
        stream.write(f"version={version}\nsha256={checksums[architecture]}\n")


if __name__ == "__main__":
    main()
