"""Validate the plugin manifest version against semantic Git tags."""

from __future__ import annotations

import json
from pathlib import Path
import re
import subprocess
import sys

REPOSITORY_ROOT = Path(__file__).resolve().parents[2]
MANIFEST_PATH = REPOSITORY_ROOT / ".claude-plugin" / "plugin.json"
VERSION_PATTERN = re.compile(r"^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$")


def parse_version(value: str) -> tuple[int, int, int]:
    """Parse a canonical major.minor.patch version."""
    match = VERSION_PATTERN.fullmatch(value)
    if match is None:
        raise ValueError(f"version {value!r} must use canonical major.minor.patch")
    return tuple(int(component) for component in match.groups())


def next_release_tag(manifest_version: str, tags: list[str]) -> str:
    """Return the next tag when the manifest exceeds all semantic tags."""
    parsed_tags = [
        parse_version(tag[1:])
        for tag in tags
        if tag.startswith("v") and VERSION_PATTERN.fullmatch(tag[1:])
    ]
    if not parsed_tags:
        raise ValueError("repository has no semantic Git tags")

    manifest = parse_version(manifest_version)
    latest = max(parsed_tags)
    if manifest <= latest:
        latest_tag = ".".join(str(component) for component in latest)
        raise ValueError(
            f"manifest version {manifest_version} must exceed latest Git tag v{latest_tag}"
        )
    return f"v{manifest_version}"


def read_manifest_version() -> str:
    """Read the release version from the plugin manifest."""
    manifest = json.loads(MANIFEST_PATH.read_text(encoding="utf-8"))
    version = manifest.get("version")
    if not isinstance(version, str) or not version:
        raise ValueError(f"{MANIFEST_PATH} has no string version")
    return version


def git_tags() -> list[str]:
    """Return every local Git tag."""
    result = subprocess.run(
        ["git", "tag", "--list"],
        cwd=REPOSITORY_ROOT,
        check=True,
        capture_output=True,
        text=True,
    )
    return result.stdout.splitlines()


def main() -> int:
    """Print the valid next release tag or return a failing exit code."""
    try:
        print(next_release_tag(read_manifest_version(), git_tags()))
    except (OSError, ValueError, json.JSONDecodeError, subprocess.CalledProcessError) as error:
        print(f"version_check: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
