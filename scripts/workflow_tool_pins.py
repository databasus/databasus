import re
import tomllib
from pathlib import Path


def verify_workflow_tool_pins(root: Path) -> None:
    configuration = tomllib.loads((root / "mise.toml").read_text())
    tools = configuration["tools"]
    if "pnpm" not in tools or configuration.get("tool_alias", {}).get("pnpm") != "github:pnpm/pnpm":
        raise ValueError(
            "pnpm requires a short tool declaration and the github:pnpm/pnpm backend alias"
        )

    declarations = list(tools.items())
    for task in configuration.get("tasks", {}).values():
        declarations.extend(task.get("tools", {}).items())

    locks = tomllib.loads((root / "mise.lock").read_text())["tools"]
    for workflow in sorted((root / ".github/workflows").glob("*.y*ml")):
        workflow_lines = workflow.read_text().splitlines()
        for line_number, line in enumerate(workflow_lines, 1):
            arguments = re.match(
                r"\s*(?:install_args|'install_args'|\"install_args\")\s*:\s*(.*)$", line
            )
            if arguments is None:
                continue

            location = f"{workflow.name}:{line_number}"
            argument_value = arguments[1]
            following_line = next(
                (
                    candidate
                    for candidate in workflow_lines[line_number:]
                    if candidate.strip() and not candidate.lstrip().startswith("#")
                ),
                "",
            )
            if len(following_line) - len(following_line.lstrip()) > len(line) - len(line.lstrip()):
                argument_value += "\n" + following_line

            try:
                workflow_tools = get_workflow_tools(argument_value)
            except ValueError as error:
                raise ValueError(f"{location}: {error}") from error

            for tool_spec in workflow_tools:
                if tool_spec in tools:
                    name, version = tool_spec, tools[tool_spec]
                else:
                    name, separator, version = tool_spec.rpartition("@")
                    if (
                        not separator
                        or not re.fullmatch(r"v?\d+\.\d+\.\d+", version)
                        or (name, version) not in declarations
                    ):
                        raise ValueError(
                            f"{location}: {tool_spec}: missing exact mise tool declaration"
                        )

                if not any(record.get("version") == version for record in locks.get(name, [])):
                    raise ValueError(f"{location}: {tool_spec}: missing native tool lock")


def get_workflow_tools(arguments: str) -> list[str]:
    scalar = re.fullmatch(
        r"(?:'([^'\n]*)'|\"([^\"\\\n]*)\"|([^'\"#\n]+?))[ \t]*(?:#.*)?", arguments
    )
    if scalar is not None:
        tools = next(value for value in scalar.groups() if value is not None).split()
        if tools and all(
            re.fullmatch(r"[A-Za-z0-9_:@./+-]+", tool) and not tool.startswith("-")
            for tool in tools
        ):
            return tools

    raise ValueError("unsupported install_args; use a static single-line tool list")
