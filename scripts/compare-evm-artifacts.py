#!/usr/bin/env python3
"""Compare public SDK artifacts with the exact revision validated by private CI."""
import json
import sys
from pathlib import Path

current, validated = (Path(root) / "pkg/blockchain/evm/artifacts" for root in sys.argv[1:])
for contract in ("Custody", "ConfigRegistry"):
    if json.loads((current / f"{contract}.abi").read_text()) != json.loads((validated / f"{contract}.abi").read_text()):
        sys.exit(f"{contract} ABI differs from the source-validated revision")
    normalize = lambda path: path.read_text().strip().removeprefix("0x")
    if normalize(current / f"{contract}.bin") != normalize(validated / f"{contract}.bin"):
        sys.exit(f"{contract} bytecode differs from the source-validated revision")
print("Current artifacts match the exact source-validated revision.")
