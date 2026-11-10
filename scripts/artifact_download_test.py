import hashlib
import http.server
import tempfile
import threading
import unittest
from functools import partial
from pathlib import Path
from urllib.error import HTTPError

from artifact_download import download_verified_artifact


class ArtifactHandler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self) -> None:
        if self.path == "/redirect-to-file":
            self.send_response(302)
            self.send_header("Location", Path(self.translate_path("/client.tar")).as_uri())
            self.end_headers()
            return

        super().do_GET()

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

    def test_refuses_unsafe_sources_with_a_matching_hash(self) -> None:
        source = self.root / "client.tar"
        source.write_bytes(b"private file")
        checksum = hashlib.sha256(source.read_bytes()).hexdigest()
        destination = self.root / "download"

        for url in (
            source.as_uri(),
            "ftp://example.com/client.tar",
            "data:text/plain,private%20file",
            "http://example.com/client.tar",
            "https://user:password@example.com/client.tar",
        ):
            with self.subTest(url=url):
                with self.assertRaisesRegex(ValueError, "credential-free HTTPS source"):
                    download_verified_artifact(url, destination, checksum)

                self.assertFalse(destination.exists())

    def test_refuses_redirects_to_a_local_file_with_a_matching_hash(self) -> None:
        source = self.root / "client.tar"
        source.write_bytes(b"private file")
        checksum = hashlib.sha256(source.read_bytes()).hexdigest()
        destination = self.root / "download"
        destination.write_bytes(b"previous download")

        with self.assertRaises(HTTPError) as redirect_failure:
            download_verified_artifact(self.url + "/redirect-to-file", destination, checksum)

        with redirect_failure.exception:
            self.assertRegex(str(redirect_failure.exception), "Redirection to url.*not allowed")

        self.assertFalse(destination.exists())


if __name__ == "__main__":
    unittest.main()
