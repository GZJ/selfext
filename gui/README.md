# selfext GUI

The optional desktop interface is an independent Python project managed by
[uv](https://docs.astral.sh/uv/), using PySide6 (Qt 6). The Go CLI remains a
standalone project and does not depend on Python or Qt.

## Run

From the repository root:

```sh
make gui-run
```

This builds the native Go CLI, syncs the locked Python environment, and opens the
GUI. Alternatively, after building/installing the CLI:

```sh
cd gui
uv sync --locked
uv run --locked selfextg
# or: uv run --locked python -m selfext_gui
```

The project pins Python 3.13 in `.python-version`. uv can provision that Python
version if needed. Python 3.11–3.14 are supported by the project metadata.
Dependencies are declared in `pyproject.toml` and pinned in `uv.lock`; `gui/.venv/`
is local and is not committed. No manual pip installation is required.

In development, the GUI finds the CLI in this order:

1. `SELFEXT_BIN`, if set to an executable path.
2. The repository's `target/selfext` (`selfext.exe` on Windows).
3. `selfext` on PATH.

The frozen application uses the CLI bundled with it. Drop a local archive,
select the target OS/architecture, and click Generate. Logs and success/failure
appear in a separate window. Active packaging jobs must finish before closing.

## Build a standalone GUI

From the repository root:

```sh
make gui
```

The Make target builds the CLI and runs PyInstaller through uv's locked `build`
dependency group. Output is `target/selfextg` on Linux, `target/selfextg.exe` on
Windows, and `target/selfextg.app` on macOS. Python and Qt are bundled, so users
do not need uv, Python, or a separate CLI installation.

Build on the target operating system with matching Python/Go architectures.
PyInstaller does not cross-compile this application. On Windows, use a Windows
uv/Python installation and a POSIX shell with GNU Make (e.g. MSYS2 or Git Bash
with Make installed). Temporary packaging files live in `build/`.

## Test and update dependencies

```sh
make gui-test
# equivalent, after building the CLI:
uv run --project gui --locked python -m unittest discover -s gui/tests -v

uv lock --project gui --upgrade
```

Tests use Qt's offscreen platform and exercise the real GUI-to-CLI packaging
flow, process failures, target selection, and CLI discovery. Commit both
`pyproject.toml` and `uv.lock` when changing dependencies.
