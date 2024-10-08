import sys
import os
import shutil
import subprocess
from pathlib import Path
from PyQt5.QtWidgets import (
    QApplication,
    QMainWindow,
    QLabel,
    QVBoxLayout,
    QHBoxLayout,
    QWidget,
    QComboBox,
    QPushButton,
    QMessageBox,
    QDialog,
    QTextEdit,
    QFrame,
)
from PyQt5.QtCore import Qt, QThread, pyqtSignal
from PyQt5.QtGui import QDropEvent, QDragEnterEvent, QFont, QColor, QPalette


class DragDropLabel(QLabel):
    def __init__(self, text, parent=None):
        super().__init__(text, parent)
        self.setAcceptDrops(True)

    def dragEnterEvent(self, event: QDragEnterEvent):
        if event.mimeData().hasUrls():
            event.acceptProposedAction()

    def dropEvent(self, event: QDropEvent):
        urls = event.mimeData().urls()
        if urls:
            file_path = urls[0].toLocalFile()
            self.setText(file_path)
            self.setStyleSheet(
                "border: 2px dashed #aaa; padding: 20px; background-color: white;"
            )


class OutputWindow(QDialog):
    def __init__(self, file_name, parent=None):
        super().__init__(parent)
        self.file_name = file_name
        self.initUI()

    def initUI(self):
        self.setWindowTitle(f"selfext: generating {self.file_name}.exe ...")
        self.setGeometry(200, 200, 600, 400)
        layout = QVBoxLayout()

        self.outputConsole = QTextEdit(self)
        self.outputConsole.setReadOnly(True)
        self.outputConsole.setFont(QFont("Courier", 10))

        palette = self.outputConsole.palette()
        palette.setColor(QPalette.Base, QColor(0, 0, 0))
        palette.setColor(QPalette.Text, QColor(0, 255, 0))
        self.outputConsole.setPalette(palette)
        layout.addWidget(self.outputConsole)

        bottom_layout = QHBoxLayout()

        self.statusLabel = QLabel("Executing...", self)
        self.statusLabel.setAlignment(Qt.AlignLeft | Qt.AlignVCenter)
        bottom_layout.addWidget(self.statusLabel)

        bottom_layout.addStretch(1)

        self.closeButton = QPushButton("Close", self)
        self.closeButton.clicked.connect(self.close)
        self.closeButton.setEnabled(False)
        bottom_layout.addWidget(self.closeButton)

        line = QFrame()
        line.setFrameShape(QFrame.HLine)
        line.setFrameShadow(QFrame.Sunken)
        layout.addWidget(line)

        layout.addLayout(bottom_layout)
        self.setLayout(layout)

    def appendOutput(self, text):
        self.outputConsole.append(text)

    def setStatus(self, success):
        if success:
            self.statusLabel.setText("Execution Successful")
            self.statusLabel.setStyleSheet("color: green;")
            self.closeButton.setEnabled(True)
            self.setWindowTitle(
                f"selfext: {self.file_name}.exe generated successfully."
            )
        else:
            self.statusLabel.setText("Execution Failed")
            self.statusLabel.setStyleSheet("color: red;")
            self.closeButton.setEnabled(True)
            self.setWindowTitle(
                f"selfext: failed to generate {self.file_name}.exe in selfext!!"
            )


class DragDropWindow(QMainWindow):
    def __init__(self):
        super().__init__()
        self.initUI()
        self.workers = []

    def initUI(self):
        self.setWindowTitle("selfextg")
        self.setGeometry(100, 100, 600, 400)

        self.archiveLabel = DragDropLabel("Drag and drop a file here", self)
        self.archiveLabel.setAlignment(Qt.AlignCenter)
        self.archiveLabel.setStyleSheet("border: 2px dashed #aaa; padding: 20px;")
        self.archiveLabel.setFont(QFont("Arial", 14))

        self.osComboBox = QComboBox(self)
        self.osComboBox.addItems(["windows", "linux", "darwin"])
        self.osComboBox.setFont(QFont("Arial", 12))
        self.osComboBox.currentIndexChanged.connect(self.onOsChanged)

        self.archComboBox = QComboBox(self)
        self.archComboBox.addItems(["amd64", "arm64", "386"])
        self.archComboBox.setFont(QFont("Arial", 12))

        self.generateButton = QPushButton("Generate", self)
        self.generateButton.clicked.connect(self.onGenerateClicked)
        self.generateButton.setFont(QFont("Arial", 12))

        central_widget = QWidget(self)
        self.setCentralWidget(central_widget)

        main_layout = QVBoxLayout(central_widget)
        main_layout.addWidget(self.archiveLabel)

        options_layout = QHBoxLayout()

        left_layout = QHBoxLayout()
        left_layout.addWidget(QLabel("OS:", self))
        left_layout.addWidget(self.osComboBox)
        left_layout.addWidget(QLabel("Arch:", self))
        left_layout.addWidget(self.archComboBox)
        left_layout.addStretch(1)

        right_layout = QHBoxLayout()
        right_layout.addStretch(1)
        right_layout.addWidget(self.generateButton)

        options_layout.addLayout(left_layout)
        options_layout.addLayout(right_layout)

        main_layout.addLayout(options_layout)

    def get_selfext_path(self):
        if sys.platform.startswith("win"):
            selfext = "selfext.exe"
        else:
            selfext = "selfext"

        if getattr(sys, "frozen", False):
            application_path = sys._MEIPASS
            source_path = os.path.join(application_path, selfext)

            if os.path.exists(source_path):
                if sys.platform.startswith("win"):
                    user_config_dir = Path(os.environ["APPDATA"])
                elif sys.platform.startswith("darwin"):
                    user_config_dir = Path.home() / "Library" / "Application Support"
                else:
                    user_config_dir = Path.home() / ".config"

                selfext_dir = user_config_dir / "selfextg"
                selfext_dir.mkdir(parents=True, exist_ok=True)
                dest_path = selfext_dir / selfext
                print("user config dir: ", user_config_dir)
                print("selfext src path: ", source_path)
                print("selfext dst path: ", dest_path)

                if not dest_path.exists():
                    print("copying ...")
                    shutil.copy2(source_path, dest_path)
                    print("copy completed")
                    if not sys.platform.startswith("win"):
                        os.chmod(dest_path, 0o755)

                return str(dest_path)

        if shutil.which(selfext):
            return selfext
        else:
            return None

    def onOsChanged(self):
        selected_os = self.osComboBox.currentText()
        if selected_os == "darwin":
            self.archComboBox.clear()
            self.archComboBox.addItems(["arm64", "amd64"])
        else:
            self.archComboBox.clear()
            self.archComboBox.addItems(["amd64", "arm64", "386"])

    def onGenerateClicked(self):
        archive_path = self.archiveLabel.text()
        selected_os = self.osComboBox.currentText()
        selected_arch = self.archComboBox.currentText()

        selfext_path = self.get_selfext_path()
        if not selfext_path:
            QMessageBox.critical(self, "Error", "selfext executable not found!")
            return

        command = [
            selfext_path,
            "--archive",
            archive_path,
            "--os",
            selected_os,
            "--arch",
            selected_arch,
        ]

        file_name = os.path.basename(archive_path)
        output_window = OutputWindow(file_name, self)
        output_window.show()

        worker = Worker(command, archive_path)
        worker.output.connect(output_window.appendOutput)
        worker.finished.connect(
            lambda return_code: self.onCommandFinished(return_code, output_window)
        )
        worker.error.connect(lambda error: self.onCommandError(error, output_window))
        worker.start()

        self.workers.append(worker)

    def onCommandFinished(self, return_code, output_window):
        output_window.setStatus(return_code == 0)
        if return_code == 0:
            file_name = self.archiveLabel.text().split("/")[-1]
            success_message = f"{file_name}.exe generated successfully!"
            output_window.appendOutput(success_message)
        else:
            output_window.appendOutput("Command execution failed.")

    def onCommandError(self, error, output_window):
        output_window.appendOutput(f"Error: {error}")
        output_window.setStatus(False)

    def closeEvent(self, event):
        for worker in self.workers:
            worker.quit()
            worker.wait()
        super().closeEvent(event)


class Worker(QThread):
    output = pyqtSignal(str)
    finished = pyqtSignal(int)
    error = pyqtSignal(str)

    def __init__(self, command, archive_path):
        super().__init__()
        self.command = command
        self.archive_path = archive_path

    def run(self):
        try:
            process = subprocess.Popen(
                self.command,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                text=True,
                bufsize=1,
                universal_newlines=True,
                creationflags=subprocess.CREATE_NO_WINDOW,
            )

            for line in process.stdout:
                self.output.emit(line.strip())

            return_code = process.wait()
            self.finished.emit(return_code)
        except Exception as e:
            self.error.emit(str(e))


if __name__ == "__main__":
    app = QApplication(sys.argv)
    mainWin = DragDropWindow()
    mainWin.show()
    sys.exit(app.exec_())
