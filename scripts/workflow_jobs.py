"""Reads the jobs of a workflow file, which the standard library cannot parse as YAML."""

import re


def get_jobs(workflow):
    jobs_section = workflow[workflow.index("\njobs:\n") :]
    return {
        match.group(1): match.group(2)
        for match in re.finditer(r"^  ([a-zA-Z0-9_-]+):\n(.*?)(?=^  [a-zA-Z0-9_-]+:\n|\Z)", jobs_section, re.M | re.S)
    }
