#!/usr/bin/env python3
"""Validate mongo-secret.yaml app credentials (ReaperC2 uses discrete MONGO_* env, not a URI)."""
from __future__ import annotations

import argparse
import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    print("PyYAML required: pip install pyyaml", file=sys.stderr)
    sys.exit(1)


def load_yaml(path: Path) -> dict:
    data = yaml.safe_load(path.read_text())
    if not isinstance(data, dict):
        raise SystemExit(f"Invalid YAML: {path}")
    return data


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--mongo-secret",
        type=Path,
        default=Path(__file__).resolve().parent.parent / "mongo-secret.yaml",
    )
    args = parser.parse_args()

    mongo = load_yaml(args.mongo_secret)
    string_data = mongo.get("stringData") or {}

    username = (string_data.get("app_username") or "").strip()
    password = string_data.get("app_password") or ""
    if not username or not password:
        print("mongo-secret.yaml must define app_username and app_password", file=sys.stderr)
        return 1

    print(f"mongo-secret.yaml app user {username!r} is set.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
