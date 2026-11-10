import subprocess
from pathlib import Path

from action_pins import verify_action_pins
from client_bundles import verify_client_bundles
from image_pins import verify_image_pins
from runtime_pins import verify_runtime_pins
from tool_pins import verify_tool_pins
from verification_extension_pins import verify_verification_extension_pins
from workflow_tool_pins import verify_workflow_tool_pins


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    failures = 0

    for label, check in (
        ("native lock presence and tool pins", verify_tool_pins),
        ("coordinated runtime versions", verify_runtime_pins),
        ("workflow tool declarations", verify_workflow_tool_pins),
        ("workflow action commits", verify_action_pins),
        ("known container image digests", verify_image_pins),
        ("verification extension versions and checksums", verify_verification_extension_pins),
        ("local vendored client hashes and permissions", verify_client_bundles),
    ):
        try:
            check(root)
            print(f"PASS: {label}")
        except (
            OSError,
            ValueError,
            LookupError,
            TypeError,
            AttributeError,
            StopIteration,
            subprocess.SubprocessError,
        ) as error:
            print(f"FAIL: {label}: {error}")
            failures += 1

    print(
        "Checked known local declarations and client files; release age and upstream review remain the updater's responsibility. Native installations verify package graphs and downloaded content."
    )

    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())
