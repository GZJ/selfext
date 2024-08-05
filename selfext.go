package main

import (
	"fmt"
	"io"
	log "log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	A "github.com/gzj/selfext/assets"
	"github.com/gzj/selfext/internal/version"
	flag "github.com/spf13/pflag"
)

var G Go

type Go struct {
	Bin  string
	Root string
}

func (g *Go) Build(src, dst, targetOS, targetArch string) {
	log.Debug("Starting go build", "src", src, "dst", dst, "targetOS", targetOS, "targetArch", targetArch)

	cmd := exec.Command(g.Bin, "build", "-o", dst, src)
	cmd.Env = append(cmd.Environ(), "GOOS="+targetOS, "GOARCH="+targetArch)
	cmd.Dir = filepath.Dir(src)

	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out

	log.Debug("Executing go build command", "command", cmd.String())

	err := cmd.Run()
	if err != nil {
		log.Error("go build error",
			"error", err,
			"output", out.String(),
			"exitCode", cmd.ProcessState.ExitCode())
		os.Exit(1)
	}

	log.Info("go build successful",
		"targetOS", targetOS,
		"targetArch", targetArch,
		"exitCode", cmd.ProcessState.ExitCode())
	log.Debug("go build output", "stdout", out.String())
}

func (g *Go) Run(execDir string, args []string) {
	log.Debug("Starting go run", "execDir", execDir, "args", args)

	cmd := exec.Command(g.Bin, args...)
	cmd.Dir = execDir

	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out

	log.Debug("Executing go run command", "command", cmd.String())

	err := cmd.Run()

	if err != nil {
		log.Error("go run error",
			"error", err,
			"output", out.String(),
			"exitCode", cmd.ProcessState.ExitCode())
		os.Exit(1)
	}

	log.Info("go run successful",
		"args", strings.Join(args, " "),
		"exitCode", cmd.ProcessState.ExitCode())

	log.Debug("go run output", "stdout", out.String())
}

func init() {
	userCfgDir, err := os.UserConfigDir()
	if err != nil {
		log.Error("user config error", log.Any("error", err))
	}
	dst := filepath.Join(userCfgDir, "selfext", "go")
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		A.AssetsGoExtract(filepath.Join(userCfgDir, "selfext"))
	}
	var goBin string
	if runtime.GOOS == "windows" {
		goBin = filepath.Join(userCfgDir, "selfext", "go", "go", "bin", "go.exe")
	} else {
		goBin = filepath.Join(userCfgDir, "selfext", "go", "go", "bin", "go")
	}
	goRoot := filepath.Join(userCfgDir, "selfext", "go", "go")
	G = Go{goBin, goRoot}
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	srcFileInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcFileInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to open destination file: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

func expandPath(path string) (string, error) {
	if len(path) == 0 {
		return "", fmt.Errorf("path cannot be empty")
	}

	if path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("error getting user home directory: %w", err)
		}
		if len(path) > 1 && path[1] == '/' {
			path = filepath.Join(homeDir, path[2:])
		} else {
			path = homeDir
		}
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("error getting absolute path: %w", err)
	}

	return absPath, nil
}

func main() {
	var (
		archive     string
		exeOS       string
		exeArch     string
		versionFlag bool
		verbose     bool
	)

	flag.StringVar(&archive, "archive", "", "archive file name (e.g., .zip, .tar.gz)")
	flag.StringVar(&exeOS, "os", runtime.GOOS, "exe os")
	flag.StringVar(&exeArch, "arch", runtime.GOARCH, "exe arch")
	flag.BoolVarP(&versionFlag, "version", "V", false, "print the version number and exit")
	flag.BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")
	flag.Parse()

	a := flag.Args()
	if len(a) > 0 {
		archive = a[0]
	}

	if versionFlag {
		version.Version()
		os.Exit(0)
	}

	logLevel := log.LevelInfo
	if verbose {
		logLevel = log.LevelDebug
	}
	logger := log.New(log.NewTextHandler(os.Stderr, &log.HandlerOptions{Level: logLevel}))
	log.SetDefault(logger)

	archive, _ = expandPath(archive)

	if archive != "" {
		tmpDir, err := os.MkdirTemp("", "selfext_wrapper")
		if err != nil {
			log.Error("create temp dir error...", log.Any("error", err))
			os.Exit(1)
		}
		defer os.RemoveAll(tmpDir)

		archivePath := filepath.Dir(archive)
		archiveName := filepath.Base(archive)

		wrapperPath := filepath.Join(tmpDir, "wrapper")

		A.AssetsWrapperExtract(tmpDir)

		dstPath, _ := expandPath(path.Join(wrapperPath, archiveName))
		log.Info("Copying the archive to the temp directory.", "archive name", archiveName, "temp path", dstPath)
		err = copyFile(archive, dstPath)
		if err != nil {
			log.Error("copy archive to temp dir", log.Any("error", err))
			os.Exit(1)
		}

		log.Info("Generating wrapper...")
		G.Run(wrapperPath, []string{"run", filepath.Join(wrapperPath, "wrapper_generator.go"), "--file", archiveName})

		log.Info("Building self-extracting archive...")
		G.Build(filepath.Join(wrapperPath, "wrapper.go"), filepath.Join(archivePath, archiveName+".exe"), exeOS, exeArch)
	}
}
