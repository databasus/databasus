import hashlib
import json
import tarfile
from pathlib import Path


def get_archived_image_id(archive_path: Path) -> str:
    with tarfile.open(archive_path) as archive:
        manifest_stream = archive.extractfile("manifest.json")
        if manifest_stream is None:
            raise ValueError("Recovery archive has no image manifest")
        with manifest_stream:
            manifest: object = json.load(manifest_stream)
        if (
            not isinstance(manifest, list)
            or len(manifest) != 1
            or not isinstance(manifest[0], dict)
            or not isinstance(manifest[0].get("Config"), str)
        ):
            raise ValueError("Recovery archive must contain exactly one image config")

        config_stream = archive.extractfile(manifest[0]["Config"])
        if config_stream is None:
            raise ValueError("Recovery archive has no image config")
        with config_stream:
            config_digest = hashlib.sha256(config_stream.read()).hexdigest()

    return f"sha256:{config_digest}"
