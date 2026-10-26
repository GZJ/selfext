"""Exercise Qt widgets and actual asynchronous subprocesses without a display."""

import os
os.environ["QT_QPA_PLATFORM"] = "offscreen"

from pathlib import Path
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import patch
import zipfile

from PySide6.QtCore import QMimeData, QPointF, QProcess, Qt, QUrl
from PySide6.QtGui import QDropEvent
from PySide6.QtTest import QTest
from PySide6.QtWidgets import QApplication

from selfext_gui import cli
from selfext_gui.app import DragDropLabel, DragDropWindow, OutputWindow


class GuiTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.app = QApplication.instance() or QApplication([])

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.windows = []

    def tearDown(self):
        for window in reversed(self.windows):
            if isinstance(window, OutputWindow) and window.running:
                window.process.kill()
                window.process.waitForFinished(5000)
            window.close()
        self.app.processEvents()
        self.temp.cleanup()

    def wait_for_job(self, job):
        self.windows.append(job)
        deadline = time.monotonic() + 15
        while job.success is None and time.monotonic() < deadline:
            QTest.qWait(10)
        self.assertIsNotNone(job.success, "process did not complete")
        self.assertFalse(job.running)
        self.assertTrue(job.close_button.isEnabled())

    def test_targets(self):
        window = DragDropWindow()
        self.windows.append(window)
        window.os_combo.setCurrentText("darwin")
        self.assertEqual([window.arch_combo.itemText(i) for i in range(window.arch_combo.count())], ["arm64", "amd64"])
        window.os_combo.setCurrentText("windows")
        self.assertNotEqual(window.arch_combo.findText("386"), -1)

    def test_local_drag_drop(self):
        label = DragDropLabel("drop")
        self.windows.append(label)
        archive = self.root / "<archive>.zip"
        # Angle brackets are not legal Windows filenames.
        if sys.platform == "win32":
            archive = self.root / "archive.zip"
        archive.write_bytes(b"fixture")
        mime = QMimeData()
        mime.setUrls([QUrl.fromLocalFile(str(archive))])
        event = QDropEvent(QPointF(5, 5), Qt.DropAction.CopyAction, mime, Qt.MouseButton.LeftButton, Qt.KeyboardModifier.NoModifier)
        label.dropEvent(event)
        self.assertEqual(label.text(), str(archive))
        self.assertEqual(label.textFormat(), Qt.TextFormat.PlainText)
        mime.setUrls([QUrl("https://example.com/archive.zip")])
        remote = QDropEvent(QPointF(5, 5), Qt.DropAction.CopyAction, mime, Qt.MouseButton.LeftButton, Qt.KeyboardModifier.NoModifier)
        label.dropEvent(remote)
        self.assertEqual(label.text(), str(archive))

    def test_output_and_failed_exit(self):
        job = OutputWindow("test.run")
        job.show()
        job.start([sys.executable, "-c", "import os,sys; os.write(1, '中文 <log>\\n'.encode('utf-8')); sys.exit(2)"])
        self.wait_for_job(job)
        self.assertFalse(job.success)
        self.assertIn("中文 <log>", job.output_console.toPlainText())

    def test_failed_to_start(self):
        job = OutputWindow("test.run")
        job.start([str(self.root / "missing-command")])
        self.wait_for_job(job)
        self.assertFalse(job.success)
        self.assertIn("Error:", job.output_console.toPlainText())

    def test_running_job_cannot_be_closed(self):
        job = OutputWindow("test.run")
        job.show()
        job.start([sys.executable, "-c", "import time; time.sleep(0.3)"])
        self.assertTrue(job.running)
        self.assertFalse(job.close())
        job.reject()
        self.assertTrue(job.isVisible())
        self.wait_for_job(job)
        self.assertTrue(job.success)
        self.assertTrue(job.close())

    def test_real_packaging(self):
        executable = cli.find_selfext()
        self.assertIsNotNone(executable, "build the CLI first with make build")
        archive = self.root / "资料 & archive.zip"
        with zipfile.ZipFile(archive, "w") as z:
            z.writestr("nested/hello.txt", "packaged by PySide6")
        window = DragDropWindow()
        self.windows.append(window)
        window.archive_label.setText(str(archive))
        window.on_generate_clicked()
        self.assertEqual(len(window.jobs), 1)
        job = window.jobs[0]
        # Owned by the main window; don't retain its wrapper after Qt deletes it.
        deadline = time.monotonic() + 15
        while job.success is None and time.monotonic() < deadline:
            QTest.qWait(10)
        self.assertTrue(job.success, job.output_console.toPlainText())
        suffix = ".exe" if sys.platform == "win32" else ".run"
        output = Path(str(archive) + suffix)
        self.assertTrue(output.is_file())
        destination = self.root / "unpacked"
        result = subprocess.run([str(output), "--directory", str(destination)], capture_output=True, timeout=15)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((destination / "nested/hello.txt").read_text(), "packaged by PySide6")

    def test_invalid_archive(self):
        window = DragDropWindow()
        self.windows.append(window)
        window.archive_label.setText(str(self.root / "missing.zip"))
        with patch("selfext_gui.app.QMessageBox.critical") as error:
            window.on_generate_clicked()
        error.assert_called_once()
        self.assertFalse(window.jobs)


class DiscoveryTests(unittest.TestCase):
    def test_explicit_override(self):
        with patch.dict(os.environ, {"SELFEXT_BIN": sys.executable}):
            self.assertEqual(cli.find_selfext(), str(Path(sys.executable).resolve()))
        with patch.dict(os.environ, {"SELFEXT_BIN": "/nonexistent/selfext"}):
            self.assertIsNone(cli.find_selfext())

    def test_frozen_uses_bundled_cli(self):
        with tempfile.TemporaryDirectory() as temp:
            name = "selfext.exe" if sys.platform == "win32" else "selfext"
            binary = Path(temp) / name
            binary.write_bytes(b"bundled")
            with patch.object(sys, "frozen", True, create=True), patch.object(sys, "_MEIPASS", temp, create=True):
                with patch.dict(os.environ, {"SELFEXT_BIN": sys.executable}):
                    self.assertEqual(cli.find_selfext(), str(binary))
                binary.unlink()
                self.assertIsNone(cli.find_selfext())


if __name__ == "__main__":
    unittest.main()
