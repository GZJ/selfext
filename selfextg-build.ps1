cp target/selfext_windows-amd64.exe target/selfext.exe
pyinstaller.exe  --onefile selfextg.py --add-binary "target/selfext.exe:." --noconsole --distpath ./target
