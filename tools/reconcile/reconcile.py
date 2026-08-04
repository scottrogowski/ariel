"""Reconcile generated repository files with their authored sources."""

from __future__ import annotations

from pathlib import Path
import sys

REPOSITORY_ROOT = Path(__file__).resolve().parents[2]
GUIDE_PATH = REPOSITORY_ROOT / "internal" / "guide" / "guide.txt"
SKILL_PATH = REPOSITORY_ROOT / "skills" / "create-walkthrough" / "SKILL.md"
BEGIN_MARKER = "<!-- BEGIN GENERATED: ariel guide — regenerate with `make reconcile`; do not edit by hand -->"
END_MARKER = "<!-- END GENERATED: ariel guide -->"


def render_block(guide: str) -> str:
    """Render the generated Markdown block."""
    return f"{BEGIN_MARKER}\n\n```\n{guide.rstrip()}\n```\n\n{END_MARKER}"


def replace_block(skill: str, generated_block: str) -> str:
    """Replace the marker-delimited block in a skill document."""
    start = skill.find(BEGIN_MARKER)
    if start < 0:
        raise ValueError(f"begin marker not found: {BEGIN_MARKER!r}")
    end = skill.find(END_MARKER, start)
    if end < 0:
        raise ValueError(f"end marker not found: {END_MARKER!r}")
    end += len(END_MARKER)
    return skill[:start] + generated_block + skill[end:]


def reconcile() -> None:
    """Update every generated repository file."""
    guide = GUIDE_PATH.read_text(encoding="utf-8")
    skill = SKILL_PATH.read_text(encoding="utf-8")
    updated = replace_block(skill, render_block(guide))
    SKILL_PATH.write_text(updated, encoding="utf-8")


def main() -> int:
    """Reconcile generated files or return a failing exit code."""
    try:
        reconcile()
    except (OSError, ValueError) as error:
        print(f"reconcile: {error}", file=sys.stderr)
        return 1
    print(f"reconciled {SKILL_PATH.relative_to(REPOSITORY_ROOT)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
