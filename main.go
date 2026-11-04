package main

import (
	"os"
	"runtime/debug"
	"strings"

	"github.com/Eagle-Konbu/sanat/cmd"
)

// version is set by GoReleaser via ldflags. Other builds fall back to the build info.
// Release builds cannot rely on the build info alone: files generated during the
// release (e.g. sanat-dev, dist/) make the tree dirty, which turns the version into
// "vX.Y.Z+dirty".
var version = ""

func main() {
	cmd.SetVersion(resolveVersion())

	if err := cmd.Execute(); err != nil {
		os.Exit(cmd.ExitCode(err))
	}
}

func resolveVersion() string {
	if version != "" {
		return version
	}

	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}

	return "dev"
}
