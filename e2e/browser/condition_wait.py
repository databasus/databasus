import time
from collections.abc import Callable


def wait_until[Observation](
    description: str,
    read: Callable[[], Observation],
    accepts: Callable[[Observation], bool],
    *,
    limit_seconds: int = 120,
) -> Observation:
    deadline = time.monotonic() + limit_seconds

    while True:
        observation = read()
        if accepts(observation):
            return observation

        assert time.monotonic() < deadline, f"timed out waiting for {description}: {observation}"
        time.sleep(0.2)
