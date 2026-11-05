from pathlib import Path
import re
import sys
import tomllib
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parent))

from workflow_jobs import get_jobs  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]
WORKFLOWS = ROOT / ".github/workflows"
CI_RELEASE = (WORKFLOWS / "ci-release.yml").read_text()
TOOLS = tomllib.loads((ROOT / "mise.toml").read_text())["tools"]


class BuildInputsTest(unittest.TestCase):
    def test_trivy_is_pinned_and_not_installed_from_apt(self):
        trivy_version = TOOLS["aqua:aquasecurity/trivy"]

        self.assertRegex(trivy_version, r"^\d+\.\d+\.\d+$")
        self.assertNotEqual(trivy_version, "0.69.4")
        self.assertNotIn("trivy", TOOLS)

        for workflow_path in WORKFLOWS.glob("*.yml"):
            with self.subTest(workflow=workflow_path.name):
                self.assertNotIn("apt-get install -y trivy", workflow_path.read_text())

        for job_name in ("dockerfile-scan", "build-image"):
            with self.subTest(job_name=job_name):
                self.assertRegex(get_jobs(CI_RELEASE)[job_name], r"install_args:.*aqua:aquasecurity/trivy")

    def test_semver_version_is_exact(self):
        self.assertRegex(TOOLS["npm:semver"], r"^\d+\.\d+\.\d+$")
        self.assertRegex(get_jobs(CI_RELEASE)["determine-version"], r"install_args:.*npm:semver")

    def test_base_images_use_digests(self):
        base_images = re.findall(r"^FROM (?:--platform=\S+ )?(\S+)", (ROOT / "Dockerfile").read_text(), re.M)

        self.assertEqual(len(base_images), 4)
        for base_image in base_images:
            with self.subTest(base_image=base_image):
                self.assertRegex(base_image, r"^[a-z0-9./-]+:[A-Za-z0-9._-]+@sha256:[a-f0-9]{64}$")

    def test_frontend_base_image_matches_the_pinned_node(self):
        dockerfile = (ROOT / "Dockerfile").read_text()

        self.assertIn(f"node:{TOOLS['node']}-alpine@sha256:", dockerfile)


if __name__ == "__main__":
    unittest.main()
