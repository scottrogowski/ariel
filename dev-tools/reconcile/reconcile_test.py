"""Tests for repository reconciliation."""

import unittest

import reconcile


class ReconcileTest(unittest.TestCase):
    """Prevent generated skill guidance from drifting from the Ariel guide."""

    def test_skill_contains_current_guide(self) -> None:
        guide = reconcile.GUIDE_PATH.read_text(encoding="utf-8")
        skill = reconcile.SKILL_PATH.read_text(encoding="utf-8")
        expected = reconcile.render_block(guide)
        self.assertIn(expected, skill)

    # Prevent shared metadata from drifting between agent plugin manifests.
    def test_codex_manifest_matches_claude_manifest(self) -> None:
        expected = reconcile.render_codex_manifest(reconcile.read_claude_manifest())
        actual = reconcile.CODEX_MANIFEST_PATH.read_text(encoding="utf-8")
        self.assertEqual(actual, expected)

    # Prevent marketplace metadata from drifting from the plugin manifest.
    def test_codex_marketplace_matches_claude_manifest(self) -> None:
        expected = reconcile.render_codex_marketplace(reconcile.read_claude_manifest())
        actual = reconcile.CODEX_MARKETPLACE_PATH.read_text(encoding="utf-8")
        self.assertEqual(actual, expected)

    def test_replace_block_rejects_missing_markers(self) -> None:
        with self.assertRaises(ValueError):
            reconcile.replace_block("unmanaged content", "generated content")


if __name__ == "__main__":
    unittest.main()
