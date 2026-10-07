import sys
sys.dont_write_bytecode = True

import hashlib
import importlib.util
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("artifacts", Path(__file__).with_name("check-release-artifacts.py"))
artifacts = importlib.util.module_from_spec(spec)
spec.loader.exec_module(artifacts)


class ArtifactCheckTest(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        self.evm = self.root / "pkg/blockchain/evm/artifacts"
        self.sol = self.root / "pkg/blockchain/sol/artifacts"
        self.evm.mkdir(parents=True)
        self.sol.mkdir(parents=True)
        (self.evm / "Custody.abi").write_text("[]")
        (self.evm / "Custody.bin").write_text("6000")
        (self.evm / "custody-source-revision").write_text("1" * 40)
        (self.sol / "custody-program-revision").write_text("1" * 40)
        (self.sol / "custody.json").write_text("{}")
        (self.sol / "custody.so").write_bytes(b"\x7fELFprogram")
        self.checksums()

    def checksums(self):
        for directory, manifest, names in (
            (self.evm, "artifacts.sha256", ["Custody.abi", "Custody.bin"]),
            (self.sol, "custody.so.sha256", ["custody.so"]),
        ):
            (directory / manifest).write_text("".join(
                f"{hashlib.sha256((directory / name).read_bytes()).hexdigest()}  {name}\n"
                for name in names
            ))

    def test_valid_checkout_needs_no_external_validation_record(self):
        artifacts.check(self.root)

    def test_artifact_corruption_fails(self):
        for path, changed in (
            (self.evm / "Custody.abi", b'[{"type":"function"}]'),
            (self.evm / "Custody.bin", b"6001"),
            (self.sol / "custody.so", b"\x7fELFchanged"),
        ):
            with self.subTest(path=path.name):
                original = path.read_bytes()
                path.write_bytes(changed)
                with self.assertRaisesRegex(ValueError, "Checksum mismatch"):
                    artifacts.check(self.root)
                path.write_bytes(original)

    def test_missing_or_extra_manifest_entries_fail(self):
        manifest = self.evm / "artifacts.sha256"
        original = manifest.read_text()
        for contents in (original.splitlines()[0] + "\n", original + "0" * 64 + "  extra.bin\n"):
            with self.subTest(contents=contents):
                manifest.write_text(contents)
                with self.assertRaisesRegex(ValueError, "must cover exactly"):
                    artifacts.check(self.root)

    def test_invalid_formats_fail_even_with_updated_checksums(self):
        for path, changed in (
            (self.evm / "Custody.abi", b"{}"),
            (self.evm / "Custody.bin", b"not-bytecode"),
            (self.sol / "custody.json", b"[]"),
            (self.sol / "custody.so", b"not-ELF"),
        ):
            with self.subTest(path=path.name):
                original = path.read_bytes()
                path.write_bytes(changed)
                self.checksums()
                with self.assertRaises(ValueError):
                    artifacts.check(self.root)
                path.write_bytes(original)
                self.checksums()

    def test_source_revision_metadata_is_checked(self):
        for path, changed in (
            (self.evm / "custody-source-revision", "master"),
            (self.sol / "custody-program-revision", "2" * 40),
        ):
            with self.subTest(path=path.name):
                original = path.read_text()
                path.write_text(changed)
                with self.assertRaises(ValueError):
                    artifacts.check(self.root)
                path.write_text(original)


if __name__ == "__main__":
    unittest.main()
