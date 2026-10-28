import os
from pathlib import Path
import signal
import selectors
import subprocess
import sys
import tempfile
import unittest


ROOT = Path(__file__).resolve().parent.parent


class BackendShardRunnerTest(unittest.TestCase):
    def setUp(self):
        temporary_directory = tempfile.TemporaryDirectory(prefix="backend-shard-runner-test-")
        self.addCleanup(temporary_directory.cleanup)
        self.directory = Path(temporary_directory.name)
        executable_directory = self.directory / "bin"
        executable_directory.mkdir()
        self.environment = {
            **os.environ,
            "PATH": f"{executable_directory}:{os.environ['PATH']}",
            "SHARD_TEST_DIRECTORY": str(self.directory),
            "TMPDIR": str(self.directory),
            "LIBEDIT_SHIM_DIR": str(self.directory / "libedit"),
        }
        self.write_executable(executable_directory / "mise", """#!/usr/bin/env bash
case "$*" in
    *backend:test-group-packages*) echo './fake' ;;
    *backend:test-group-tests*) echo "${@: -1}" ;;
    *) exit 2 ;;
esac
""")
        self.write_executable(executable_directory / "go", f"#!{sys.executable}\n" + """
import os
from pathlib import Path
import sys
directory = Path(os.environ['SHARD_TEST_DIRECTORY'])
assert '-failfast' in sys.argv
pattern = sys.argv[sys.argv.index('-run') + 1]
(directory / 'test-started').touch()
if (directory / 'interrupt').exists():
    with (directory / 'started').open('w') as started:
        started.write(str(os.getpid()))
if pattern.endswith('1-of-2'):
    with (directory / 'rendezvous').open('wb') as rendezvous:
        rendezvous.write(b'x')
    sys.exit(1)
with (directory / 'rendezvous').open('rb') as rendezvous:
    assert rendezvous.read() == b'x'
(directory / 'other-shard-completed').touch()
""")
        os.mkfifo(self.directory / "rendezvous")

    def write_executable(self, path, content):
        path.write_text(content)
        path.chmod(0o755)

    def run_shards(self, shards):
        return subprocess.run(
            ["bash", str(ROOT / "mise-tasks/backend/test-group")],
            cwd=ROOT / "backend",
            env={**self.environment, "usage_shards": " ".join(shards)},
            capture_output=True,
            text=True,
            timeout=10,
        )

    def test_runs_shards_concurrently_and_waits_for_every_shard_after_a_failure(self):
        completed = self.run_shards(["physical-api-1-of-2", "physical-api-2-of-2"])

        self.assertEqual(completed.returncode, 1, completed.stderr)
        self.assertTrue((self.directory / "other-shard-completed").exists())
        self.assertIn("running physical-api-1-of-2", completed.stdout)
        self.assertIn("running physical-api-2-of-2", completed.stdout)
        self.assertEqual(list(self.directory.glob("databasus-test-shards.*")), [])

    def test_rejects_invalid_and_duplicate_selections_before_starting_tests(self):
        for shards in (
            ["physical-api-1-of-2", "other"],
            ["unknown-1-of-3"],
            ["physical"],
            ["physical-api-1-of-2", "physical-api-1-of-2"],
            ["physical-api-3-of-2"],
        ):
            with self.subTest(shards=shards):
                completed = self.run_shards(shards)

                self.assertNotEqual(completed.returncode, 0)
                self.assertFalse((self.directory / "test-started").exists())

    def test_stops_its_child_and_removes_temporary_logs_after_interruption(self):
        (self.directory / "interrupt").touch()
        os.mkfifo(self.directory / "started")
        process = subprocess.Popen(
            ["bash", str(ROOT / "mise-tasks/backend/test-group")],
            cwd=ROOT / "backend",
            env={**self.environment, "usage_shards": "physical-api-1-of-2"},
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )
        self.addCleanup(lambda: process.kill() if process.poll() is None else None)
        started_descriptor = os.open(self.directory / "started", os.O_RDONLY | os.O_NONBLOCK)
        self.addCleanup(os.close, started_descriptor)
        with selectors.DefaultSelector() as readiness:
            readiness.register(started_descriptor, selectors.EVENT_READ)
            self.assertTrue(readiness.select(timeout=10), "the child did not start")
            child_pid = int(os.read(started_descriptor, 100))

        process.send_signal(signal.SIGTERM)
        process.communicate(timeout=10)

        self.assertEqual(process.returncode, 143)
        with self.assertRaises(ProcessLookupError):
            os.kill(child_pid, 0)
        self.assertEqual(list(self.directory.glob("databasus-test-shards.*")), [])
