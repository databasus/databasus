from pathlib import PurePosixPath


def is_documentation_path(path: str) -> bool:
    document = PurePosixPath(path)
    if document.suffix != ".md" or path.startswith("openspec/"):
        return False
    if "generated" in document.parts or path.startswith(".agents/skills/openspec-"):
        return False

    return (
        path.startswith(("docs/", ".agents/skills/"))
        or document.name == "AGENTS.md"
        or document.name == "README.md"
        or document.name.startswith("README.")
        or (
            document.parent == PurePosixPath(".agent-hooks")
            and document.name.endswith("-prompt.md")
        )
    )
