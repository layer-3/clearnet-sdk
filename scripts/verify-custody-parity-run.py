#!/usr/bin/env python3
"""Check private CI result metadata without copying private source or logs."""
import json
import re
import sys
from pathlib import Path


def verified(runs, sdk_revision, custody_revision):
    if not all(re.fullmatch(r"[0-9a-f]{40}", rev) for rev in (sdk_revision, custody_revision)):
        return False
    return any(
        run.get("status") == "completed"
        and run.get("conclusion") == "success"
        and ((run.get("event") == "workflow_dispatch" and run.get("display_title") == f"EIP-712 parity {sdk_revision}")
             or (run.get("event") == "push" and run.get("display_title") == "EIP-712 parity pinned candidate"))
        and run.get("path") == ".github/workflows/test-eip712-artifacts.yml"
        and run.get("head_sha") == custody_revision
        and run.get("head_repository", {}).get("full_name") == "layer-3/custody"
        for run in runs
    )


if __name__ == "__main__":
    sdk_revision, revision_file, results_file = sys.argv[1:]
    custody_revision = Path(revision_file).read_text().strip()
    runs = json.loads(Path(results_file).read_text())
    if not verified(runs, sdk_revision, custody_revision):
        sys.exit("No successful private custody parity run for these exact SDK and custody revisions. Run the private parity workflow before publishing.")
    print("Verified private custody source/artifact parity for this SDK commit.")
