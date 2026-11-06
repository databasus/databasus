import re


def get_required_match(pattern: str, text: str, flags: int = 0) -> re.Match[str]:
    matched_pattern = re.search(pattern, text, flags)
    if matched_pattern is None:
        raise AssertionError(f"Required pattern did not match: {pattern}")

    return matched_pattern
