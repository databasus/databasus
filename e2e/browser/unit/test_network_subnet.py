import pytest
from network_subnet import (
    SUBNET_ATTEMPTS,
    get_random_test_subnet,
    is_subnet_overlap,
    is_test_subnet,
    start_with_free_subnet,
)

OVERLAP_OUTPUT = ["Error response from daemon: invalid pool request: Pool overlaps with other one"]


class TestRandomTestSubnet:
    def test_takes_a_24_bit_subnet_from_the_test_range(self) -> None:
        subnets = {get_random_test_subnet() for _ in range(200)}

        assert all(subnet.endswith(".0/24") for subnet in subnets)
        assert all(is_test_subnet(subnet) for subnet in subnets), subnets


class TestTestSubnet:
    @pytest.mark.parametrize(
        ("subnet", "is_inside"),
        [
            ("10.160.0.0/24", True),
            ("10.191.255.0/24", True),
            ("172.18.0.0/16", False),
            ("192.168.16.0/20", False),
            ("10.192.0.0/24", False),
            ("10.128.0.0/9", False),
        ],
    )
    def test_accepts_only_subnets_inside_the_test_range(self, subnet: str, is_inside: bool) -> None:
        assert is_test_subnet(subnet) is is_inside


class TestStartWithFreeSubnet:
    def test_tries_another_subnet_when_one_overlaps(self) -> None:
        tried_subnets: list[str] = []

        def start(subnet: str) -> tuple[int, list[str]]:
            tried_subnets.append(subnet)
            if len(tried_subnets) < 3:
                return 1, OVERLAP_OUTPUT

            return 0, []

        assert start_with_free_subnet(start) == 0
        assert len(tried_subnets) == 3

    def test_returns_another_failure_without_trying_again(self) -> None:
        tried_subnets: list[str] = []

        def start(subnet: str) -> tuple[int, list[str]]:
            tried_subnets.append(subnet)

            return 17, ["container databasus is unhealthy"]

        assert start_with_free_subnet(start) == 17
        assert len(tried_subnets) == 1

    def test_returns_the_overlap_failure_when_every_attempt_overlaps(self) -> None:
        tried_subnets: list[str] = []

        def start(subnet: str) -> tuple[int, list[str]]:
            tried_subnets.append(subnet)

            return 1, OVERLAP_OUTPUT

        assert start_with_free_subnet(start) == 1
        assert len(tried_subnets) == SUBNET_ATTEMPTS

    def test_recognizes_the_daemon_overlap_message(self) -> None:
        assert is_subnet_overlap(OVERLAP_OUTPUT)
        assert not is_subnet_overlap(["network created"])
