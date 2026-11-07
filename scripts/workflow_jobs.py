"""Reads the jobs of a workflow file, which the standard library cannot parse as YAML."""

import re


def get_steps(job: str) -> list[str]:
    return re.findall(r"^      - name: [^\n]*\n.*?(?=^      - |\Z)", job, re.M | re.S)


def get_jobs(workflow: str) -> dict[str, str]:
    workflow = expand_step_aliases(workflow)
    jobs_section = workflow[workflow.index("\njobs:\n") :]
    return {
        match.group(1): match.group(2)
        for match in re.finditer(
            r"^  ([a-zA-Z0-9_-]+):\n(.*?)(?=^  [a-zA-Z0-9_-]+:\n|\Z)", jobs_section, re.M | re.S
        )
    }


def expand_step_aliases(workflow: str) -> str:
    anchored_steps: dict[str, str] = {}

    def expand_step(step_match: re.Match[str]) -> str:
        anchor_name = step_match["anchor"]
        if anchor_name is not None:
            if anchor_name in anchored_steps:
                raise AssertionError(f"Duplicate step anchor: {anchor_name}")

            step_body = step_match["step"]
            expanded_step = "      - " + step_body[8:]
            anchored_steps[anchor_name] = expanded_step
            return expanded_step

        alias_name = step_match["alias"]
        if alias_name not in anchored_steps:
            raise AssertionError(f"Unknown step anchor: {alias_name}")

        return anchored_steps[alias_name]

    expanded_workflow = re.sub(
        r"^      - &(?P<anchor>[A-Za-z_][\w-]*)\n"
        r"(?P<step>        name: [^\n]*\n(?:(?: {8}[^\n]*)?\n)*)"
        r"|^      - \*(?P<alias>[A-Za-z_][\w-]*)(?:\n|$)",
        expand_step,
        workflow,
        flags=re.M,
    )
    if re.search(r"^      - [&*]", expanded_workflow, re.M):
        raise AssertionError("Unsupported step anchor or alias")

    return expanded_workflow
