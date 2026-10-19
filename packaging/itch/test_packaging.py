import json
import os
import pathlib
import plistlib
try:
    import pty
except ImportError:
    pty = None
import shlex
import shutil
import subprocess
import tempfile
import unittest

from publish import commands
from stage import HERE, TARGETS, stage


class ItchPackagingTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="Sea Battle 2 builds with spaces ")
        self.addCleanup(self.temporary.cleanup)
        self.root = pathlib.Path(self.temporary.name)
        self.dist = self.root / "dist"
        self.dist.mkdir()
        (self.dist / "metadata.json").write_text(json.dumps({"version": "1.0.0"}))
        artifacts = []
        self.sources = {}
        for system, arch in TARGETS:
            binary = self.dist / f"{system}_{arch}" / "sea-battle"
            binary.parent.mkdir()
            binary.write_text('#!/bin/sh\nprintf "%s\\n" "$@"\n')
            binary.chmod(0o755)
            self.sources[(system, arch)] = binary
            artifacts.append({"type": "Binary", "goos": system, "goarch": arch,
                              "path": str(binary), "extra": {"ID": "sea-battle"}})
        (self.dist / "artifacts.json").write_text(json.dumps(artifacts))
        self.out = self.dist / "itch"
        stage(self.dist, self.out)

    def test_all_packages_preserve_release_binaries_and_can_launch(self):
        for (system, arch), channel in TARGETS.items():
            package = self.out / channel
            if system == "linux":
                binary = package / "sea-battle"
                launcher = package / "Play.sh"
            elif system == "windows":
                binary = package / "sea-battle.exe"
                launcher = package / "Play.cmd"
                manifest = (package / ".itch.toml").read_text()
                self.assertIn('path = "sea-battle.exe"', manifest)
                self.assertIn('platform = "windows"', manifest)
                self.assertIn('console = true', manifest)
            else:
                app = package / "Sea Battle 2.app" / "Contents"
                binary = app / "Resources" / "sea-battle"
                launcher = package / "Play.command"
                info = plistlib.loads((app / "Info.plist").read_bytes())
                self.assertTrue((app / "MacOS" / info["CFBundleExecutable"]).is_file())
                self.assertEqual(info["CFBundleShortVersionString"], "1.0.0")
                if os.name != "nt":
                    result = subprocess.run([str(launcher), "argument with spaces", "literal $value"],
                                            capture_output=True, text=True, check=True)
                    self.assertEqual(result.stdout.splitlines(), ["argument with spaces", "literal $value"])
            self.assertEqual(binary.read_bytes(), self.sources[(system, arch)].read_bytes())
            self.assertTrue(os.access(binary, os.X_OK))
            self.assertTrue(launcher.is_file())
            if system != "windows":
                self.assertTrue(os.access(launcher, os.X_OK))
            self.assertTrue((package / ".itch.toml").is_file())

    def mock_path(self, terminal):
        directory = self.root / "mock-bin"
        directory.mkdir(exist_ok=True)
        for name in ["bash", "dirname"]:
            path = directory / name
            if not path.exists():
                path.symlink_to(shutil.which(name))
        command = directory / terminal
        command.write_text('#!/bin/sh\nprintf "%s\\n" "$@"\n')
        command.chmod(0o755)
        return directory

    @unittest.skipUnless(os.name == "nt", "Windows command launcher")
    def test_windows_launcher_preserves_arguments_and_exit_code(self):
        source = self.root / "launcher-fixture.go"
        source.write_text('package main\nimport ("fmt"; "os")\n'
                          'func main() { for _, arg := range os.Args[1:] { fmt.Println(arg) } }\n')
        package = self.out / "windows-amd64"
        (package / "sea-battle.exe").unlink()
        subprocess.run(["go", "build", "-o", str(package / "sea-battle.exe"), str(source)], check=True)
        result = subprocess.run([str(package / "Play.cmd"), "argument with spaces", "literal $value"],
                                shell=True, capture_output=True, text=True, check=True, timeout=10)
        self.assertEqual(result.stdout.splitlines(), ["argument with spaces", "literal $value"])

    @unittest.skipIf(os.name == "nt", "Unix shell launcher")
    def test_linux_launcher_opens_terminal_and_preserves_arguments(self):
        for terminal, prefix in [("xdg-terminal-exec", []), ("gnome-terminal", ["--"]),
                                 ("konsole", ["-e"]), ("wezterm", ["start", "--"])]:
            with self.subTest(terminal=terminal):
                directory = self.mock_path(terminal)
                env = dict(os.environ, PATH=str(directory))
                package = self.out / "linux-amd64"
                result = subprocess.run([str(package / "Play.sh"), "two words", "literal $value"],
                                        env=env, capture_output=True, text=True, check=True)
                self.assertEqual(result.stdout.splitlines(), prefix + [str(package / "sea-battle"), "two words", "literal $value"])
                (directory / terminal).unlink()

    @unittest.skipIf(os.name == "nt", "Unix shell launcher")
    def test_missing_terminal_has_actionable_error(self):
        directory = self.mock_path("temporary-terminal")
        (directory / "temporary-terminal").unlink()
        result = subprocess.run([str(self.out / "linux-amd64" / "Play.sh")],
                                env=dict(os.environ, PATH=str(directory)), capture_output=True, text=True)
        self.assertEqual(result.returncode, 1)
        self.assertIn("./sea-battle", result.stderr)

    @unittest.skipIf(os.name == "nt", "Unix shell launcher")
    def test_xfce_command_string_preserves_paths_and_arguments(self):
        directory = self.mock_path("xfce4-terminal")
        package = self.out / "linux-amd64"
        result = subprocess.run([str(package / "Play.sh"), "two words", "literal $value"],
                                env=dict(os.environ, PATH=str(directory)), capture_output=True,
                                text=True, check=True)
        arguments = result.stdout.splitlines()
        self.assertEqual(arguments[:2], ["--disable-server", "--command"])
        launched = subprocess.run(shlex.split(arguments[2]), capture_output=True, text=True, check=True)
        self.assertEqual(launched.stdout.splitlines(), ["two words", "literal $value"])

    @unittest.skipIf(pty is None, "Requires Unix pseudo-terminal")
    def test_linux_uses_existing_terminal_without_opening_another(self):
        directory = self.mock_path("xdg-terminal-exec")
        master, slave = pty.openpty()
        try:
            process = subprocess.Popen([str(self.out / "linux-amd64" / "Play.sh"), "from existing terminal"],
                                       env=dict(os.environ, PATH=str(directory)),
                                       stdin=slave, stdout=slave, stderr=slave)
            self.assertEqual(process.wait(timeout=5), 0)
            self.assertEqual(os.read(master, 4096).decode().strip(), "from existing terminal")
        finally:
            os.close(master)
            os.close(slave)

    def test_missing_release_target_prevents_packaging(self):
        artifacts = json.loads((self.dist / "artifacts.json").read_text())
        (self.dist / "artifacts.json").write_text(json.dumps(artifacts[:-1]))
        with self.assertRaisesRegex(ValueError, "missing release targets"):
            stage(self.dist, self.root / "incomplete")
        self.assertFalse((self.root / "incomplete").exists())

    def test_publication_validates_all_channels_first(self):
        plan = commands(self.out, "tty-games/sea-battle-2", "v1.0.0", "butler")
        count = len(TARGETS)
        self.assertEqual([command[1] for command in plan], ["validate"] * count + ["push"] * count)
        for command, (system, _) in zip(plan[:count], TARGETS):
            self.assertEqual(command[3], "osx" if system == "darwin" else system)
        for command, channel in zip(plan[count:], TARGETS.values()):
            self.assertIn("tty-games/sea-battle-2:" + channel, command)
            self.assertEqual(command[-2:], ["--userversion", "1.0.0"])
        with self.assertRaisesRegex(ValueError, "stable"):
            commands(self.out, "tty-games/sea-battle-2", "v1.0.0-rc1", "butler")
        (self.out / "mac-arm64" / "VERSION.txt").write_text("2.0.0\n")
        with self.assertRaisesRegex(ValueError, "wrong package version"):
            commands(self.out, "tty-games/sea-battle-2", "1.0.0", "butler")


if __name__ == "__main__":
    unittest.main()

class RetainedReleaseTest(unittest.TestCase):
    def test_restages_retained_archives_without_build_metadata(self):
        import hashlib
        import io
        import tarfile
        import zipfile
        from stage import release_binaries
        with tempfile.TemporaryDirectory() as directory:
            dist = pathlib.Path(directory)
            checksums = []
            for system, arch in TARGETS:
                entry = 'sea-battle.exe' if system == 'windows' else 'sea-battle'
                extension = 'zip' if system == 'windows' else 'tar.gz'
                path = dist / f'sea-battle_1.4.3_{system}_{arch}.{extension}'
                names = [entry, 'LICENSE', 'README.md', 'PLAYING.md']
                if system == 'windows':
                    with zipfile.ZipFile(path, 'w') as archive:
                        for name in names:
                            archive.writestr(name, b'exact release bytes')
                else:
                    with tarfile.open(path, 'w:gz') as archive:
                        for name in names:
                            data = b'exact release bytes'
                            info = tarfile.TarInfo(name)
                            info.size = len(data)
                            info.mode = 0o755
                            archive.addfile(info, io.BytesIO(data))
                checksums.append(hashlib.sha256(path.read_bytes()).hexdigest() + '  ' + path.name)
            (dist / 'checksums.txt').write_text('\n'.join(checksums) + '\n')
            binaries = release_binaries(dist, '1.4.3')
            self.assertEqual(set(binaries), set(TARGETS))
            for binary in binaries.values():
                self.assertEqual(binary.read_bytes(), b'exact release bytes')
            stage(dist, dist / 'itch', 'v1.4.3')
            path.write_bytes(b'corrupt')
            with self.assertRaisesRegex(ValueError, 'checksum'):
                release_binaries(dist, '1.4.3')
