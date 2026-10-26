"""Drag-and-drop archive packaging with PySide6."""

import codecs
from pathlib import Path
import platform
import sys

from PySide6.QtCore import QProcess, Qt, Signal, Slot
from PySide6.QtGui import QColor, QDragEnterEvent, QDropEvent, QFont, QPalette
from PySide6.QtWidgets import (
    QApplication,
    QComboBox,
    QDialog,
    QFrame,
    QHBoxLayout,
    QLabel,
    QMainWindow,
    QMessageBox,
    QPlainTextEdit,
    QPushButton,
    QVBoxLayout,
    QWidget,
)

from selfext_gui.cli import find_selfext


class DragDropLabel(QLabel):
    def __init__(self, text, parent=None):
        super().__init__(text, parent)
        self.setAcceptDrops(True)
        # Archive names are plain text, even when they contain HTML-like text.
        self.setTextFormat(Qt.TextFormat.PlainText)

    @staticmethod
    def local_file(event):
        for url in event.mimeData().urls():
            if url.isLocalFile() and Path(url.toLocalFile()).is_file():
                return url.toLocalFile()
        return None

    def dragEnterEvent(self, event: QDragEnterEvent):
        if self.local_file(event):
            event.acceptProposedAction()

    def dropEvent(self, event: QDropEvent):
        if path := self.local_file(event):
            self.setText(path)
            event.acceptProposedAction()


class OutputWindow(QDialog):
    completed = Signal(bool)

    def __init__(self, file_name, parent=None):
        super().__init__(parent)
        self.file_name = file_name
        self.success = None
        self._decoder = codecs.getincrementaldecoder("utf-8")(errors="replace")
        self.setWindowTitle(f"selfext: generating {file_name} ...")
        self.resize(600, 400)
        layout = QVBoxLayout(self)

        self.output_console = QPlainTextEdit(self)
        self.output_console.setReadOnly(True)
        self.output_console.setFont(QFont("Courier", 10))
        palette = self.output_console.palette()
        palette.setColor(QPalette.ColorRole.Base, QColor(0, 0, 0))
        palette.setColor(QPalette.ColorRole.Text, QColor(0, 255, 0))
        self.output_console.setPalette(palette)
        layout.addWidget(self.output_console)

        line = QFrame(self)
        line.setFrameShape(QFrame.Shape.HLine)
        line.setFrameShadow(QFrame.Shadow.Sunken)
        layout.addWidget(line)
        bottom = QHBoxLayout()
        self.status_label = QLabel("Executing...", self)
        bottom.addWidget(self.status_label)
        bottom.addStretch(1)
        self.close_button = QPushButton("Close", self)
        self.close_button.setEnabled(False)
        self.close_button.clicked.connect(self.close)
        bottom.addWidget(self.close_button)
        layout.addLayout(bottom)

        self.process = QProcess(self)
        self.process.setProcessChannelMode(QProcess.ProcessChannelMode.MergedChannels)
        self.process.readyReadStandardOutput.connect(self.read_output)
        self.process.finished.connect(self.process_finished)
        self.process.errorOccurred.connect(self.process_error)

    def start(self, command):
        self.process.start(command[0], command[1:])

    @property
    def running(self):
        return self.process.state() != QProcess.ProcessState.NotRunning

    @Slot()
    def read_output(self):
        text = self._decoder.decode(bytes(self.process.readAllStandardOutput()))
        self.output_console.insertPlainText(text)
        self.output_console.ensureCursorVisible()

    @Slot(int, QProcess.ExitStatus)
    def process_finished(self, code, exit_status):
        self.read_output()
        self.output_console.insertPlainText(self._decoder.decode(b"", final=True))
        success = code == 0 and exit_status == QProcess.ExitStatus.NormalExit
        self.finish(success)

    @Slot(QProcess.ProcessError)
    def process_error(self, error):
        self.output_console.appendPlainText(f"Error: {self.process.errorString()}")
        # FailedToStart emits no finished signal. Crashes finish normally below.
        if error == QProcess.ProcessError.FailedToStart:
            self.finish(False)

    def finish(self, success):
        if self.success is not None:
            return
        self.success = success
        self.status_label.setText("Execution Successful" if success else "Execution Failed")
        self.status_label.setStyleSheet("color: green;" if success else "color: red;")
        self.close_button.setEnabled(True)
        result = "generated successfully" if success else "generation failed"
        self.setWindowTitle(f"selfext: {self.file_name} {result}")
        self.output_console.appendPlainText(f"\n{self.file_name}: {result}.")
        self.completed.emit(success)

    def reject(self):
        # Escape must not destroy a running child process either.
        if not self.running:
            super().reject()

    def closeEvent(self, event):
        if self.running:
            event.ignore()
        else:
            super().closeEvent(event)


class DragDropWindow(QMainWindow):
    def __init__(self):
        super().__init__()
        self.jobs = []
        self.setWindowTitle("selfextg")
        self.resize(600, 400)
        central = QWidget(self)
        self.setCentralWidget(central)
        layout = QVBoxLayout(central)

        self.archive_label = DragDropLabel("Drag and drop an archive here", self)
        self.archive_label.setAlignment(Qt.AlignmentFlag.AlignCenter)
        self.archive_label.setStyleSheet("border: 2px dashed #aaa; padding: 20px;")
        self.archive_label.setFont(QFont("Arial", 14))
        layout.addWidget(self.archive_label)

        controls = QHBoxLayout()
        self.os_combo = QComboBox(self)
        self.os_combo.addItems(["windows", "linux", "darwin"])
        self.arch_combo = QComboBox(self)
        self.os_combo.currentTextChanged.connect(self.on_os_changed)
        controls.addWidget(QLabel("OS:", self))
        controls.addWidget(self.os_combo)
        controls.addWidget(QLabel("Arch:", self))
        controls.addWidget(self.arch_combo)
        controls.addStretch(1)
        self.generate_button = QPushButton("Generate", self)
        self.generate_button.clicked.connect(self.on_generate_clicked)
        controls.addWidget(self.generate_button)
        layout.addLayout(controls)

        host_os = {"win32": "windows", "darwin": "darwin"}.get(sys.platform, "linux")
        self.os_combo.setCurrentText(host_os)
        self.on_os_changed()
        host_arch = {"x86_64": "amd64", "amd64": "amd64", "aarch64": "arm64", "arm64": "arm64", "i686": "386", "x86": "386"}.get(platform.machine().lower())
        if host_arch:
            self.arch_combo.setCurrentText(host_arch)

    @Slot()
    def on_os_changed(self):
        previous = self.arch_combo.currentText()
        self.arch_combo.clear()
        self.arch_combo.addItems(["arm64", "amd64"] if self.os_combo.currentText() == "darwin" else ["amd64", "arm64", "386"])
        if self.arch_combo.findText(previous) >= 0:
            self.arch_combo.setCurrentText(previous)

    @Slot()
    def on_generate_clicked(self):
        archive = Path(self.archive_label.text()).expanduser()
        if not archive.is_file():
            QMessageBox.critical(self, "Error", "Please drop an existing archive file.")
            return
        executable = find_selfext()
        if not executable:
            QMessageBox.critical(self, "Error", "selfext was not found. Run make build, install it on PATH, or set SELFEXT_BIN.")
            return
        target_os = self.os_combo.currentText()
        output = str(archive.resolve()) + (".exe" if target_os == "windows" else ".run")
        command = [executable, "--archive", str(archive.resolve()), "--os", target_os, "--arch", self.arch_combo.currentText(), "--output", output]
        job = OutputWindow(Path(output).name, self)
        job.setAttribute(Qt.WidgetAttribute.WA_DeleteOnClose)
        self.jobs.append(job)
        job.destroyed.connect(lambda _=None: self.jobs.remove(job))
        job.show()
        job.start(command)

    def closeEvent(self, event):
        if any(job.running for job in self.jobs):
            QMessageBox.information(self, "Packaging in progress", "Wait for the active packaging jobs to finish before closing.")
            event.ignore()
        else:
            super().closeEvent(event)


def main() -> int:
    app = QApplication(sys.argv)
    app.setApplicationName("selfextg")
    window = DragDropWindow()
    window.show()
    return app.exec()
