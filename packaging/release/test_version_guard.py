import pathlib
import subprocess
import sys
import tempfile
import unittest


class TapVersionGuardTest(unittest.TestCase):
    def test_first_publish_retry_and_downgrade(self):
        with tempfile.TemporaryDirectory() as tmp:
            formula = pathlib.Path(tmp) / 'sea-battle.rb'
            guard = pathlib.Path(__file__).with_name('check-version.py')
            def check(version):
                return subprocess.run([sys.executable, str(guard), version, str(formula)],
                                      capture_output=True, text=True).returncode
            self.assertEqual(check('v1.0.0'), 0)
            formula.write_text('  version "1.0.0"\n')
            self.assertEqual(check('v1.0.0'), 0)
            self.assertNotEqual(check('v0.9.0'), 0)
            self.assertEqual(check('v1.0.1'), 0)
            formula.write_text('unknown formula contents')
            self.assertNotEqual(check('v1.0.1'), 0)
