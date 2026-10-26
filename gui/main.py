"""PyInstaller entry point; normal development uses `uv run selfextg`."""

from selfext_gui.app import main

if __name__ == "__main__":
    raise SystemExit(main())
