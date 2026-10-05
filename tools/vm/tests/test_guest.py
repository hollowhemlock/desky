"""Filesystem and failure-path tests; no VM or package installation required."""
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch

sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location("guest", Path(__file__).parents[1] / "guest.py")
guest = importlib.util.module_from_spec(spec)
spec.loader.exec_module(guest)


class ExportTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.parent = Path(self.temp.name)
        self.incoming = self.parent / ".incoming-test"
        self.incoming.mkdir()
        self.manifest = {"schema": 1, "revision": "a" * 40, "archive_sha256": "", "files": []}
        with tarfile.open(self.incoming / "source.tar", "w") as archive:
            for name, contents, mode in [("go.mod", b"module fixture\ngo 1.27.0\n", 0o644),
                                         ("source/main.go", b"package main\n", 0o644),
                                         ("tools/run.sh", b"#!/bin/sh\nexit 0\n", 0o755)]:
                entry = tarfile.TarInfo(name)
                entry.size, entry.mode = len(contents), mode
                archive.addfile(entry, io.BytesIO(contents))
                self.manifest["files"].append({"path": name, "type": "file", "executable": mode == 0o755,
                                               "sha256": hashlib.sha256(contents).hexdigest()})
        self.manifest["archive_sha256"] = guest.sha256(self.incoming / "source.tar")
        (self.incoming / "manifest.json").write_text(json.dumps(self.manifest))

    def publish(self):
        return guest.publish(self.incoming, self.parent, self.manifest["archive_sha256"])

    def test_publish_repeat_and_generated_outputs(self):
        root = self.publish()
        self.assertEqual(self.publish(), root)
        (root / "bin").mkdir()
        (root / "bin/desk").write_bytes(b"executable")
        self.assertEqual(guest.verify_tree(root), self.manifest)
        self.assertEqual(self.publish(), root)

    def test_added_source_or_test_invalidates_revision(self):
        root = self.publish()
        for name in ("source/extra.go", "source/extra_test.go", "bin/extra_test.go"):
            with self.subTest(name=name):
                file = root / name
                file.parent.mkdir(exist_ok=True)
                file.write_text("package main\n")
                with self.assertRaisesRegex(ValueError, "Unexpected source file"):
                    guest.verify_tree(root)
                with self.assertRaises(ValueError):
                    self.publish()
                self.assertTrue(file.exists())
                file.unlink()

    def test_modified_missing_mode_and_unexpected_directory(self):
        root = self.publish()
        source = root / "source/main.go"
        source.write_text("changed")
        with self.assertRaisesRegex(ValueError, "contents"):
            guest.verify_tree(root)
        source.unlink()
        with self.assertRaisesRegex(ValueError, "missing"):
            guest.verify_tree(root)

    def test_bootstrap_reuses_non_dpkg_editor(self):
        root = self.publish()

        def metadata(args):
            if args[0] == 'dpkg-query':
                self.assertNotIn('code', args)
            return 'fixture version'

        with patch.object(guest, 'ubuntu_desktop'), patch.object(guest, 'run'), \
                patch.object(guest, 'configure_go', return_value='/usr/local/bin/go'), \
                patch.object(guest.shutil, 'which', side_effect=lambda name: '/usr/bin/' + name), \
                patch.object(guest.subprocess, 'run', return_value=subprocess.CompletedProcess([], 0, 'install ok installed')), \
                patch.object(guest, 'output', side_effect=metadata):
            guest.bootstrap(root)
        self.assertEqual(json.loads((root / '.desky-vm/bootstrap.json').read_text())['code_version'], 'fixture version')

    def test_bad_checksum_and_incomplete_archive_never_published(self):
        with self.assertRaisesRegex(ValueError, "checksum"):
            guest.publish(self.incoming, self.parent, "0" * 64)
        self.assertFalse((self.parent / self.manifest["revision"]).exists())
        self.manifest["files"].append({"path": "missing", "type": "file", "executable": False, "sha256": "0" * 64})
        (self.incoming / "manifest.json").write_text(json.dumps(self.manifest))
        with self.assertRaisesRegex(ValueError, "incomplete"):
            self.publish()
        self.assertFalse((self.parent / self.manifest["revision"]).exists())

    def test_existing_different_manifest_and_partial_staging_preserved(self):
        root = self.publish()
        original = (root / "source/main.go").read_bytes()
        self.manifest["archive_sha256"] = "1" * 64
        (root / ".desky-vm/manifest.json").write_text(json.dumps(self.manifest))
        with self.assertRaises(ValueError):
            guest.publish(self.incoming, self.parent, guest.sha256(self.incoming / "source.tar"))
        self.assertEqual((root / "source/main.go").read_bytes(), original)

    @unittest.skipUnless(sys.platform == "linux", "Linux filesystem semantics")
    def test_symlink_destination_and_generated_symlink_rejected(self):
        target = self.parent / "unrelated"
        target.mkdir()
        destination = self.parent / self.manifest["revision"]
        destination.symlink_to(target)
        with self.assertRaises(ValueError):
            self.publish()
        destination.unlink()
        root = self.publish()
        (root / "bin").mkdir()
        (root / "bin/desk").symlink_to("/bin/true")
        with self.assertRaises(ValueError):
            guest.verify_tree(root)

    @unittest.skipUnless(sys.platform == "linux", "Linux locking")
    def test_concurrent_publication_refuses_without_changes(self):
        with guest.locked(self.parent):
            with self.assertRaises(BlockingIOError):
                self.publish()
        self.assertFalse((self.parent / self.manifest["revision"]).exists())
        self.publish()


class ObservationTests(unittest.TestCase):
    def test_only_explicit_observations_pass(self):
        for answer, expected in [("p", "passed"), ("f", "failed"), ("", "unverified"), ("yes", "unverified")]:
            with patch("builtins.input", return_value=answer):
                self.assertEqual(guest.observe("Question"), expected)
        with patch("builtins.input", side_effect=EOFError):
            self.assertEqual(guest.observe("Question"), "unverified")


@unittest.skipUnless(sys.platform == "linux", "Linux guest prerequisites")
class GuestPlatformTests(unittest.TestCase):
    def test_accepted_lts_and_rejected_platforms(self):
        for distro, version, architecture, uid, accepted in (
            ("ubuntu", "26.04", "x86_64", 1000, True),
            ("ubuntu", "24.04", "x86_64", 1000, True),
            ("ubuntu", "25.10", "x86_64", 1000, False),
            ("debian", "26.04", "x86_64", 1000, False),
            ("ubuntu", "26.04", "aarch64", 1000, False),
            ("ubuntu", "26.04", "x86_64", 0, False),
        ):
            with self.subTest(distro=distro, version=version, architecture=architecture, uid=uid), \
                    patch.object(guest.os, "getuid", return_value=uid), \
                    patch.object(Path, "read_text", return_value=f'ID={distro}\nVERSION_ID="{version}"\n'), \
                    patch.object(guest, "output", return_value=architecture):
                if accepted:
                    guest.ubuntu_desktop()
                else:
                    with self.assertRaises(ValueError):
                        guest.ubuntu_desktop()


@unittest.skipUnless(sys.platform == "linux", "Bash and Unix account command fixtures")
class FinalizationTests(unittest.TestCase):
    def test_failed_account_checks_do_not_publish_success(self):
        finalizer = (Path(__file__).parents[1] / "finalize.sh").read_text()
        for failure in ("passwd", "unlocked", "group", "sudo", "", "vendor"):
            with self.subTest(failure=failure), tempfile.TemporaryDirectory() as temp:
                root = Path(temp)
                binaries = root / "bin"
                binaries.mkdir()
                scripts = {
                    "passwd": '#!/bin/sh\n[ "$FAIL" = passwd ] && exit 1\nif [ "$1" = --status ]; then if [ "$FAIL" = unlocked ]; then echo "root P"; else echo "root L"; fi; fi\nexit 0\n',
                    "id": '#!/bin/sh\nif [ "$FAIL" = group ]; then echo dev; else echo "dev sudo"; fi\n',
                    "sudo": '#!/bin/sh\n[ "$FAIL" != sudo ]\n',
                    "chown": '#!/bin/sh\nexit 0\n',
                    "install": '#!/bin/sh\nshift 7\nmkdir -p "$1"\n',
                }
                for name, source in scripts.items():
                    path = binaries / name
                    path.write_text(source)
                    path.chmod(0o755)
                attempt = "a" * 8 + "-" + "b" * 4 + "-" + "c" * 4 + "-" + "d" * 4 + "-" + "e" * 12
                script = finalizer.replace("__GUEST_USER__", "dev").replace("__ATTEMPT__", attempt)
                script = script.replace("/var/lib/desky-vm", str(root / "records"))
                vendor_exit = 1 if failure == "vendor" else 0
                script = f'if [ {vendor_exit} = 0 ]; then\n(\n{script}\n)\nelse exit 1; fi\n'
                env = dict(os.environ, PATH=str(binaries) + os.pathsep + os.environ["PATH"], FAIL=failure)
                result = subprocess.run(["bash", "-c", script], env=env, capture_output=True, text=True)
                record = root / "records/installed.json"
                if failure:
                    self.assertNotEqual(result.returncode, 0, result.stderr)
                    self.assertFalse(record.exists())
                else:
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertEqual(json.loads(record.read_text()), {"attempt": attempt, "user": "dev", "root_locked": True, "sudo": True})


if __name__ == "__main__":
    unittest.main()
