"""Tests for release version validation."""

import unittest

from version_check import next_release_tag, parse_version


class ParseVersionTest(unittest.TestCase):
    """Prevent ambiguous versions from reaching release automation."""

    def test_accepts_semantic_version(self) -> None:
        self.assertEqual(parse_version("12.34.56"), (12, 34, 56))

    def test_rejects_malformed_versions(self) -> None:
        for value in ("1.2", "1.02.3", "1.2.3-beta.1", "1.two.3"):
            with self.subTest(value=value), self.assertRaises(ValueError):
                parse_version(value)


class NextReleaseTagTest(unittest.TestCase):
    """Prevent existing releases from being overwritten or superseded incorrectly."""

    def test_accepts_version_above_highest_tag(self) -> None:
        tags = ["v1.5.0", "preview", "v0.9.0", "v1.10.0", "v2.0.0-beta"]
        self.assertEqual(next_release_tag("2.0.0", tags), "v2.0.0")

    def test_rejects_equal_version(self) -> None:
        with self.assertRaises(ValueError):
            next_release_tag("0.1.0", ["v0.1.0"])

    def test_rejects_lower_version(self) -> None:
        with self.assertRaises(ValueError):
            next_release_tag("0.0.9", ["v0.1.0"])

    def test_rejects_missing_semantic_tags(self) -> None:
        with self.assertRaises(ValueError):
            next_release_tag("0.1.0", ["preview"])


if __name__ == "__main__":
    unittest.main()
