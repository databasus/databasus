"""Every network a test creates itself takes a subnet from the test network range that docs/testing.md
names, outside Docker's default address pools, so those pools stay free for the verification agent.
The backend container fixtures follow the same range; this suite imports no backend code."""

import ipaddress
import secrets
from collections.abc import Callable

TEST_NETWORK_RANGE = ipaddress.IPv4Network("10.160.0.0/11")
SUBNET_ATTEMPTS = 8


def get_random_test_subnet() -> str:
    first_octet, second_octet = TEST_NETWORK_RANGE.network_address.packed[:2]

    return f"{first_octet}.{second_octet + secrets.randbelow(32)}.{secrets.randbelow(256)}.0/24"


def is_test_subnet(subnet: str) -> bool:
    network = ipaddress.ip_network(subnet)

    return isinstance(network, ipaddress.IPv4Network) and network.subnet_of(TEST_NETWORK_RANGE)


def is_subnet_overlap(output_lines: list[str]) -> bool:
    return any("overlaps" in line for line in output_lines)


def start_with_free_subnet(start: Callable[[str], tuple[int, list[str]]]) -> int:
    """Calls start with random test subnets until one does not overlap another network."""
    exit_code = 0

    for _ in range(SUBNET_ATTEMPTS):
        exit_code, output_lines = start(get_random_test_subnet())
        if exit_code == 0 or not is_subnet_overlap(output_lines):
            return exit_code

    return exit_code
