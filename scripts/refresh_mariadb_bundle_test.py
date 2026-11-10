import hashlib
import http.server
import json
import subprocess
import tarfile
import tempfile
import threading
import unittest
from functools import partial
from pathlib import Path

from refresh_mariadb_bundle import MariaDBRefreshSelection, refresh_mariadb_bundle


class ArtifactHandler(http.server.SimpleHTTPRequestHandler):
    def log_message(self, format: str, *args: object) -> None:
        pass


class RefreshMariaDBBundleTest(unittest.TestCase):
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

    def test_refreshes_matching_packages_and_preserves_clients_after_a_corrupted_download(
        self,
    ) -> None:
        clients = self.create_packages()
        selection = MariaDBRefreshSelection(bundle="10.6", architecture="amd64")

        refresh_mariadb_bundle(self.root, selection)

        self.assertEqual([path.read_bytes() for path in clients], [b"mariadb", b"mariadb-dump"])
        self.assertTrue(all(path.stat().st_mode & 0o111 for path in clients))

        (self.root / "mariadb-dump.deb").write_bytes(b"corrupted package")

        with self.assertRaisesRegex(ValueError, "SHA256 mismatch"):
            refresh_mariadb_bundle(self.root, selection)

        self.assertEqual([path.read_bytes() for path in clients], [b"mariadb", b"mariadb-dump"])

    def test_refuses_archive_paths_and_links_outside_the_extraction_directory(self) -> None:
        escaped_client = self.root / "escaped-client"
        traversal = tarfile.TarInfo("../../" + self.root.name + "/escaped-client")
        symlink = tarfile.TarInfo("usr/bin/escape")
        symlink.type = tarfile.SYMTYPE
        symlink.linkname = str(escaped_client)
        hardlink = tarfile.TarInfo("usr/bin/escape")
        hardlink.type = tarfile.LNKTYPE
        hardlink.linkname = str(escaped_client)
        selection = MariaDBRefreshSelection(bundle="10.6", architecture="amd64")

        for member in (traversal, symlink, hardlink):
            with self.subTest(member=member.name, type=member.type):
                clients = self.create_packages(member)
                for client in clients:
                    client.parent.mkdir(parents=True, exist_ok=True)
                    client.write_bytes(b"existing client")

                with self.assertRaises(tarfile.FilterError):
                    refresh_mariadb_bundle(self.root, selection)

                self.assertFalse(escaped_client.exists())
                self.assertEqual(
                    [client.read_bytes() for client in clients], [b"existing client"] * 2
                )

    def create_packages(self, unsafe_member: tarfile.TarInfo | None = None) -> list[Path]:
        archives: list[dict[str, object]] = []
        clients: list[Path] = []
        for name in ("mariadb", "mariadb-dump"):
            source = self.root / name
            source.mkdir(exist_ok=True)
            binary = source / "client"
            binary.write_bytes(name.encode())
            binary.chmod(0o755)
            with tarfile.open(source / "data.tar.xz", "w:xz") as payload:
                payload.add(binary, arcname="usr/bin/" + name)
                if unsafe_member is not None:
                    payload.addfile(unsafe_member)
            (source / "debian-binary").write_text("2.0\n")
            subprocess.run(
                ["ar", "rc", str(self.root / f"{name}.deb"), "debian-binary", "data.tar.xz"],
                cwd=source,
                check=True,
                capture_output=True,
            )
            target = f"assets/tools/x64/mariadb/mariadb-10.6/bin/{name}"
            clients.append(self.root / target)
            archives.append(
                {
                    "bundle": "mariadb-10.6",
                    "version": "10.6.28",
                    "build_family": "ubuntu2204",
                    "architecture": "x64",
                    "url": self.url + f"/{name}.deb",
                    "sha256": hashlib.sha256((self.root / f"{name}.deb").read_bytes()).hexdigest(),
                    "files": [
                        {
                            "path": target,
                            "member": f"usr/bin/{name}",
                            "sha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
                            "mode": 0o755,
                        }
                    ],
                }
            )
        manifest = self.root / "tools/dependencies/clients.json"
        manifest.parent.mkdir(parents=True, exist_ok=True)
        manifest.write_text(json.dumps({"schema": 1, "archives": archives}))

        return clients


if __name__ == "__main__":
    unittest.main()
