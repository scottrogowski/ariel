"""Reconcile generated repository files with their authored sources."""

from __future__ import annotations

import json
from pathlib import Path
import sys

REPOSITORY_ROOT = Path(__file__).resolve().parents[2]
GUIDE_PATH = REPOSITORY_ROOT / "internal" / "guide" / "guide.txt"
SKILL_PATH = REPOSITORY_ROOT / "skills" / "create-walkthrough" / "SKILL.md"
CLAUDE_MANIFEST_PATH = REPOSITORY_ROOT / ".claude-plugin" / "plugin.json"
CODEX_MANIFEST_PATH = REPOSITORY_ROOT / ".codex-plugin" / "plugin.json"
BEGIN_MARKER = "<!-- BEGIN GENERATED: ariel guide — regenerate with `make reconcile`; do not edit by hand -->"
END_MARKER = "<!-- END GENERATED: ariel guide -->"
SHARED_MANIFEST_FIELDS = (
    "name",
    "version",
    "description",
    "author",
    "homepage",
    "repository",
    "license",
    "keywords",
)


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


def read_claude_manifest() -> dict[str, object]:
    """Read shared plugin metadata from the authored manifest."""
    manifest = json.loads(CLAUDE_MANIFEST_PATH.read_text(encoding="utf-8"))
    if not isinstance(manifest, dict):
        raise ValueError(f"{CLAUDE_MANIFEST_PATH} must contain an object")
    missing_fields = [field for field in SHARED_MANIFEST_FIELDS if field not in manifest]
    if missing_fields:
        missing = ", ".join(missing_fields)
        raise ValueError(f"{CLAUDE_MANIFEST_PATH} lacks required fields: {missing}")
    return manifest


def render_codex_manifest(claude_manifest: dict[str, object]) -> str:
    """Render the generated Codex plugin manifest."""
    author = claude_manifest["author"]
    if not isinstance(author, dict) or not isinstance(author.get("name"), str):
        raise ValueError(f"{CLAUDE_MANIFEST_PATH} has no author name")
    manifest = {field: claude_manifest[field] for field in SHARED_MANIFEST_FIELDS}
    manifest.update(
        {
            "skills": "./skills/",
            "interface": {
                "displayName": "Ariel",
                "shortDescription": "Create guided Mermaid diagram walkthroughs.",
                "longDescription": "Ariel teaches Codex to create and render narrated Mermaid diagram walkthroughs.",
                "developerName": author["name"],
                "category": "Productivity",
                "capabilities": ["Command-line", "Visualization"],
                "defaultPrompt": "Create an Ariel walkthrough for this system.",
            },
        }
    )
    return json.dumps(manifest, indent=2) + "\n"


def reconcile() -> None:
    """Update every generated repository file."""
    guide = GUIDE_PATH.read_text(encoding="utf-8")
    skill = SKILL_PATH.read_text(encoding="utf-8")
    updated = replace_block(skill, render_block(guide))
    SKILL_PATH.write_text(updated, encoding="utf-8")
    codex_manifest = render_codex_manifest(read_claude_manifest())
    CODEX_MANIFEST_PATH.write_text(codex_manifest, encoding="utf-8")


def main() -> int:
    """Reconcile generated files or return a failing exit code."""
    try:
        reconcile()
    except (OSError, ValueError) as error:
        print(f"reconcile: {error}", file=sys.stderr)
        return 1
    print("reconciled generated repository files")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
