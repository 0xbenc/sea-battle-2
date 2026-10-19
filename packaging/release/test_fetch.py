import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('tty_release_publish', Path(__file__).with_name('fetch.py'))
publisher = importlib.util.module_from_spec(spec)
spec.loader.exec_module(publisher)

class ReleaseReplayTest(unittest.TestCase):
    def test_download_replay_checks_every_asset_digest(self):
        tag = 'v1.4.3'
        files = {}
        for system, arch in publisher.TARGETS:
            extension = 'zip' if system == 'windows' else 'tar.gz'
            files[f'sea-battle_1.4.3_{system}_{arch}.{extension}'] = (system + arch).encode()
        files['checksums.txt'] = ''.join(hashlib.sha256(data).hexdigest() + '  ' + name + '\n' for name, data in files.items()).encode()
        release = {'tag_name': tag, 'draft': False, 'assets': [
            {'name': name, 'size': len(data), 'digest': 'sha256:' + hashlib.sha256(data).hexdigest(),
             'browser_download_url': f'https://github.com/0xbenc/sea-battle-2/releases/download/{tag}/{name}'}
            for name, data in files.items()]}
        def fetch(url, limit, api=False):
            return json.dumps(release).encode() if api else files[url.rsplit('/', 1)[-1]]
        with tempfile.TemporaryDirectory() as tmp, patch.object(publisher, 'fetch', side_effect=fetch):
            directory = Path(tmp)
            publisher.release_files(tag, directory)
            self.assertEqual(len(list(directory.iterdir())), 7)
            release['assets'][0]['digest'] = 'sha256:' + '0' * 64
            with self.assertRaisesRegex(ValueError, 'identity mismatch'):
                publisher.release_files(tag, directory)

    def test_invalid_tag_never_contacts_network(self):
        with patch.object(publisher, 'fetch') as fetch:
            with self.assertRaises(ValueError):
                publisher.release_files('../../bad', Path('unused'))
            fetch.assert_not_called()
