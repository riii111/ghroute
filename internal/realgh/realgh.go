package realgh

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

func Find() (string, error) {
	if path := os.Getenv("GHROUTE_GH"); path != "" {
		return path, nil
	}

	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	self = resolve(self)

	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, binaryName())
		if isExecutable(candidate) && resolve(candidate) != self {
			return candidate, nil
		}
	}
	return "", errors.New("gh not found on PATH")
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "gh.exe"
	}
	return "gh"
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

func resolve(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return path
}
