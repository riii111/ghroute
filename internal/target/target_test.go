package target

import (
	"path/filepath"
	"testing"
)

func TestOwnerOfLocation(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "owner/repo", value: "work-org/app", want: "work-org"},
		{name: "host/owner/repo", value: "github.com/work-org/app", want: "work-org"},
		{name: "https URL", value: "https://github.com/work-org/app", want: "work-org"},
		{name: "https URL with user", value: "https://user@github.com/me/x.git", want: "me"},
		{name: "ssh URL", value: "ssh://git@github.com/work-org/app", want: "work-org"},
		{name: "scp-like remote", value: "git@github.com:work-org/app.git", want: "work-org"},
		{name: "scp-like remote with host alias", value: "git@github.com-alias:me/dotfiles", want: "me"},
		{name: "repo only", value: "app", want: ""},
		{name: "empty", value: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OwnerOfLocation(tt.value)

			if got != tt.want {
				t.Errorf("OwnerOfLocation(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	const url = "https://github.com/me/dotfiles/issues/1"
	tests := []struct {
		name      string
		args      []string
		vars      map[string]string
		dir       string
		origin    string
		want      Owner
		wantFound bool
	}{
		{name: "-R", args: []string{"pr", "view", "1", "-R", "work-org/app"}, want: Owner{"work-org", SourceRepoFlag}, wantFound: true},
		{name: "-R=", args: []string{"pr", "view", "-R=work-org/app"}, want: Owner{"work-org", SourceRepoFlag}, wantFound: true},
		{name: "-Rvalue", args: []string{"pr", "view", "-RWork-Org/app"}, want: Owner{"Work-Org", SourceRepoFlag}, wantFound: true},
		{name: "--repo with host", args: []string{"--repo", "github.com/work-org/app"}, want: Owner{"work-org", SourceRepoFlag}, wantFound: true},
		{name: "--repo=", args: []string{"--repo=me/dotfiles"}, want: Owner{"me", SourceRepoFlag}, wantFound: true},
		{name: "last -R wins", args: []string{"-R", "me/a", "-R", "work-org/a"}, want: Owner{"work-org", SourceRepoFlag}, wantFound: true},
		{name: "GH_REPO", args: []string{"pr", "view"}, vars: map[string]string{"GH_REPO": "work-org/app"}, want: Owner{"work-org", SourceGHRepo}, wantFound: true},
		{name: "flag beats URL and origin", args: []string{"pr", "view", url, "-R", "work-org/app"}, origin: "git@github.com:me/x", want: Owner{"work-org", SourceRepoFlag}, wantFound: true},
		{name: "URL beats origin", args: []string{"pr", "checkout", "-f", url}, origin: "git@github.com:work-org/app", want: Owner{"me", SourceURL}, wantFound: true},
		{name: "www URL", args: []string{"pr", "view", "https://www.github.com/me/app/pull/1"}, want: Owner{"me", SourceURL}, wantFound: true},
		{name: "URL inside text is ignored", args: []string{"issue", "close", "1", "--comment", "see " + url}, origin: "git@github.com:work-org/app", want: Owner{"work-org", SourceOrigin}, wantFound: true},
		{name: "origin", args: []string{"pr", "list"}, origin: "ssh://git@github.com/work-org/app.git", want: Owner{"work-org", SourceOrigin}, wantFound: true},
		{name: "ghq path", args: []string{"pr", "list"}, dir: filepath.FromSlash("/home/me/ghq/github.com/work-org/app/sub"), want: Owner{"work-org", SourceGhq}, wantFound: true},
		{name: "GHQ_ROOT", args: []string{"pr", "list"}, vars: map[string]string{"GHQ_ROOT": "/src"}, dir: filepath.FromSlash("/src/github.com/me/app"), want: Owner{"me", SourceGhq}, wantFound: true},
		{name: "ghq host root", args: []string{"pr", "list"}, dir: filepath.FromSlash("/home/me/ghq/github.com")},
		{name: "nothing", args: []string{"pr", "list"}, dir: "/tmp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := fakeEnv(tt.vars, tt.dir, tt.origin)

			got, found := Resolve(tt.args, env)

			if got != tt.want || found != tt.wantFound {
				t.Errorf("Resolve = %+v, %v; want %+v, %v", got, found, tt.want, tt.wantFound)
			}
		})
	}
}

// --- Helpers ---

func fakeEnv(vars map[string]string, dir, origin string) Env {
	return Env{
		Getenv: func(name string) string { return vars[name] },
		Dir:    dir,
		Home:   "/home/me",
		Origin: func(string) (string, bool) { return origin, origin != "" },
	}
}
