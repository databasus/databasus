import leftover_sweep
from leftover_sweep import DeadRun, KnownRuns, read_known_runs, select_dead_runs
from supervisor_file import SupervisorRecord


def _make_record(run_id: str, built_image: str | None = None) -> SupervisorRecord:
    return SupervisorRecord(run_id=run_id, pid=1, process_start_time="1", built_image=built_image)


class TestSelectDeadRuns:
    def test_selects_a_dead_run_with_its_built_image(self):
        dead_runs = select_dead_runs(
            KnownRuns([_make_record("dead", "databasus-e2e-dead:candidate")], ["databasus-e2e-dead"]),
            lambda record: False,
            "own",
        )

        assert dead_runs == [DeadRun("dead", "databasus-e2e-dead:candidate", True)]

    def test_leaves_a_run_whose_supervisor_is_alive(self):
        dead_runs = select_dead_runs(KnownRuns([_make_record("live")], ["databasus-e2e-live"]), lambda record: True, "own")

        assert dead_runs == []

    def test_leaves_its_own_run_alone(self):
        dead_runs = select_dead_runs(KnownRuns([_make_record("own")], ["databasus-e2e-own"]), lambda record: False, "own")

        assert dead_runs == []

    def test_selects_a_project_with_no_supervisor_file_without_any_image(self):
        dead_runs = select_dead_runs(KnownRuns([], ["databasus-e2e-orphan"]), lambda record: True, "own")

        assert dead_runs == [DeadRun("orphan", None, False)]

    def test_never_selects_an_image_the_run_did_not_build(self):
        dead_runs = select_dead_runs(KnownRuns([_make_record("named")], []), lambda record: False, "own")

        assert dead_runs == [DeadRun("named", None, True)]


class TestReadKnownRuns:
    def test_lists_projects_before_reading_records_so_a_starting_run_never_looks_dead(self, monkeypatch):
        reading_order = []
        monkeypatch.setattr(leftover_sweep, "list_project_names", lambda: reading_order.append("projects") or [])
        monkeypatch.setattr(leftover_sweep, "read_supervisor_records", lambda: reading_order.append("records") or [])

        read_known_runs()

        assert reading_order == ["projects", "records"]
