"""Real Unix PTY validation; only Gate API is mocked by the surrounding suite.
The child owns no terminal descriptor after exit, and no real credentials are used.
"""
import json
import os
import pty
import select
import signal
import subprocess
import sys
import termios

binary, home, scenario = sys.argv[1:]
master, slave = pty.openpty()
before = termios.tcgetattr(slave)
child = None
try:
    child = subprocess.Popen([binary, "config", "init", "--hide-input"],
                             stdin=slave, stdout=subprocess.PIPE,
                             stderr=subprocess.PIPE, env=os.environ.copy())
    # Wait for the first prompt; a functioning implementation enters raw mode
    # before this point, including when all credentials are pasted at once.
    if not select.select([child.stdout], [], [], 5)[0]:
        raise AssertionError("first prompt timed out")
    prefix = os.read(child.stdout.fileno(), 4096)
    if b"(hidden)" not in prefix:
        raise AssertionError("hidden prompt missing: " + repr(prefix))
    success = scenario in ("paste", "secret-eof", "empty")
    if scenario == "term":
        child.send_signal(signal.SIGTERM)
    else:
        data = {
            "paste": "default\r\nsynthetic-key\r\nsynthetic-中文\x7fx\r\n".encode(),
            "secret-eof": b"default\nsynthetic-key\nsynthetic-secret\x04",
            "empty": b"\n\n\n",
            "ctrl-c": b"synthetic-profile\n\x03",
            "early-eof": b"synthetic-profile\x04",
            "oversize": b"default\n" + b"k" * 65537 + b"\nsecret\n",
        }[scenario]
        # Small writes avoid a blocking PTY write if the bounded parser exits.
        for pos in range(0, len(data), 1024):
            if child.poll() is not None:
                break
            if not select.select([], [master], [], 5)[1]:
                raise AssertionError("input write timed out")
            os.write(master, data[pos:pos+1024])
    out, err = child.communicate(timeout=10)
    output = prefix + out + err
    echo = b""
    while select.select([master], [], [], 0)[0]:
        echo += os.read(master, 4096)
    if b"synthetic-" in output + echo:
        raise AssertionError("credential leaked to terminal/output")
    after = termios.tcgetattr(slave)
    # Darwin sets the transient kernel PENDIN marker on canonical-mode restore,
    # even for a bare stdlib raw/restore round trip. Compare all configured state
    # (including ECHO/ICANON/ISIG and control chars) except that OS pending marker.
    pending = getattr(termios, "PENDIN", 0)
    after[3] &= ~pending
    before[3] &= ~pending
    if after != before:
        raise AssertionError("terminal settings were not restored: " + repr([(i, a, b) for i, (a, b) in enumerate(zip(before, after)) if a != b]))
    path = os.path.join(home, ".gate-cli", "config.yaml")
    if success:
        if child.returncode != 0 or not os.path.isfile(path):
            raise AssertionError("normal hidden input failed: " + repr(output))
        with open(path, "rb") as handle:
            saved = handle.read()
        if scenario == "paste" and "synthetic-中x".encode() not in saved:
            raise AssertionError("UTF8 editing/CRLF lost")
        if scenario == "secret-eof" and b"synthetic-secret" not in saved:
            raise AssertionError("final nonempty EOF lost")
    elif child.returncode == 0 or os.path.exists(path):
        raise AssertionError("invalid input committed configuration")
    print(json.dumps({"scenario": scenario, "restored": True, "success": success}))
finally:
    if child is not None and child.poll() is None:
        child.kill()
        child.communicate()
    os.close(master)
    os.close(slave)
