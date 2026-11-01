from dataclasses import asdict
from functools import cache
import hashlib
import json
import os
from pathlib import Path


EXECUTION_POLICY_VERSION = 1


@cache
def get_review_profile_identity(root, profile):
    hooks_dir = Path(__file__).resolve().parent
    inputs = set(hooks_dir.glob("*-prompt.md")) | set(hooks_dir.glob("*.schema.json"))

    if profile.client == "codex":
        codex_home = Path(os.environ.get("CODEX_HOME", Path.home() / ".codex"))
        inputs.add(codex_home / "config.toml")
        inputs.update(codex_home.glob("*.config.toml"))
        inputs.add(Path("/etc/codex/config.toml"))
        inputs.add(Path("/etc/codex/requirements.toml"))
        for directory in (root, *root.parents):
            inputs.add(directory / ".codex/config.toml")

    digests = {}
    for path in sorted(inputs):
        try:
            digests[str(path)] = hashlib.sha256(path.read_bytes()).hexdigest()
        except FileNotFoundError:
            digests[str(path)] = None
    identity = {"profile": asdict(profile), "inputs": digests, "execution_policy": EXECUTION_POLICY_VERSION}
    return hashlib.sha256(json.dumps(identity, sort_keys=True).encode()).hexdigest()
