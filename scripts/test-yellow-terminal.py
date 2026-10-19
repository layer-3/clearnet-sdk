#!/usr/bin/env python3
"""Check hidden input and cancellation in an actual Unix pseudo-terminal."""
import os
import fcntl
import struct
import pathlib
import pty
import select
import signal
import subprocess
import tempfile
import termios
import time
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]
# Published fixture values, never a user's account or passphrase.
KEY = b"0" * 63 + b"1"
PASSPHRASE = b"terminal test passphrase"


class TerminalTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.temp = tempfile.TemporaryDirectory(prefix="yellow-terminal-")
        cls.binary = pathlib.Path(cls.temp.name) / "yellow-tests"
        subprocess.run(["go", "test", "-c", "-o", str(cls.binary), "./internal/yellow"], cwd=ROOT, check=True)

    @classmethod
    def tearDownClass(cls):
        cls.temp.cleanup()

    def launch(self, mode="import", no_color=False, reuse_data=False, redirect_output=False):
        if not reuse_data:
            self.local = tempfile.TemporaryDirectory(prefix="yellow-terminal-data-")
            self.addCleanup(self.local.cleanup)
        self.master, self.slave = pty.openpty()
        fcntl.ioctl(self.slave, termios.TIOCSWINSZ, struct.pack("HHHH",24,80,0,0))
        self.original = termios.tcgetattr(self.slave)
        terminal_fds = [self.master, self.slave]
        def close_terminal():
            while terminal_fds:
                os.close(terminal_fds.pop())
        self.close_terminal = close_terminal
        self.addCleanup(close_terminal)
        env = dict(os.environ, YELLOW_TEST_TERMINAL_DIR=self.local.name, YELLOW_TEST_TERMINAL_MODE=mode, YELLOW_TEST_NO_COLOR="1" if no_color else "0")
        kwargs = {}
        command = [str(self.binary), "-test.run=^TestTerminalDriver$"]
        if mode == "export":
            def attach_terminal():
                os.setsid()
                fcntl.ioctl(self.slave, termios.TIOCSCTTY, 0)
            kwargs["preexec_fn"] = attach_terminal
            # Keep the controlling session alive while checking restored modes;
            # Darwin revokes the tty when its session leader exits.
            command = ["/bin/sh", "-c", '"$1" -test.run=^TestTerminalDriver$; yellow_test_code=$?; printf "\\nYELLOW_TEST_DONE:%s\\n" "$yellow_test_code" > /dev/tty; read -r yellow_test_ack; exit "$yellow_test_code"', "yellow-terminal", str(self.binary)]
        self.process = subprocess.Popen(command, stdin=self.slave,
                                        stdout=subprocess.PIPE if redirect_output else self.slave,
                                        stderr=subprocess.PIPE if redirect_output else self.slave, env=env, **kwargs)
        self.addCleanup(self.stop)
        self.transcript = b""

    def stop(self):
        if self.process.poll() is None:
            self.process.kill()
        self.close_terminal()
        self.process.wait(timeout=5)

    def expect(self, marker, after=0):
        deadline = time.monotonic() + 15
        while marker not in self.transcript[after:]:
            self.assertLess(time.monotonic(), deadline, "terminal prompt timed out")
            ready, _, _ = select.select([self.master], [], [], 0.2)
            if ready:
                self.transcript += os.read(self.master, 65536)

    def assert_terminal_restored(self):
        current, original = termios.tcgetattr(self.slave), list(self.original)
        # Darwin sets PENDIN after canonical input; it is a kernel bookkeeping
        # flag, not an echo/raw-mode change made by the application.
        current[3] &= ~termios.PENDIN
        original[3] &= ~termios.PENDIN
        self.assertEqual(current, original, "terminal input mode was not restored")

    def test_shell_live_completion_history_and_exit(self):
        self.launch("shell")
        self.expect(b"no network / no account")
        mark = len(self.transcript)
        os.write(self.master, b"ver")
        self.expect(b"Show the installed version", after=mark)
        os.write(self.master, b"\t")
        time.sleep(0.1)
        mark = len(self.transcript)
        os.write(self.master, b"\r")
        self.expect(b"yellow test", after=mark)
        self.expect(b"Ready", after=mark)
        mark = len(self.transcript)
        os.write(self.master, b"\x1b[A\r")
        self.expect(b"yellow test", after=mark)
        self.expect(b"Ready", after=mark)
        os.write(self.master, b"exit\r")
        self.assertEqual(self.process.wait(timeout=5), 0)
        self.assert_terminal_restored()
        self.assertNotIn(b"\x1b[?1049h", self.transcript, "shell used alternate screen")
        self.assertNotIn(b"\x1bM", self.transcript, "shell used reverse index and risked scrollback")

    def test_shell_hands_off_secret_input_and_keeps_it_out_of_history(self):
        self.launch("shell", no_color=True)
        self.expect(b"no network / no account")
        os.write(self.master, b"account import terminal-test\r")
        self.expect(b"Import type:")
        os.write(self.master, b"key\n")
        self.expect(b"Enter key (hidden):")
        os.write(self.master, KEY + b"\n")
        self.expect(b"New encryption passphrase")
        os.write(self.master, PASSPHRASE + b"\n")
        self.expect(b"Repeat passphrase")
        mark = len(self.transcript)
        os.write(self.master, PASSPHRASE + b"\n")
        self.expect(b"Ready", after=mark)
        self.expect(b"no network / terminal-test", after=mark)
        os.write(self.master, b"\x1b[A")
        time.sleep(0.2)
        os.write(self.master, b"\x03")
        self.expect(b"Input cleared", after=mark)
        os.write(self.master, b"exit\r")
        self.assertEqual(self.process.wait(timeout=5), 0)
        self.assert_terminal_restored()
        self.assertNotIn(KEY, self.transcript, "shell echoed a private key")
        self.assertNotIn(PASSPHRASE, self.transcript, "shell echoed a passphrase")
        self.assertNotIn(b"\x1b[1;33m", self.transcript, "--no-color ignored")
        self.assertEqual(sorted(p.name for p in pathlib.Path(self.local.name).iterdir()), [".lock", "accounts", "config.json"], "shell persisted command history")

    def test_shell_interrupt_restores_terminal(self):
        self.launch("shell")
        self.expect(b"no network / no account")
        self.process.send_signal(signal.SIGINT)
        self.assertEqual(self.process.wait(timeout=5), 130)
        self.assert_terminal_restored()

    def test_interrupt_during_visible_input(self):
        self.launch()
        self.expect(b"Import type:")
        self.process.send_signal(signal.SIGINT)
        self.assertEqual(self.process.wait(timeout=5), 130)
        self.assert_terminal_restored()

    def test_hidden_import(self):
        self.launch()
        self.expect(b"Import type:")
        os.write(self.master, b"key\n")
        self.expect(b"Enter key (hidden):")
        os.write(self.master, KEY + b"\n")
        self.expect(b"New encryption passphrase")
        os.write(self.master, PASSPHRASE + b"\n")
        self.expect(b"Repeat passphrase")
        os.write(self.master, PASSPHRASE + b"\n")
        self.expect(b'"version":1')
        self.assertEqual(self.process.wait(timeout=5), 0)
        self.assertNotIn(KEY, self.transcript, "private key input was echoed")
        self.assertNotIn(PASSPHRASE, self.transcript, "passphrase input was echoed")
        self.assert_terminal_restored()

    def test_interrupt_during_secret(self):
        self.launch()
        self.expect(b"Import type:")
        os.write(self.master, b"key\n")
        self.expect(b"Enter key (hidden):")
        self.process.send_signal(signal.SIGINT)
        self.assertEqual(self.process.wait(timeout=5), 130)
        self.assert_terminal_restored()

    def test_control_c_during_secret(self):
        self.launch()
        self.expect(b"Import type:")
        os.write(self.master, b"key\n")
        self.expect(b"Enter key (hidden):")
        os.write(self.master, b"\x03")
        self.assertEqual(self.process.wait(timeout=5), 130)
        self.assert_terminal_restored()
        self.assertFalse((pathlib.Path(self.local.name) / "accounts" / "terminal-test.json").exists())

    def import_fixture(self):
        self.launch()
        self.expect(b"Import type:")
        os.write(self.master, b"key\n")
        self.expect(b"Enter key (hidden):")
        os.write(self.master, KEY + b"\n")
        self.expect(b"New encryption passphrase")
        os.write(self.master, PASSPHRASE + b"\n")
        self.expect(b"Repeat passphrase")
        os.write(self.master, PASSPHRASE + b"\n")
        self.expect(b'"version":1')
        self.assertEqual(self.process.wait(timeout=5), 0)
        self.assertNotIn(KEY, self.transcript)

    def test_private_key_export_is_terminal_only(self):
        self.import_fixture()
        self.launch("export", reuse_data=True, redirect_output=True)
        # Prompts are redirected to stderr here; the secret still goes only to /dev/tty.
        deadline = time.monotonic() + 15
        prompt = b""
        while b"Unlock account:" not in prompt:
            self.assertLess(time.monotonic(), deadline)
            ready, _, _ = select.select([self.process.stderr], [], [], 0.2)
            if ready:
                prompt += os.read(self.process.stderr.fileno(), 4096)
        os.write(self.master, PASSPHRASE + b"\n")
        self.expect(b"Anyone with this key can control this account")
        self.assertNotIn(KEY, self.transcript)
        os.write(self.master, b"yes\n")
        self.expect(KEY)
        self.expect(b"operation journal")
        self.expect(b"YELLOW_TEST_DONE:0")
        self.assert_terminal_restored()
        os.write(self.master, b"\n")
        time.sleep(0.1)
        self.close_terminal()
        stdout, stderr = self.process.communicate(timeout=5)
        self.assertEqual(self.process.returncode, 0)
        self.assertNotIn(KEY, stdout + stderr + prompt)
        self.assertNotIn(PASSPHRASE, self.transcript + stdout + stderr + prompt)

    def test_cancelled_private_key_export(self):
        self.import_fixture()
        self.launch("export", reuse_data=True)
        self.expect(b"Unlock account:")
        os.write(self.master, PASSPHRASE + b"\n")
        self.expect(b"Reveal this private key [yes/no]")
        os.write(self.master, b"no\n")
        self.expect(b"YELLOW_TEST_DONE:130")
        self.assert_terminal_restored()
        os.write(self.master, b"\n")
        time.sleep(0.1)
        self.close_terminal()
        self.assertEqual(self.process.wait(timeout=5), 130)
        self.assertNotIn(KEY, self.transcript)


if __name__ == "__main__":
    unittest.main()
