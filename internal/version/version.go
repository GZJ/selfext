package version

import (
	"fmt"
	"runtime"
)

var (
	gitVersion       = ""
	gitCommit        = ""
	gitBranch        = ""
	buildDate        = ""
	goVersionEmbeded = ""
)

type VersionInfo struct {
	GitVersion       string `json:"gitVersion"`
	GitCommit        string `json:"gitCommit"`
	GitBranch        string `json:"gitBranch"`
	BuildDate        string `json:"buildDate"`
	GoVersion        string `json:"goVersion"`
	GoVersionEmbeded string `json:"goVersionEmbeded"`
	Compiler         string `json:"compiler"`
	Platform         string `json:"platform"`
}

func (v *VersionInfo) String() string {
	return fmt.Sprintf(
		"GitVersion: %s\nGitCommit: %s\nGitBranch: %s\nBuildDate: %s\nGoVersion: %s\nGoVersionEmbeded: %s\nCompiler: %s\nPlatform: %s",
		v.GitVersion, v.GitCommit, v.GitBranch, v.BuildDate, v.GoVersion, v.GoVersionEmbeded, v.Compiler, v.Platform)
}

func Get() *VersionInfo {
	return &VersionInfo{
		GitVersion:       gitVersion,
		GitCommit:        gitCommit,
		GitBranch:        gitBranch,
		BuildDate:        buildDate,
		GoVersion:        runtime.Version(),
		GoVersionEmbeded: goVersionEmbeded,
		Compiler:         runtime.Compiler,
		Platform:         fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

func Version() {
	fmt.Println(Get().String())
}
