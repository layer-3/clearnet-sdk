import sys
sys.dont_write_bytecode = True

import importlib.util
from pathlib import Path
import unittest
import tempfile
import subprocess

spec = importlib.util.spec_from_file_location("parity", Path(__file__).with_name("verify-custody-parity-run.py"))
parity = importlib.util.module_from_spec(spec)
spec.loader.exec_module(parity)

class ParityGateTest(unittest.TestCase):
    def test_exact_success_only(self):
        sdk, custody = "1" * 40, "2" * 40
        run = dict(status="completed", conclusion="success", event="workflow_dispatch",
                   path=".github/workflows/test-sdk-artifact-parity.yml", head_sha=custody,
                   display_title=f"SDK artifact parity {sdk}", head_repository={"full_name": "layer-3/custody"})
        self.assertTrue(parity.verified([run], sdk, custody))
        self.assertFalse(parity.verified([], sdk, custody))
        for field, wrong in dict(status="in_progress", conclusion="failure", event="pull_request",
                                 path="other.yml", head_sha="3"*40, display_title="EIP-712 parity other",
                                 head_repository={"full_name": "someone/fork"}).items():
            with self.subTest(field=field):
                self.assertFalse(parity.verified([dict(run, **{field: wrong})], sdk, custody))
        self.assertFalse(parity.verified([run], "master", custody))

class ArtifactComparisonTest(unittest.TestCase):
    def test_changed_artifact_fails(self):
        with tempfile.TemporaryDirectory() as directory:
            roots = [Path(directory) / name for name in ("current", "validated")]
            for root in roots:
                artifacts = root / "pkg/blockchain/evm/artifacts"
                artifacts.mkdir(parents=True)
                for contract in ("Custody", "ConfigRegistry"):
                    (artifacts / f"{contract}.abi").write_text("[]")
                    (artifacts / f"{contract}.bin").write_text("6000")
                sol_artifacts = root / "pkg/blockchain/sol/artifacts"
                sol_artifacts.mkdir(parents=True)
                (sol_artifacts / "custody.json").write_text("{}")
                (sol_artifacts / "custody.so").write_bytes(b"program")
            command = [sys.executable, str(Path(__file__).with_name("compare-evm-artifacts.py")), *map(str, roots)]
            self.assertEqual(subprocess.run(command, capture_output=True).returncode, 0)
            for contract in ("Custody", "ConfigRegistry"):
                for extension, original, changed in (("abi", "[]", '[{"type":"function"}]'), ("bin", "6000", "6001")):
                    file = roots[0] / f"pkg/blockchain/evm/artifacts/{contract}.{extension}"
                    with self.subTest(file=file.name):
                        file.write_text(changed)
                        self.assertNotEqual(subprocess.run(command, capture_output=True).returncode, 0)
                        file.write_text(original)
            for name, original, changed in (("custody.json", "{}", '{"constants":[]}'), ("custody.so", b"program", b"changed")):
                file = roots[0] / "pkg/blockchain/sol/artifacts" / name
                with self.subTest(file=name):
                    if isinstance(original, bytes):
                        file.write_bytes(changed)
                    else:
                        file.write_text(changed)
                    self.assertNotEqual(subprocess.run(command, capture_output=True).returncode, 0)
                    if isinstance(original, bytes):
                        file.write_bytes(original)
                    else:
                        file.write_text(original)

if __name__ == "__main__":
    unittest.main()
