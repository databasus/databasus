import os
import subprocess
import sys

import pytest

import supervisor_file
from supervisor_file import SupervisorRecord, get_process_start_time, is_supervisor_alive


class TestIsSupervisorAlive:
    def test_is_true_for_a_running_process_with_its_start_time(self):
        record = SupervisorRecord("run", os.getpid(), get_process_start_time(os.getpid()), None)

        assert is_supervisor_alive(record)

    def test_is_false_when_another_process_reuses_the_pid(self):
        record = SupervisorRecord("run", os.getpid(), "another-start-time", None)

        assert not is_supervisor_alive(record)

    def test_is_false_when_the_process_exited_but_its_parent_has_not_reaped_it(self):
        child = subprocess.Popen([sys.executable, "-c", "import signal; signal.pause()"])

        try:
            record = SupervisorRecord("exited", child.pid, get_process_start_time(child.pid), None)
            child.terminate()
            os.waitid(os.P_PID, child.pid, os.WEXITED | os.WNOWAIT)

            assert not is_supervisor_alive(record)
        finally:
            child.terminate()
            child.wait()


class TestReadSupervisorRecords:
    def test_treats_a_damaged_file_as_a_dead_run_instead_of_failing(self, tmp_path, monkeypatch):
        monkeypatch.setattr(supervisor_file, "SUPERVISORS_DIRECTORY", tmp_path)
        damaged_run_directory = tmp_path / "damaged"
        damaged_run_directory.mkdir()
        (damaged_run_directory / supervisor_file.SUPERVISOR_FILE_NAME).write_text('{"run_id": "dam')

        [record] = supervisor_file.read_supervisor_records()

        assert record.run_id == "damaged"
        assert not is_supervisor_alive(record)

    def test_leaves_no_temporary_file_after_writing_a_record(self, tmp_path, monkeypatch):
        monkeypatch.setattr(supervisor_file, "SUPERVISORS_DIRECTORY", tmp_path)

        supervisor_file.write_supervisor_record(SupervisorRecord("written", 1, "1", None))

        assert [path.name for path in (tmp_path / "written").iterdir()] == [supervisor_file.SUPERVISOR_FILE_NAME]
        assert supervisor_file.read_supervisor_records() == [SupervisorRecord("written", 1, "1", None)]

    def test_keeps_the_previous_record_whole_when_a_write_fails_before_its_rename(self, tmp_path, monkeypatch):
        monkeypatch.setattr(supervisor_file, "SUPERVISORS_DIRECTORY", tmp_path)
        first_record = SupervisorRecord("written", 1, "1", None)
        supervisor_file.write_supervisor_record(first_record)

        def refuse_rename(_source, _destination):
            raise OSError("rename refused")

        monkeypatch.setattr(supervisor_file.os, "replace", refuse_rename)

        with pytest.raises(OSError, match="rename refused"):
            supervisor_file.write_supervisor_record(SupervisorRecord("written", 2, "2", "image"))

        assert supervisor_file.read_supervisor_records() == [first_record]
