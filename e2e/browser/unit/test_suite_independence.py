import re
from pathlib import Path

SUITE_DIRECTORY = Path(__file__).parent.parent
OWN_FILE = Path(__file__)
_APPLICATION_TREE_IMPORT = re.compile(r"^\s*(from|import)\s+(backend|frontend|databasus_backend)\b", re.MULTILINE)
_APPLICATION_TREE_PATH = re.compile(r"(backend|frontend)/")
_ROUTE_INTERCEPTION = re.compile(r"\.route\(\s*([^,]+),")
_ALLOWED_ROUTE_TARGETS = {"GITHUB_REPOSITORY_ADDRESS", 'f"{GITHUB_REPOSITORY_ADDRESS}/releases/latest"'}


def _get_suite_files() -> list[Path]:
    return [
        path for path in SUITE_DIRECTORY.rglob("*.py")
        if ".venv" not in path.parts and path != OWN_FILE
    ]


class TestSuiteIndependence:
    def test_imports_nothing_from_the_application_trees(self):
        importing_files = [
            path.name for path in _get_suite_files() if _APPLICATION_TREE_IMPORT.search(path.read_text())
        ]

        assert importing_files == []

    def test_scenarios_hold_no_path_into_the_application_trees(self):
        scenario_files = (SUITE_DIRECTORY / "scenarios").rglob("*.py")

        files_with_paths = [path.name for path in scenario_files if _APPLICATION_TREE_PATH.search(path.read_text())]

        assert files_with_paths == []

    def test_scenarios_intercept_only_the_allowed_external_services(self):
        scenario_files = (SUITE_DIRECTORY / "scenarios").rglob("*.py")

        route_targets = {
            target.strip() for path in scenario_files for target in _ROUTE_INTERCEPTION.findall(path.read_text())
        }

        assert route_targets <= _ALLOWED_ROUTE_TARGETS
