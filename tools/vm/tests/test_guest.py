"""Checkout integrity, repeatable setup and qualification; no package installs."""
from contextlib import ExitStack
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location("guest", Path(__file__).parents[1] / "guest.py")
guest = importlib.util.module_from_spec(spec)
spec.loader.exec_module(guest)


@unittest.skipUnless(sys.platform == "linux", "Linux checkout semantics")
class CheckoutTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        parent = Path(self.temp.name)
        seed = parent / "seed"
        seed.mkdir()
        self.git(seed, "init", "-q")
        self.git(seed, "config", "core.autocrlf", "false")
        (seed / ".gitignore").write_text("/bin/\n/.cache/\n")
        (seed / "go.mod").write_text("module fixture\ngo 1.27.0\n")
        (seed / "main.go").write_text("package main\nfunc main() {}\n")
        (seed / "run.sh").write_text("#!/bin/sh\nexit 0\n")
        (seed / "run.sh").chmod(0o755)
        (seed / "link").symlink_to("main.go")
        self.git(seed, "add", "--", ".gitignore", "go.mod", "main.go", "run.sh", "link")
        self.commit(seed)
        self.root = parent / "checkout 雪 & spaces; 'quote'"
        subprocess.run(["git", "clone", "-q", str(seed), str(self.root)], check=True)
        self.snapshot = guest.verify_tree(self.root)

    @staticmethod
    def git(root, *args):
        return subprocess.check_output(["git", "-C", str(root), *args], stderr=subprocess.DEVNULL)

    def commit(self, root):
        self.git(root, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-qm", "fixture")

    def test_clone_cli_and_generated_outputs(self):
        result = subprocess.run([sys.executable, guest.__file__, "verify", str(self.root)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.strip(), self.snapshot["revision"])
        with guest.locked(self.root):
            (self.root / ".cache/vm/bootstrap.json").write_text("{}")
        (self.root / "bin").mkdir()
        (self.root / "bin/desk").write_bytes(b"fixture executable")
        self.assertEqual(guest.verify_tree(self.root), self.snapshot)

    def test_modified_missing_modes_and_symlinks(self):
        source = self.root / "main.go"
        original = source.read_bytes()
        source.write_text("changed")
        with self.assertRaisesRegex(ValueError, "contents"):
            guest.verify_tree(self.root)
        source.unlink()
        with self.assertRaisesRegex(ValueError, "missing"):
            guest.verify_tree(self.root)
        source.write_bytes(original)
        source.chmod(0o755)
        with self.assertRaisesRegex(ValueError, "mode"):
            guest.verify_tree(self.root)
        source.chmod(0o644)
        (self.root / "link").unlink()
        (self.root / "link").symlink_to("go.mod")
        with self.assertRaisesRegex(ValueError, "contents"):
            guest.verify_tree(self.root)

    def test_untracked_and_ignored_source_refuse_qualification(self):
        for name in ("extra.go", "extra_test.go", "bin/extra.go", ".cache/extra_test.go"):
            with self.subTest(name=name):
                path = self.root / name
                path.parent.mkdir(exist_ok=True)
                path.write_text("package main\n")
                with self.assertRaisesRegex(ValueError, "Unexpected source file"):
                    guest.verify_tree(self.root)
                self.assertTrue(path.exists())
                path.unlink()

    def test_revision_change_and_staged_contents(self):
        (self.root / "main.go").write_text("package main\n// updated\n")
        self.git(self.root, "add", "main.go")
        with self.assertRaisesRegex(ValueError, "contents"):
            guest.verify_tree(self.root)
        self.commit(self.root)
        with self.assertRaisesRegex(ValueError, "revision changed"):
            guest.verify_tree(self.root, self.snapshot)
        self.assertNotEqual(guest.verify_tree(self.root)["revision"], self.snapshot["revision"])

    def test_workflow_symlinks_and_concurrent_run(self):
        (self.root / "bin").mkdir()
        (self.root / "bin/desk").symlink_to("/bin/true")
        with self.assertRaisesRegex(ValueError, "Unexpected source file"):
            guest.verify_tree(self.root)
        (self.root / "bin/desk").unlink()
        with guest.locked(self.root):
            with self.assertRaises(BlockingIOError):
                with guest.locked(self.root):
                    pass
        self.assertEqual(guest.verify_tree(self.root), self.snapshot)

    def test_interrupted_metadata_write_is_retryable(self):
        for name in (".cache/vm/bootstrap.json", ".cache/vm-runs/" + "a" * 32 + "/report.json"):
            with self.subTest(name=name):
                path = self.root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                guest.atomic_json(path, {"complete": False})
                with patch.object(guest.os, "replace", side_effect=OSError("interrupted")):
                    with self.assertRaises(OSError):
                        guest.atomic_json(path, {"complete": True})
                self.assertEqual(json.loads(path.read_text()), {"complete": False})
                self.assertEqual(list(path.parent.glob(".*.writing-*")), [])
                # Simulate a hard interruption that cannot execute finally.
                leftover = path.with_name(f".{path.name}.writing-" + "b" * 32)
                leftover.write_text('{"incomplete":')
                self.assertEqual(guest.verify_tree(self.root), self.snapshot)
                guest.atomic_json(path, {"complete": True})
                self.assertEqual(json.loads(path.read_text()), {"complete": True})
                self.assertTrue(leftover.is_file())
                leftover.unlink()
                leftover.symlink_to(path.name)
                with self.assertRaisesRegex(ValueError, "Unexpected source file"):
                    guest.verify_tree(self.root)
                leftover.unlink()
                unexpected = path.with_name(".writing-extra.go")
                unexpected.write_text("package main\n")
                with self.assertRaisesRegex(ValueError, "Unexpected source file"):
                    guest.verify_tree(self.root)
                unexpected.unlink()

    def bootstrap_mocks(self, package_missing=False, fail_install=False):
        real_run = subprocess.run

        def process(args, **kwargs):
            if args[0] == "dpkg-query":
                return subprocess.CompletedProcess(args, 0, "" if package_missing else "install ok installed")
            return real_run(args, **kwargs)

        def install(args, **kwargs):
            if fail_install and args[:3] == ["sudo", "apt-get", "install"]:
                raise subprocess.CalledProcessError(1, args)
            if args[1] == "build":
                self.assertEqual(kwargs["cwd"], self.root)
                (self.root / "bin/desk").write_bytes(b"fixture binary")

        return (patch.object(guest, "ubuntu_desktop"),
                patch.object(guest, "run", side_effect=install),
                patch.object(guest.subprocess, "run", side_effect=process),
                patch.object(guest.shutil, "which", side_effect=lambda n: "/usr/bin/" + n),
                patch.object(guest, "configure_go", return_value="/fixture/go/bin/go"),
                patch.object(guest, "output", return_value="fixture version"))

    def test_bootstrap_twice_builds_without_reinstalling(self):
        with ExitStack() as stack:
            mocks = [stack.enter_context(p) for p in self.bootstrap_mocks()]
            guest.bootstrap(self.root)
            guest.bootstrap(self.root)
        calls = [c.args[0] for c in mocks[1].call_args_list]
        self.assertFalse(any("apt-get" in c for c in calls))
        self.assertEqual(sum("build" in c for c in calls), 2)
        setup = json.loads((self.root / ".cache/vm/bootstrap.json").read_text())
        self.assertEqual(setup["revision"], self.snapshot["revision"])
        self.assertEqual(setup["code_version"], "fixture version")
        self.assertEqual(guest.verify_tree(self.root), self.snapshot)

    def test_partial_bootstrap_is_retryable(self):
        with ExitStack() as stack:
            for p in self.bootstrap_mocks(package_missing=True, fail_install=True):
                stack.enter_context(p)
            with self.assertRaises(subprocess.CalledProcessError):
                guest.bootstrap(self.root)
        self.assertFalse((self.root / ".cache/vm/bootstrap.json").exists())
        with ExitStack() as stack:
            for p in self.bootstrap_mocks():
                stack.enter_context(p)
            guest.bootstrap(self.root)
        self.assertTrue((self.root / ".cache/vm/bootstrap.json").is_file())

    def test_qualification_records_failure_and_changed_source(self):
        with guest.locked(self.root):
            guest.atomic_json(self.root / ".cache/vm/bootstrap.json", {"go": "/fixture/go", "revision": self.snapshot["revision"]})
        real_run = subprocess.run
        for changed in (False, True):
            def process(args, **kwargs):
                if args[0] == "/fixture/go":
                    if changed:
                        (self.root / "extra_test.go").write_text("package main\n")
                    return subprocess.CompletedProcess(args, 0 if changed else 1)
                return real_run(args, **kwargs)
            with patch.object(guest, "ubuntu_desktop"), patch.object(guest.sys.stdin, "isatty", return_value=True), \
                    patch.dict(os.environ, {"PATH": os.environ["PATH"], "DISPLAY": ":1"}, clear=True), \
                    patch.object(guest, "output", return_value="fixture version"), \
                    patch.object(guest.subprocess, "run", side_effect=process):
                if changed:
                    with self.assertRaisesRegex(ValueError, "Unexpected source file"):
                        guest.qualify(self.root)
                else:
                    guest.qualify(self.root)
        reports = [json.loads(p.read_text()) for p in (self.root / ".cache/vm-runs").glob("*/report.json")]
        self.assertEqual(len(reports), 2)
        self.assertEqual({r["source_integrity"] for r in reports}, {"passed", "failed"})
        for report in reports:
            self.assertEqual(report["result"], "incomplete")
            self.assertEqual(set(report["observations"].values()), {"unverified"})


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
                    patch.object(Path, "read_text", autospec=True, side_effect=lambda p: {
                        "/etc/os-release": f'ID={distro}\nVERSION_ID="{version}"\n',
                        "/proc/cmdline": "root=UUID=fixture ro",
                        "/proc/mounts": "/dev/sda2 / ext4 rw 0 0\n",
                    }[str(p)]), \
                    patch.object(guest, "output", return_value=architecture):
                if accepted:
                    guest.ubuntu_desktop()
                else:
                    with self.assertRaises(ValueError):
                        guest.ubuntu_desktop()

    def test_live_installer_sessions_are_rejected(self):
        for command_line in ('boot=casper quiet splash', 'BOOT=casper persistent', 'boot=live', '"boot=casper"'):
            with self.subTest(command_line=command_line), patch.object(guest.os, "getuid", return_value=1000), \
                    patch.object(Path, "read_text", return_value=command_line):
                with self.assertRaisesRegex(ValueError, "Boot the installed Ubuntu"):
                    guest.ubuntu_desktop()

    def test_live_overlay_without_boot_option_is_rejected(self):
        # Ubuntu 26.04.1's actual GRUB entry has no boot=casper/live option.
        for mounts in ("overlay / overlay rw 0 0\n", "/dev/loop0 / squashfs ro 0 0\n",
                       "tmpfs / tmpfs rw 0 0\n", "missing root mount\n"):
            with self.subTest(mounts=mounts), patch.object(guest.os, "getuid", return_value=1000), \
                    patch.object(Path, "read_text", autospec=True, side_effect=lambda p: mounts
                                 if str(p) == "/proc/mounts" else "BOOT_IMAGE=/casper/vmlinuz --- quiet splash"):
                with self.assertRaisesRegex(ValueError, "persistent root filesystem"):
                    guest.ubuntu_desktop()


if __name__ == "__main__":
    unittest.main()
