#!/usr/bin/env python3
"""
Generate a short-lived HS256 JWT for smoke testing from .env file.
Reads AUTH_JWT_SECRET, AUTH_JWT_ISSUER, AUTH_JWT_AUDIENCE from .env.
Prints the token to stdout.

Usage:
  TOKEN=$(python3 scripts/gen_jwt.py)
  TOKEN=$(python3 scripts/gen_jwt.py --team-id <uuid>)
"""
import argparse
import base64
import hashlib
import hmac
import json
import pathlib
import time


def b64url(data: bytes) -> str:
    return base64.urlsafe_b64encode(data).rstrip(b"=").decode()


def load_env(path: str = ".env") -> dict:
    env = {}
    for line in pathlib.Path(path).read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, v = line.split("=", 1)
        env[k.strip()] = v.strip()
    return env


def main() -> None:
    parser = argparse.ArgumentParser(description="Mint HS256 JWT from .env")
    parser.add_argument(
        "--team-id",
        default="11111111-1111-1111-1111-111111111111",
        help="team_id claim (default: all-ones UUID without grant)",
    )
    parser.add_argument(
        "--ttl",
        type=int,
        default=3600,
        help="token lifetime in seconds (default: 3600)",
    )
    parser.add_argument(
        "--env-file",
        default=".env",
        help="path to .env file (default: .env)",
    )
    args = parser.parse_args()

    env = load_env(args.env_file)
    secret = env["AUTH_JWT_SECRET"]
    iss = env.get("AUTH_JWT_ISSUER", "")
    aud = env.get("AUTH_JWT_AUDIENCE", "")

    now = int(time.time())
    header = {"alg": "HS256", "typ": "JWT"}
    payload = {
        "team_id": args.team_id,
        "iss": iss,
        "aud": aud,
        "iat": now,
        "nbf": now,
        "exp": now + args.ttl,
    }

    h = b64url(json.dumps(header, separators=(",", ":")).encode())
    p = b64url(json.dumps(payload, separators=(",", ":")).encode())
    msg = f"{h}.{p}".encode()
    sig = b64url(hmac.new(secret.encode(), msg, hashlib.sha256).digest())
    print(f"{h}.{p}.{sig}")


if __name__ == "__main__":
    main()
