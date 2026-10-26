"""Resolve the standalone Go CLI without depending on Qt."""

import os
from pathlib import Path
import shutil
import sys


def find_selfext() -> str | None:
    name = "selfext.exe" if sys.platform == "win32" else "selfext"
    if getattr(sys, "frozen", False):
        # Prefer the exact CLI shipped with this GUI over any system copy.
        bundled = Path(sys._MEIPASS) / name
        return str(bundled) if bundled.is_file() else None

    if override := os.environ.get("SELFEXT_BIN"):
        path = Path(override).expanduser().resolve()
        return str(path) if path.is_file() and os.access(path, os.X_OK) else None

    # Editable uv installs can use `make build` without modifying PATH.
    root = Path(__file__).resolve().parents[3]
    local = root / "target" / name
    if (root / "go.mod").is_file() and local.is_file() and os.access(local, os.X_OK):
        return str(local)
    return shutil.which(name)
