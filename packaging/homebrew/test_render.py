import unittest

from render import ARCHES, render


class FormulaReleaseTest(unittest.TestCase):
    def setUp(self):
        self.checksums = "\n".join(
            f"{index:064x}  sea-battle_1.0.0_{arch}.tar.gz"
            for index, arch in enumerate(ARCHES, 1)
        )

    def test_all_platforms_use_release_checksums(self):
        formula = render("v1.0.0", self.checksums)
        self.assertNotIn("{{", formula)
        self.assertIn('version "1.0.0"', formula)
        for index, arch in enumerate(ARCHES, 1):
            self.assertIn(f"/v1.0.0/sea-battle_1.0.0_{arch}.tar.gz", formula)
            self.assertIn(f'sha256 "{index:064x}"', formula)

    def test_incomplete_release_is_rejected(self):
        with self.assertRaisesRegex(ValueError, "missing"):
            render("1.0.0", self.checksums.splitlines()[0])

    def test_invalid_checksums_are_rejected(self):
        with self.assertRaisesRegex(ValueError, "invalid SHA-256"):
            render("1.0.0", "bogus  sea-battle_1.0.0_linux_amd64.tar.gz")

    def test_duplicate_archive_is_rejected(self):
        with self.assertRaisesRegex(ValueError, "duplicate"):
            render("1.0.0", self.checksums + "\n" + self.checksums.splitlines()[0])

    def test_prereleases_and_invalid_versions_are_rejected(self):
        for version in ["v1.0.0-rc1", "", "v1.0", "v01.0.0", '1.0.0"']:
            with self.subTest(version=version), self.assertRaisesRegex(ValueError, "stable"):
                render(version, self.checksums)


if __name__ == "__main__":
    unittest.main()
