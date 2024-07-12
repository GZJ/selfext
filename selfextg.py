import sys
import subprocess
from functools import partial
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
)
from PyQt5.QtCore import Qt, QThread, pyqtSignal
from PyQt5.QtGui import QDropEvent, QDragEnterEvent, QFont


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


class DragDropWindow(QMainWindow):
    def __init__(self):
        super().__init__()
        self.initUI()

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

        command = [
            "selfext",
            "--archive",
            archive_path,
            "--os",
            selected_os,
            "--arch",
            selected_arch,
        ]

        self.worker = Worker(command, archive_path)
        self.worker.finished.connect(
            partial(self.onCommandFinished, archive_path=archive_path)
        )
        self.worker.error.connect(
            partial(self.onCommandError, archive_path=archive_path)
        )
        self.worker.start()

    def onCommandFinished(self, output, archive_path):
        file_name = archive_path.split("/")[-1]
        success_message = f"{file_name}.exe generated successfully!"
        print(output)
        QMessageBox.information(self, "Execution Result", success_message)

    def onCommandError(self, error, archive_path):
        file_name = archive_path.split("/")[-1]
        error_message = f"Failed to generate {file_name}.exe!"
        print(f"Error executing command: {error}")
        QMessageBox.critical(self, "Execution Result", error_message)


class Worker(QThread):
    finished = pyqtSignal(str)
    error = pyqtSignal(str)

    def __init__(self, command, archive_path):
        super().__init__()
        self.command = command
        self.archive_path = archive_path

    def run(self):
        try:
            result = subprocess.run(
                self.command, capture_output=True, text=True, check=True
            )
            self.finished.emit(result.stdout)
        except subprocess.CalledProcessError as e:
            self.error.emit(e.stderr)


if __name__ == "__main__":
    app = QApplication(sys.argv)
    mainWin = DragDropWindow()
    mainWin.show()
    sys.exit(app.exec_())
