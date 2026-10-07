#!/usr/bin/env python3
"""Validate shipped SDK artifacts without querying their source repositories."""
import hashlib
import json
import re
import sys
from pathlib import Path


def check_checksums(directory, manifest, expected):
    entries = {}
    for line in (directory / manifest).read_text().splitlines():
        match = re.fullmatch(r"([0-9a-f]{64})  ([A-Za-z0-9_.-]+)", line)
        if not match or match[2] in entries:
            raise ValueError(f"Invalid checksum entry in {manifest}: {line}")
        entries[match[2]] = match[1]
    if set(entries) != expected:
        raise ValueError(f"{manifest} must cover exactly {sorted(expected)}")
    for name, digest in entries.items():
        if hashlib.sha256((directory / name).read_bytes()).hexdigest() != digest:
            raise ValueError(f"Checksum mismatch: {name}")


def check(root):
    evm = root / "pkg/blockchain/evm/artifacts"
    sol = root / "pkg/blockchain/sol/artifacts"
    source = (evm / "custody-source-revision").read_text().strip()
    if not re.fullmatch(r"[0-9a-f]{40}", source):
        raise ValueError("Invalid custody source revision")
    if (sol / "custody-program-revision").read_text().strip() != source:
        raise ValueError("Solana and EVM custody source revisions differ")

    abi_files = sorted(evm.glob("*.abi"))
    bin_files = sorted(evm.glob("*.bin"))
    if not abi_files or not bin_files:
        raise ValueError("Missing EVM ABI or bytecode artifacts")
    for path in abi_files:
        if not isinstance(json.loads(path.read_text()), list):
            raise ValueError(f"ABI must be a JSON array: {path.name}")
    for path in bin_files:
        bytecode = path.read_text().strip().removeprefix("0x")
        if not re.fullmatch(r"(?:[0-9a-fA-F]{2})+", bytecode):
            raise ValueError(f"Invalid bytecode: {path.name}")
        if not path.with_suffix(".abi").is_file():
            raise ValueError(f"Missing ABI for {path.name}")
    check_checksums(evm, "artifacts.sha256", {p.name for p in abi_files + bin_files})

    if not isinstance(json.loads((sol / "custody.json").read_text()), dict):
        raise ValueError("Solana IDL must be a JSON object")
    check_checksums(sol, "custody.so.sha256", {"custody.so"})
    if not (sol / "custody.so").read_bytes().startswith(b"\x7fELF"):
        raise ValueError("Solana program must be an ELF binary")


if __name__ == "__main__":
    try:
        check(Path(sys.argv[1] if len(sys.argv) > 1 else "."))
    except (OSError, ValueError) as error:
        sys.exit(str(error))
    print("SDK artifact formats, checksums, and source-revision metadata verified.")
