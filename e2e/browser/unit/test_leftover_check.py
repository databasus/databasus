import pytest
from leftover_check import assert_nothing_of_the_project_remains


class TestAssertNothingOfTheProjectRemains:
    def test_fails_and_names_a_surviving_container(self) -> None:
        with pytest.raises(AssertionError, match="container databasus-e2e-run-browser-1"):
            assert_nothing_of_the_project_remains(
                "databasus-e2e-run",
                lambda project_name: ["container databasus-e2e-run-browser-1"],
            )

    def test_passes_when_nothing_survives(self) -> None:
        assert_nothing_of_the_project_remains("databasus-e2e-run", lambda project_name: [])
