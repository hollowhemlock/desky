#!/usr/bin/env python3
"""Ubuntu checkout setup and observed desktop qualification; stdlib only."""
import argparse
import contextlib
import datetime
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import shlex
import shutil
import stat
import subprocess
import sys
import tarfile
import tempfile
import urllib.request
import uuid


def sha256(path):
    with open(path, "rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def atomic_json(path, value):
    path = Path(path)
    temp = path.with_name(f".{path.name}.writing-{uuid.uuid4().hex}")
    fd = os.open(temp, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as stream:
            json.dump(value, stream, indent=2, ensure_ascii=False)
            stream.write("\n")
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temp, path)
    finally:
        temp.unlink(missing_ok=True)


def unlinked(path):
    path = Path(os.path.abspath(path))
    for part in [path, *path.parents]:
        if part.is_symlink():
            raise ValueError("Linked workflow path refused")
    return path


@contextlib.contextmanager
def locked(root):
    import fcntl
    root = unlinked(root)
    state = unlinked(root / ".cache/vm")
    state.mkdir(parents=True, exist_ok=True)
    lock = state / "operation.lock"
    fd = os.open(lock, os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    try:
        if not stat.S_ISREG(os.fstat(fd).st_mode):
            raise ValueError("Invalid workflow lock")
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        yield
    finally:
        os.close(fd)


def git_snapshot(root):
    def git(*args):
        return subprocess.check_output(["git", "-C", str(root), *args], stderr=subprocess.DEVNULL)

    if Path(os.fsdecode(git("rev-parse", "--show-toplevel")).rstrip("\n")) != root:
        raise ValueError("Run from the root of a cloned Desky repository")
    revision = git("rev-parse", "--verify", "HEAD").decode().strip()
    if not re.fullmatch(r"[0-9a-f]{40}|[0-9a-f]{64}", revision):
        raise ValueError("Cannot identify the checked-out commit")
    files = []
    for entry in git("ls-tree", "-rz", "--full-tree", revision).split(b"\0"):
        if not entry:
            continue
        metadata, name = entry.split(b"\t", 1)
        mode, kind, oid = metadata.decode().split()
        name = os.fsdecode(name)
        if kind != "blob" or mode not in ("100644", "100755", "120000"):
            raise ValueError("Submodules and unsupported Git entries cannot be qualified")
        if PurePosixPath(name).parts[0] in (".cache", "bin"):
            raise ValueError("Workflow output directories must not be committed")
        files.append({"path": name, "type": "symlink" if mode == "120000" else "file",
                      "executable": mode == "100755", "oid": oid})
    return {"revision": revision, "files": files}


def generated_file(name):
    if name in (".cache/vm/bootstrap.json", ".cache/vm/operation.lock", "bin/desk"):
        return True
    # A killed process cannot remove its temporary metadata. These exact names
    # stay outside source discovery and are never read as completed records.
    if re.fullmatch(r"\.cache/(?:vm/\.bootstrap\.json|vm-runs/[0-9a-f]{32}/\.report\.json)\.writing-[0-9a-f]{32}", name):
        return True
    # The verification runner and Go package discovery ignore this dot-directory.
    # Only this workflow's run namespace may accumulate application state and logs.
    return bool(re.fullmatch(r"\.cache/vm-runs/[0-9a-f]{32}/(?:report\.json|verification\.log|fixture/.+|xdg/.+)", name))


def verify_tree(root, snapshot=None):
    root = unlinked(root)
    current = git_snapshot(root)
    if snapshot is not None and current != snapshot:
        raise ValueError("Checked-out revision changed during the workflow")
    snapshot = current
    expected = {item["path"]: item for item in snapshot["files"]}
    parents = {str(p) for n in expected for p in PurePosixPath(n).parents if str(p) != "."}
    found = set()
    for base, directories, files in os.walk(root, followlinks=False):
        if Path(base) == root:
            # Git metadata is not source; support both clones and linked worktrees.
            directories[:] = [n for n in directories if n != ".git"]
            files = [n for n in files if n != ".git"]
        for name in directories + files:
            path = Path(base) / name
            rel = path.relative_to(root).as_posix()
            info = path.lstat()
            if stat.S_ISDIR(info.st_mode):
                workflow_dir = (
                    rel in ("bin", ".cache", ".cache/vm", ".cache/vm-runs")
                    or re.fullmatch(r"\.cache/vm-runs/[0-9a-f]{32}(?:/(?:fixture|xdg)(?:/.*)?)?", rel))
                if rel not in parents and not workflow_dir:
                    raise ValueError(f"Unexpected source directory: {rel}")
                continue
            item = expected.get(rel)
            if item is None:
                if generated_file(rel) and stat.S_ISREG(info.st_mode):
                    continue
                raise ValueError(f"Unexpected source file: {rel}")
            digest = hashlib.new("sha1" if len(item["oid"]) == 40 else "sha256")
            if item["type"] == "symlink":
                if not stat.S_ISLNK(info.st_mode):
                    raise ValueError(f"Changed source symlink: {rel}")
                data = os.fsencode(os.readlink(path))
                digest.update(f"blob {len(data)}\0".encode())
                digest.update(data)
            else:
                if not stat.S_ISREG(info.st_mode):
                    raise ValueError(f"Changed source type: {rel}")
                if bool(info.st_mode & 0o111) != item["executable"]:
                    raise ValueError(f"Changed executable mode: {rel}")
                digest.update(f"blob {info.st_size}\0".encode())
                with path.open("rb") as stream:
                    for chunk in iter(lambda: stream.read(65536), b""):
                        digest.update(chunk)
            if digest.hexdigest() != item["oid"]:
                raise ValueError(f"Changed source contents: {rel}")
            found.add(rel)
    if found != set(expected):
        raise ValueError("Committed source files are missing")
    return snapshot


def run(args, **kwargs):
    return subprocess.run(args, check=True, **kwargs)


def output(args):
    return subprocess.check_output(args, text=True, stderr=subprocess.STDOUT, timeout=30).strip()


def ubuntu_desktop():
    if sys.platform != "linux" or os.getuid() == 0:
        raise ValueError("Run as your normal Ubuntu desktop user, not root")
    if any(token.lower() in ("boot=casper", "boot=live") for token in shlex.split(Path("/proc/cmdline").read_text())):
        raise ValueError("Boot the installed Ubuntu system, not live installation media")
    # Current Ubuntu live media can omit boot=casper and still use an overlay root.
    mounts = (line.split() for line in Path("/proc/mounts").read_text().splitlines())
    roots = [mount[2] for mount in mounts if len(mount) >= 3 and mount[1] == "/"]
    if not roots or any(kind in ("overlay", "aufs", "squashfs", "tmpfs", "ramfs", "rootfs") for kind in roots):
        raise ValueError("Boot the installed Ubuntu system with a persistent root filesystem")
    release = dict(line.split("=", 1) for line in Path("/etc/os-release").read_text().splitlines() if "=" in line)
    if release.get("ID", "").strip('"') != "ubuntu" or release.get("VERSION_ID", "").strip('"') not in ("26.04", "24.04"):
        raise ValueError("This bootstrap supports Ubuntu 26.04 and 24.04 LTS only")
    if output(["uname", "-m"]) != "x86_64":
        raise ValueError("This bootstrap supports amd64 only")


def download(url, destination):
    with urllib.request.urlopen(url, timeout=120) as response, open(destination, "xb") as dest:
        shutil.copyfileobj(response, dest)


def configure_go(root):
    import re
    required = re.search(r"(?m)^go (\d+\.\d+(?:\.\d+)?)$", (root / "go.mod").read_text()).group(1)

    def version(value):
        return tuple(int(part) for part in value.split(".")) + (0,) * (3 - len(value.split(".")))

    installed = shutil.which("go")
    if installed:
        match = re.search(r"\bgo(\d+\.\d+(?:\.\d+)?)\b", output([installed, "version"]))
        if match and version(match.group(1)) >= version(required):
            return installed
    toolchains = unlinked(Path.home() / ".local/share/desky-vm/toolchains")
    toolchains.mkdir(parents=True, exist_ok=True)
    destination = toolchains / ("go" + required)
    executable = destination / "go/bin/go"
    if destination.exists():
        if executable.is_file() and output([str(executable), "version"]).startswith("go version go" + required + " "):
            return str(executable)
        raise ValueError("Existing toolchain is incomplete; preserving it for inspection")
    with urllib.request.urlopen("https://go.dev/dl/?mode=json&include=all", timeout=120) as response:
        releases = json.load(response)
    release = next(r for r in releases if r["version"] == "go" + required)
    package = next(f for f in release["files"] if f["os"] == "linux" and f["arch"] == "amd64" and f["kind"] == "archive")
    staging = Path(tempfile.mkdtemp(prefix=".install-", dir=toolchains))
    archive = staging / "go.tar.gz"
    download("https://go.dev/dl/" + package["filename"], archive)
    if sha256(archive) != package["sha256"]:
        raise ValueError("Go download checksum mismatch; incomplete files preserved")
    with tarfile.open(archive) as tar:
        tar.extractall(staging, filter="data")
    if not output([str(staging / "go/bin/go"), "version"]).startswith("go version go" + required + " "):
        raise ValueError("Downloaded Go version mismatch")
    os.rename(staging, destination)
    return str(executable)


def bootstrap(root):
    root = unlinked(root)
    snapshot = verify_tree(root)
    ubuntu_desktop()
    with locked(root):
        run(["sudo", "-v"])
        packages = ["git", "build-essential", "ca-certificates", "curl", "gnupg", "xdg-utils",
                    "gnome-terminal", "gnome-text-editor", "python3"]
        missing = [p for p in packages if subprocess.run(["dpkg-query", "-W", "-f=${Status}", p],
                    stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True).stdout != "install ok installed"]
        if not shutil.which("firefox"):
            missing.append("firefox")
        if missing:
            run(["sudo", "apt-get", "update"])
            run(["sudo", "apt-get", "install", "-y", *missing])
        if not shutil.which("code"):
            source = Path("/etc/apt/sources.list.d/desky-vm-code.sources")
            key = Path("/usr/share/keyrings/desky-vm-microsoft.gpg")
            content = ("Types: deb\nURIs: https://packages.microsoft.com/repos/code\nSuites: stable\n"
                       "Components: main\nArchitectures: amd64\nSigned-By: " + str(key) + "\n")
            # Reuse an existing configured Microsoft repository; never append a second source.
            sources = [Path("/etc/apt/sources.list"), *Path("/etc/apt/sources.list.d").glob("*.list"),
                       *Path("/etc/apt/sources.list.d").glob("*.sources")]
            configured = any("packages.microsoft.com/repos/code" in p.read_text() for p in sources if p.exists())
            if not configured:
                if source.exists() and source.read_text() != content:
                    raise ValueError("Existing VS Code source differs; preserving it")
                with tempfile.TemporaryDirectory(prefix="desky-code-") as temp:
                    temp = Path(temp)
                    download("https://packages.microsoft.com/keys/microsoft.asc", temp / "key.asc")
                    run(["gpg", "--batch", "--dearmor", "--output", str(temp / "key.gpg"), str(temp / "key.asc")])
                    if key.exists() and key.read_bytes() != (temp / "key.gpg").read_bytes():
                        raise ValueError("Existing package key differs; preserving it")
                    run(["sudo", "install", "-m", "0644", str(temp / "key.gpg"), str(key)])
                    (temp / "code.sources").write_text(content)
                    run(["sudo", "install", "-m", "0644", str(temp / "code.sources"), str(source)])
            run(["sudo", "apt-get", "update"])
            run(["sudo", "apt-get", "install", "-y", "code"])
        go = configure_go(root)
        for executable in ("git", "gcc", "xdg-open", "gnome-terminal", "gnome-text-editor", "firefox", "code"):
            if not shutil.which(executable):
                raise ValueError("Required executable unavailable: " + executable)
        build_env = os.environ.copy()
        build_env["PATH"] = str(Path(go).parent) + os.pathsep + build_env.get("PATH", "")
        (root / "bin").mkdir(exist_ok=True)
        run([go, "build", "-o", "bin/desk", "./cmd/desk"], cwd=root, env=build_env)
        verify_tree(root, snapshot)
        atomic_json(root / ".cache/vm/bootstrap.json", {"schema": 1, "go": go,
                    "go_version": output([go, "version"]), "packages": output(["dpkg-query", "-W", *packages]),
                    "code_version": output(["code", "--version"]),
                    "revision": snapshot["revision"]})
    print("Setup complete. Built:", root / "bin/desk")
    print("Optional desktop qualification: python3 tools/vm/guest.py qualify .")


OBSERVATIONS = {
    "consent": "Did Desky request approval and complete the prompt on first entry?",
    "editor": "Did VS Code open the displayed fixture directory?",
    "terminal_directory": "In the opened terminal, does pwd match the displayed fixture directory?",
    "browser": "Did Firefox display example.com with the run ID in its URL fragment?",
    "named_app": "Did GNOME Text Editor show the fixture note and run ID?",
    "cli_return": "Did control return promptly after Desky exited?",
    "survival": "After Desky exited, did all requested applications remain available?",
    "repeat_open": "Did the second entry open the requested resources with no new Desky consent prompt?",
}


def observe(prompt):
    try:
        value = input(prompt + " [p=passed/f=failed/Enter=unverified] ").strip().lower()
    except EOFError:
        value = ""
    return {"p": "passed", "f": "failed"}.get(value, "unverified")


def qualify(root):
    root = unlinked(root)
    snapshot = verify_tree(root)
    ubuntu_desktop()
    if (not sys.stdin.isatty() or not (os.getenv("DISPLAY") or os.getenv("WAYLAND_DISPLAY"))
            or any(os.getenv(k) for k in ("SSH_CONNECTION", "SSH_TTY", "WSL_DISTRO_NAME", "WSL_INTEROP"))):
        raise ValueError("Run qualification from the logged-in Ubuntu desktop terminal")
    setup = json.loads((root / ".cache/vm/bootstrap.json").read_text())
    go = setup["go"]
    if setup["revision"] != snapshot["revision"] or not Path(go).is_absolute():
        raise ValueError("Run bootstrap again for this checked-out revision")
    with locked(root):
        run_id = uuid.uuid4().hex
        run_dir = root / ".cache/vm-runs" / run_id
        run_dir.mkdir(parents=True)
        report = {"schema": 1, "revision": snapshot["revision"], "source": "git-checkout",
                  "run_id": run_id, "time_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
                  "os": Path("/etc/os-release").read_text(), "architecture": output(["uname", "-m"]),
                  "session": {k: os.getenv(k, "") for k in ("XDG_CURRENT_DESKTOP", "XDG_SESSION_TYPE")},
                  "source_integrity": "passed", "automated": "unverified", "observations": dict.fromkeys(OBSERVATIONS, "unverified")}
        report_path = run_dir / "report.json"
        atomic_json(report_path, report)
        try:
            report["versions"] = {name: output([name, "--version"]) for name in
                                  ("code", "firefox", "gnome-terminal", "gnome-text-editor")}
            report["versions"]["go"] = output([go, "version"])
            env = os.environ.copy()
            env["PATH"] = str(Path(go).parent) + os.pathsep + env["PATH"]
            with open(run_dir / "verification.log", "w") as log:
                result = subprocess.run([go, "run", "./tools/verify", "-race"], cwd=root, env=env,
                                        stdout=log, stderr=subprocess.STDOUT)
            report["automated"] = "passed" if result.returncode == 0 else "failed"
            report["verification_exit"] = result.returncode
            verify_tree(root, snapshot)
            if result.returncode:
                return report
            fixture = run_dir / "fixture" / "-Desky 雪 & spaces; punctuation"
            fixture.mkdir(parents=True)
            note = fixture / "qualification.txt"
            note.write_text("Desky desktop qualification\nRun: " + run_id + "\n")
            workspace = f'''schema_version = 1
[workspace]
name = "VM qualification"
[[resource]]
id = "editor"
type = "editor"
path = "."
[[resource]]
id = "terminal"
type = "terminal"
path = "."
[[resource]]
id = "browser"
type = "url"
url = "https://example.com/#desky-{run_id}"
[[resource]]
id = "note"
type = "app"
profile = "note"
'''
            (fixture / "workspace.toml").write_text(workspace)
            # Keep desktop/session environment intact; isolate only Desky's XDG storage.
            for var, folder in (("XDG_CONFIG_HOME", "config"), ("XDG_DATA_HOME", "data"), ("XDG_STATE_HOME", "state")):
                location = run_dir / "xdg" / folder
                location.mkdir(parents=True)
                env[var] = str(location)
            config = Path(env["XDG_CONFIG_HOME"]) / "desky"
            config.mkdir()
            toml_quote = lambda text: json.dumps(str(text), ensure_ascii=False)
            (config / "config.toml").write_text(
                'schema_version = 1\n[launchers.editor]\nexecutable = ' + toml_quote(shutil.which("code")) +
                '\nargs = ["--new-window", "{path}"]\n[launchers.terminal]\nexecutable = ' + toml_quote(shutil.which("gnome-terminal")) +
                '\nargs = ["--working-directory", "{path}"]\n[apps.note]\nexecutable = ' + toml_quote(shutil.which("gnome-text-editor")) +
                '\nargs = [' + toml_quote(note) + ']\n')
            executable = str(root / "bin/desk")
            print("Fixture directory:", fixture, "\nRun:", run_id, flush=True)
            parent_cwd = os.getcwd()
            first = subprocess.run([executable, "open", str(fixture)], env=env)
            report["first_open_exit"] = first.returncode
            for name, question in OBSERVATIONS.items():
                if name != "repeat_open":
                    report["observations"][name] = observe(question)
                    atomic_json(report_path, report)
            print("Repeating entry using the same fixture and approval.", flush=True)
            second = subprocess.run([executable, "open", str(fixture)], env=env)
            report["repeat_open_exit"] = second.returncode
            report["observations"]["repeat_open"] = observe(OBSERVATIONS["repeat_open"])
            report["parent_cwd_unchanged"] = os.getcwd() == parent_cwd
            report["dispatch"] = "passed" if first.returncode == second.returncode == 0 else "failed"
        finally:
            try:
                verify_tree(root, snapshot)
            except Exception:
                report["source_integrity"] = "failed"
            report["result"] = "passed" if (report["source_integrity"] == report["automated"] == report.get("dispatch") == "passed"
                                and report.get("parent_cwd_unchanged") is True
                                and all(v == "passed" for v in report["observations"].values())) else "incomplete"
            atomic_json(report_path, report)
            print("Report:", report_path, "\nRun ID:", run_id, "\nResult:", report["result"])
        return report


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    for name in ("verify", "bootstrap", "qualify"):
        commands.add_parser(name).add_argument("root", type=Path)
    args = parser.parse_args()
    if args.command == "verify":
        print(verify_tree(args.root)["revision"])
    elif args.command == "bootstrap":
        bootstrap(args.root)
    else:
        report = qualify(args.root)
        if report["result"] != "passed":
            return 1
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (Exception, KeyboardInterrupt) as error:
        print("VM workflow failed:", str(error), file=sys.stderr)
        sys.exit(1)
