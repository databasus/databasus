import base64
import hashlib
import hmac
import struct
import time

TIME_STEP_SECONDS = 30
CODE_DIGITS = 6


def compute_authenticator_code(secret: str, step_offset: int = 0) -> str:
    """Computes the RFC 6238 code an authenticator app shows for the secret, so the scenario plays
    the phone. A positive offset gives the code of a later step, which the instance still accepts
    and which a code already spent in the current step cannot block."""
    padded_secret = secret.upper() + "=" * (-len(secret) % 8)
    key = base64.b32decode(padded_secret)
    step = int(time.time()) // TIME_STEP_SECONDS + step_offset
    digest = hmac.new(key, struct.pack(">Q", step), hashlib.sha1).digest()
    offset = digest[-1] & 0x0F
    truncated = struct.unpack(">I", digest[offset : offset + 4])[0] & 0x7FFFFFFF

    return str(truncated % 10**CODE_DIGITS).zfill(CODE_DIGITS)
