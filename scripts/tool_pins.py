import re
import tomllib
from pathlib import Path

NATIVE_MANIFESTS = {
    "backend/go.mod": "backend/go.sum",
    "frontend/package.json": "frontend/pnpm-lock.yaml",
    "website/package.json": "website/pnpm-lock.yaml",
    "e2e/browser/pyproject.toml": "e2e/browser/uv.lock",
    "tools/pnpm/package.json": "tools/pnpm/package-lock.json",
    "tools/go/gopls/go.mod": "tools/go/gopls/go.sum",
    "tools/go/govulncheck/go.mod": "tools/go/govulncheck/go.sum",
}
PLATFORMS = ("linux-x64", "linux-arm64")
DIGEST = r"sha256:[a-f0-9]{64}"


def verify_tool_pins(root: Path) -> None:
    configuration = tomllib.loads((root / "mise.toml").read_text())
    locks = tomllib.loads((root / "mise.lock").read_text())["tools"]
    declarations = list(configuration["tools"].items())
    for task in configuration.get("tasks", {}).values():
        declarations.extend(task.get("tools", {}).items())

    for name, version in declarations:
        if not isinstance(version, str) or not re.fullmatch(r"v?\d+\.\d+\.\d+", version):
            raise ValueError(f"{name}: development tool requires an exact version")

        records = [record for record in locks.get(name, []) if record.get("version") == version]
        if len(records) != 1:
            raise ValueError(f"{name}@{version}: missing native tool lock")

        record = records[0]
        backend_alias = configuration.get("tool_alias", {}).get(name)
        if isinstance(backend_alias, str) and record.get("backend") != backend_alias:
            raise ValueError(f"{name}@{version}: configured alias and locked backend disagree")

        sidecar = record.get("aube") or record.get("uv")
        if sidecar is not None:
            directory = root / sidecar["path"]
            manifest, lock = (
                ("package.json", "aube-lock.yaml")
                if "aube" in record
                else ("pyproject.toml", "uv.lock")
            )
            if not directory.resolve().is_relative_to((root / ".mise/locks").resolve()) or not all(
                (directory / name).is_file() for name in (manifest, lock)
            ):
                raise ValueError(f"{name}@{version}: missing native lock sidecar")
        else:
            for platform in PLATFORMS:
                platform_record = record.get("platforms." + platform, {})
                if not re.fullmatch(DIGEST, platform_record.get("checksum", "")):
                    raise ValueError(f"{name}@{version}: missing {platform} checksum")
                if not isinstance(platform_record.get("url"), str) or not platform_record["url"]:
                    raise ValueError(f"{name}@{version}: missing {platform} download URL")

    for manifest, lock in NATIVE_MANIFESTS.items():
        if not (root / manifest).is_file() or not (root / lock).is_file():
            raise ValueError(f"{manifest}: native manifest and lock are required")
