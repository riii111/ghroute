//go:build unix

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

var binary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "ghroute-bin")
	if err != nil {
		panic(err)
	}
	binary = filepath.Join(dir, "ghroute")
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func TestGh(t *testing.T) {
	tests := []struct {
		name          string
		origin        string
		ghqPath       string
		args          []string
		env           []string
		removeAccount string
		wantAccount   string
		wantConfigDir string
		wantErr       string
	}{
		{name: "scp-like origin", origin: "git@github.com:work-org/app.git", args: []string{"pr", "list"}, wantAccount: "work"},
		{name: "origin with host alias", origin: "git@github.com-alias:me/dotfiles.git", args: []string{"pr", "list"}, wantAccount: "personal"},
		{name: "https origin", origin: "https://github.com/me/dotfiles.git", args: []string{"pr", "list"}, wantAccount: "personal"},
		{name: "unmapped origin", origin: "https://github.com/someone/app.git", args: []string{"pr", "list"}},
		{name: "-R beats origin", origin: "git@github.com:me/dotfiles.git", args: []string{"pr", "view", "1", "-R", "work-org/app"}, wantAccount: "work"},
		{name: "ghq path", ghqPath: "work-org/app/sub", args: []string{"pr", "list"}, wantAccount: "work"},
		{name: "gh auth", origin: "git@github.com:work-org/app.git", args: []string{"auth", "status"}},
		{name: "GH_TOKEN set", origin: "git@github.com:work-org/app.git", args: []string{"pr", "list"}, env: []string{"GH_TOKEN=x"}},
		{name: "outer GH_CONFIG_DIR kept", origin: "git@github.com:work-org/app.git", args: []string{"auth", "git-credential", "get"}, env: []string{"GH_CONFIG_DIR=/outer"}, wantConfigDir: "/outer"},
		{name: "missing account folder", args: []string{"pr", "list", "-R", "work-org/app"}, removeAccount: "work", wantErr: "hosts.yml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			dir := f.home
			switch {
			case tt.origin != "":
				dir = f.repository(tt.origin)
			case tt.ghqPath != "":
				dir = filepath.Join(f.home, "ghq/github.com", tt.ghqPath)
				f.mkdir(dir)
			}
			if tt.removeAccount != "" {
				if err := os.RemoveAll(f.accountDir(tt.removeAccount)); err != nil {
					t.Fatal(err)
				}
			}

			out, err := f.command(dir, tt.env, "gh", tt.args...).CombinedOutput()

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(string(out), tt.wantErr) {
					t.Fatalf("err = %v, output = %s", err, out)
				}
				if f.loggedLines() != nil {
					t.Error("real gh ran despite the error")
				}
				return
			}
			if err != nil {
				t.Fatalf("gh %v: %v\n%s", tt.args, err, out)
			}

			wantDir := tt.wantConfigDir
			if wantDir == "" {
				wantDir = f.configDirOf(tt.wantAccount)
			}
			want := []string{wantDir + " " + strings.Join(tt.args, " ")}
			if got := f.loggedLines(); !slices.Equal(got, want) {
				t.Errorf("log = %q, want %q", got, want)
			}
		})
	}
}

func TestGhConcurrentCallsDoNotAffectEachOther(t *testing.T) {
	f := newFixture(t)
	watch := []string{"pr", "checks", "1", "--watch", "-R", "me/dotfiles"}
	watcher := f.command(f.home, nil, "gh", watch...)
	if err := watcher.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = watcher.Process.Kill() })

	other := []string{"pr", "list", "-R", "work-org/app"}
	if out, err := f.command(f.home, nil, "gh", other...).CombinedOutput(); err != nil {
		t.Fatalf("gh %v: %v\n%s", other, err, out)
	}
	done := make(chan error, 1)
	go func() { done <- watcher.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("watcher did not finish")
	}

	want := []string{
		f.configDirOf("work") + " " + strings.Join(other, " "),
		f.configDirOf("personal") + " " + strings.Join(watch, " "),
	}
	if got := f.loggedLines(); !slices.Equal(got, want) {
		t.Errorf("log = %q, want %q", got, want)
	}
}

func TestWhich(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want func(f *fixture) string
	}{
		{
			name: "mapped owner",
			args: []string{"which", "--", "pr", "view", "-R", "work-org/app"},
			want: func(f *fixture) string {
				return fmt.Sprintf("owner:   work-org (from --repo)\naccount: work\nconfig:  %s\n", f.accountDir("work"))
			},
		},
		{
			name: "unmapped owner",
			args: []string{"which", "pr", "view", "-R", "someone/app"},
			want: func(*fixture) string {
				return "owner:   someone (from --repo)\naccount: active gh account (owner \"someone\" is not mapped)\n"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)

			out, err := f.command(f.home, nil, "ghroute", tt.args...).Output()

			if err != nil {
				t.Fatal(err)
			}
			if want := tt.want(f); string(out) != want {
				t.Errorf("which = %q, want %q", out, want)
			}
		})
	}
}

func TestDoctor(t *testing.T) {
	tests := []struct {
		name       string
		env        []string
		workLogin  string
		wantFailed bool
		wantOutput string
	}{
		{name: "all accounts match", env: []string{"GH_TOKEN=outer"}, workLogin: "me-work", wantOutput: "ok    work:"},
		{name: "token of another user", workLogin: "me", wantFailed: true, wantOutput: "FAIL  work: token belongs to me, not me-work"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			f.write(filepath.Join(f.accountDir("work"), "login"), 0o600, tt.workLogin+"\n")

			out, err := f.command(f.home, tt.env, "ghroute", "doctor").CombinedOutput()

			if (err != nil) != tt.wantFailed || !strings.Contains(string(out), tt.wantOutput) {
				t.Errorf("err = %v, output = %s", err, out)
			}
		})
	}
}

// --- Fixture ---

type fixture struct {
	t    *testing.T
	root string
	home string
	log  string
	path string
}

const fakeGh = `#!/bin/sh
case "$*" in
  *--watch*) sleep 2 ;;
  "api user --jq .login") cat "$GH_CONFIG_DIR/login"; exit ;;
esac
printf '%%s %%s\n' "${GH_CONFIG_DIR:--}" "$*" >> %q
`

const fixtureConfig = `accounts:
  personal:
    owners: [me]
  work:
    owners: [work-org]
`

func newFixture(t *testing.T) *fixture {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	f := &fixture{t: t, root: root, home: filepath.Join(root, "home"), log: filepath.Join(root, "gh.log")}

	wrapperDir, realDir := filepath.Join(root, "wrapper"), filepath.Join(root, "real")
	f.mkdir(wrapperDir)
	if err := os.Symlink(binary, filepath.Join(wrapperDir, "gh")); err != nil {
		t.Fatal(err)
	}
	f.write(filepath.Join(realDir, "gh"), 0o755, fmt.Sprintf(fakeGh, f.log))
	f.path = strings.Join([]string{wrapperDir, realDir, "/usr/bin", "/bin"}, string(os.PathListSeparator))

	for account, login := range map[string]string{"personal": "me", "work": "me-work"} {
		dir := f.accountDir(account)
		f.write(filepath.Join(dir, "hosts.yml"), 0o600, "github.com:\n    user: "+login+"\n")
		f.write(filepath.Join(dir, "login"), 0o600, login+"\n")
	}
	f.write(filepath.Join(f.home, ".config/ghroute/config.yml"), 0o600, fixtureConfig)
	return f
}

func (f *fixture) command(dir string, env []string, name string, args ...string) *exec.Cmd {
	path := binary
	if name == "gh" {
		path = filepath.Join(f.root, "wrapper", "gh")
	}
	cmd := exec.Command(path, args...)
	cmd.Dir = dir
	cmd.Env = append([]string{
		"HOME=" + f.home,
		"PATH=" + f.path,
		"GIT_CEILING_DIRECTORIES=" + f.root,
	}, env...)
	return cmd
}

func (f *fixture) repository(origin string) string {
	f.t.Helper()
	repo, err := os.MkdirTemp(f.root, "repo")
	if err != nil {
		f.t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q", repo}, {"-C", repo, "remote", "add", "origin", origin}} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			f.t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return repo
}

func (f *fixture) accountDir(account string) string {
	return filepath.Join(f.home, ".config/ghroute/accounts", account)
}

func (f *fixture) configDirOf(account string) string {
	if account == "" {
		return "-"
	}
	return f.accountDir(account)
}

func (f *fixture) loggedLines() []string {
	data, err := os.ReadFile(f.log)
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

func (f *fixture) mkdir(dir string) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) write(path string, mode os.FileMode, content string) {
	f.mkdir(filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		f.t.Fatal(err)
	}
}
