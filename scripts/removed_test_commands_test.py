from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parent.parent


class RemovedTestCommandsTest(unittest.TestCase):
    def test_the_narrow_test_commands_are_gone(self):
        mise_config = (ROOT / "mise.toml").read_text()

        self.assertNotIn("backend:test-package", mise_config)
        self.assertFalse((ROOT / "mise-tasks/backend/test-group").exists())


if __name__ == "__main__":
    unittest.main()
