package target

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Source string

const (
	SourceRepoFlag Source = "--repo"
	SourceGHRepo   Source = "GH_REPO"
	SourceURL      Source = "URL argument"
	SourceOrigin   Source = "git remote origin"
	SourceGhq      Source = "ghq path"
)

type Owner struct {
	Name   string
	Source Source
}

type Env struct {
	Getenv func(string) string
	Dir    string
	Home   string
	Origin func(dir string) (string, bool)
}

var githubURL = regexp.MustCompile(`(?i)^https://(?:www\.)?github\.com/([^/\s]+)(?:/\S*)?$`)

// --- Owner resolution ---

func Resolve(args []string, env Env) (Owner, bool) {
	repo, urlOwner := scanArgs(args)

	if repo != "" {
		return ownerOrNone(OwnerOfLocation(repo), SourceRepoFlag)
	}
	if repo := env.Getenv("GH_REPO"); repo != "" {
		return ownerOrNone(OwnerOfLocation(repo), SourceGHRepo)
	}
	if urlOwner != "" {
		return Owner{Name: urlOwner, Source: SourceURL}, true
	}

	if origin, ok := env.Origin(env.Dir); ok {
		return ownerOrNone(OwnerOfLocation(origin), SourceOrigin)
	}
	if owner := ghqOwner(env); owner != "" {
		return Owner{Name: owner, Source: SourceGhq}, true
	}
	return Owner{}, false
}

func scanArgs(args []string) (repo, urlOwner string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-R" || arg == "--repo":
			i++
			repo = ""
			if i < len(args) {
				repo = args[i]
			}
		case strings.HasPrefix(arg, "--repo="):
			repo = strings.TrimPrefix(arg, "--repo=")
		case strings.HasPrefix(arg, "-R"):
			repo = strings.TrimPrefix(strings.TrimPrefix(arg, "-R"), "=")
		case urlOwner == "":
			if m := githubURL.FindStringSubmatch(arg); m != nil {
				urlOwner = m[1]
			}
		}
	}
	return repo, urlOwner
}

func OwnerOfLocation(value string) string {
	if _, rest, ok := strings.Cut(value, "://"); ok {
		value = rest
	} else {
		value = strings.Replace(value, ":", "/", 1)
	}
	if _, rest, ok := strings.Cut(value, "@"); ok {
		value = rest
	}

	var parts []string
	for _, part := range strings.Split(value, "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}

	switch len(parts) {
	case 0, 1:
		return ""
	case 2:
		return parts[0]
	default:
		return parts[1]
	}
}

func ownerOrNone(name string, source Source) (Owner, bool) {
	if name == "" {
		return Owner{}, false
	}
	return Owner{Name: name, Source: source}, true
}

func ghqOwner(env Env) string {
	root := env.Getenv("GHQ_ROOT")
	if root == "" {
		if env.Home == "" {
			return ""
		}
		root = filepath.Join(env.Home, "ghq")
	}

	prefix := filepath.Join(root, "github.com") + string(filepath.Separator)
	rest, ok := strings.CutPrefix(env.Dir+string(filepath.Separator), prefix)
	if !ok {
		return ""
	}
	owner, _, _ := strings.Cut(rest, string(filepath.Separator))
	return owner
}

// --- Process environment ---

func ProcessEnv() Env {
	dir, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	return Env{Getenv: os.Getenv, Dir: dir, Home: home, Origin: gitOrigin}
}

func gitOrigin(dir string) (string, bool) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}
