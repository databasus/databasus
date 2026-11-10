import hashlib
import http.server
import tempfile
import threading
import unittest
from functools import partial
from pathlib import Path

from artifact_download import download_verified_artifact


class ArtifactHandler(http.server.SimpleHTTPRequestHandler):
    def log_message(self, format: str, *args: object) -> None:
        pass


class ArtifactDownloadTest(unittest.TestCase):
    def setUp(self) -> None:
        workspace = tempfile.TemporaryDirectory(prefix="client-download-")
        self.addCleanup(workspace.cleanup)
        self.root = Path(workspace.name)
        self.server = http.server.ThreadingHTTPServer(
            ("127.0.0.1", 0), partial(ArtifactHandler, directory=str(self.root))
        )
        self.thread = threading.Thread(target=self.server.serve_forever)
        self.thread.start()
        self.addCleanup(self.stop_server)
        self.url = f"http://127.0.0.1:{self.server.server_port}"

    def stop_server(self) -> None:
        self.server.shutdown()
        self.thread.join()
        self.server.server_close()

    def test_downloads_matching_content_and_removes_a_replaced_archive(self) -> None:
        source = self.root / "client.tar"
        source.write_bytes(b"reviewed client archive")
        checksum = hashlib.sha256(source.read_bytes()).hexdigest()
        destination = self.root / "download"

        download_verified_artifact(self.url + "/client.tar", destination, checksum)

        self.assertEqual(destination.read_bytes(), source.read_bytes())

        source.write_bytes(b"replaced client archive")

        with self.assertRaisesRegex(ValueError, "SHA256 mismatch"):
            download_verified_artifact(self.url + "/client.tar", destination, checksum)

        self.assertFalse(destination.exists())

    def test_refuses_a_download_without_an_expected_hash(self) -> None:
        destination = self.root / "download"

        with self.assertRaisesRegex(ValueError, "expected SHA256"):
            download_verified_artifact(self.url + "/client.tar", destination, "")

        self.assertFalse(destination.exists())


if __name__ == "__main__":
    unittest.main()
