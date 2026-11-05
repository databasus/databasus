import os
from dataclasses import dataclass
from pathlib import Path

CLIENT_ENV = "AGENT_REVIEW_CLIENT"
CLIENT_SELECTION_HINT = (
    "Select --client codex or --client claude, or set AGENT_REVIEW_CLIENT in an agent session."
)


@dataclass(frozen=True)
class ReviewProfile:
    client: str
    model: str
    quick_effort: str
    full_effort: str


REVIEW_PROFILES = {
    "claude": ReviewProfile("claude", "opus", "low", "medium"),
    "codex": ReviewProfile("codex", "configured-default", "medium", "high"),
}


def get_review_profile(explicit_client: str | None = None) -> ReviewProfile:
    client = explicit_client if explicit_client is not None else os.environ.get(CLIENT_ENV)
    if client is None or client not in REVIEW_PROFILES:
        raise ValueError(f"Invalid or missing review client. {CLIENT_SELECTION_HINT}")
    return REVIEW_PROFILES[client]


@dataclass(frozen=True)
class ReviewExecution:
    root: Path
    prompt: str
    client: str
    model: str
    effort: str
    schema_path: Path
    timeout_seconds: float
    environment: dict[str, str]
