import re
from pathlib import Path


def verify_action_pins(root: Path) -> None:
    for workflow in sorted((root / ".github/workflows").glob("*.y*ml")):
        for reference in re.findall(r"^\s*(?:-\s*)?uses:\s*([^\s#]+)", workflow.read_text(), re.M):
            if reference.startswith("./"):
                continue

            if not re.fullmatch(r"[\w./-]+@[a-f0-9]{40}", reference):
                raise ValueError(f"{workflow.name}: third-party action requires a full commit pin")
