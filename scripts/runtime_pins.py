import json
import re
import tomllib
from pathlib import Path

from read_mise_bootstrap import get_bootstrap

DIGEST = r"sha256:[a-f0-9]{64}"


def verify_runtime_pins(root: Path) -> None:
    configuration = tomllib.loads((root / "mise.toml").read_text())
    tools = configuration["tools"]
    bootstrap_version, _ = get_bootstrap(root)
    if configuration["min_version"] != bootstrap_version:
        raise ValueError("mise bootstrap and minimum version disagree")

    compiler = re.search(r"(?m)^go ([\d.]+)$", (root / "backend/go.mod").read_text())
    if compiler is None or compiler[1] != tools["go"]:
        raise ValueError("backend compiler and mise Go version disagree")

    dockerfile = (root / "Dockerfile").read_text()
    for image, tool in (("golang", "go"), ("node", "node")):
        versions = re.findall(rf"\b{image}:([\d.]+)(?:-[\w.-]+)?@{DIGEST}", dockerfile)
        if not versions or any(version != tools[tool] for version in versions):
            raise ValueError(f"{tool}: Docker build and mise runtime versions disagree")

    pnpm_version = tools["pnpm"]
    frontend = json.loads((root / "frontend/package.json").read_text())
    website = json.loads((root / "website/package.json").read_text())
    bootstrap = json.loads((root / "tools/pnpm/package.json").read_text())
    browser = json.loads((root / "frontend/browser/environment.json").read_text())
    if (
        frontend.get("packageManager") != "pnpm@" + pnpm_version
        or website.get("packageManager") != "pnpm@" + pnpm_version
        or bootstrap.get("dependencies") != {"pnpm": pnpm_version}
    ):
        raise ValueError("pnpm bootstrap, frontend, website and mise versions disagree")

    if browser.get("nodeVersion") != tools["node"] or browser.get("pnpmVersion") != pnpm_version:
        raise ValueError("browser and mise Node/pnpm versions disagree")

    python = tomllib.loads((root / "e2e/browser/pyproject.toml").read_text())
    version = browser["playwrightVersion"]
    if (
        frontend["devDependencies"].get("@playwright/test") != version
        or f"playwright=={version}" not in python["project"]["dependencies"]
    ):
        raise ValueError("browser Python and JavaScript Playwright versions disagree")

    image = browser["playwrightImage"]
    if not re.fullmatch(
        r"mcr.microsoft.com/playwright:v" + re.escape(version) + r"-[\w.-]+@" + DIGEST, image
    ):
        raise ValueError("Playwright image and package versions disagree")

    for path in ("e2e/browser/compose.yaml", "frontend/browser/Dockerfile"):
        if image not in (root / path).read_text():
            raise ValueError(f"{path}: Playwright image declarations disagree")

    node_locks = tomllib.loads((root / "mise.lock").read_text())["tools"]["node"]
    node_lock = next(record for record in node_locks if record["version"] == tools["node"])
    for architecture in ("x64", "arm64"):
        if (
            node_lock[f"platforms.linux-{architecture}"]["checksum"]
            != "sha256:" + browser["nodeArchiveSha256"][architecture]
        ):
            raise ValueError(f"browser Node archive and native {architecture} checksum disagree")

    tool_compiler = configuration["tasks"]["tools:go"]["tools"]["go"]
    for manifest in ("tools/go/gopls/go.mod", "tools/go/govulncheck/go.mod"):
        if f"\ngo {tool_compiler}\n" not in (root / manifest).read_text():
            raise ValueError(f"{manifest}: Go tool compiler versions disagree")
