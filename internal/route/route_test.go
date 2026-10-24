package route

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/riii111/ghroute/internal/config"
	"github.com/riii111/ghroute/internal/target"
)

func TestDecide(t *testing.T) {
	tests := []struct {
		name        string
		extra       string
		args        []string
		vars        map[string]string
		removeWork  bool
		wantKind    Kind
		wantAccount string
		wantErr     string
	}{
		{name: "mapped owner", args: []string{"-R", "work-org/app"}, wantKind: UseAccount, wantAccount: "work"},
		{name: "unmapped owner", args: []string{"-R", "someone/app"}, wantKind: Passthrough},
		{name: "no owner", args: []string{"pr", "list"}, wantKind: Passthrough},
		{name: "default for unmapped owner", extra: "default: personal\n", args: []string{"-R", "someone/app"}, wantKind: UseAccount, wantAccount: "personal"},
		{name: "default for no owner", extra: "default: personal\n", args: []string{"api", "user"}, wantKind: UseAccount, wantAccount: "personal"},
		{name: "strict with unmapped owner", extra: "strict: true\n", args: []string{"-R", "someone/app"}, wantErr: "not mapped"},
		{name: "strict with no owner", extra: "strict: true\n", args: []string{"api", "user"}, wantErr: "no repository owner"},
		{name: "missing account folder", args: []string{"-R", "work-org/app"}, removeWork: true, wantErr: "hosts.yml"},
		{name: "gh auth", args: []string{"auth", "status", "-R", "work-org/app"}, wantKind: Passthrough},
		{name: "gh auth git-credential", args: []string{"auth", "git-credential", "-R", "work-org/app"}, wantKind: UseAccount, wantAccount: "work"},
		{name: "GH_TOKEN set", args: []string{"-R", "work-org/app"}, vars: map[string]string{"GH_TOKEN": "x"}, wantKind: Passthrough},
		{name: "GH_CONFIG_DIR set", args: []string{"-R", "work-org/app"}, vars: map[string]string{"GH_CONFIG_DIR": "/x"}, wantKind: Passthrough},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newConfig(t, tt.extra)
			if tt.removeWork {
				if err := os.RemoveAll(cfg.Accounts["work"].ConfigDir); err != nil {
					t.Fatal(err)
				}
			}

			got, err := Decide(tt.args, fakeEnv(tt.vars), loaded(cfg, nil))

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Kind != tt.wantKind || got.Account.Name != tt.wantAccount {
				t.Errorf("Decide = %v %q, want %v %q", got.Kind, got.Account.Name, tt.wantKind, tt.wantAccount)
			}
		})
	}
}

func TestDecideWithUnusableConfig(t *testing.T) {
	loadErr := errors.New("broken config")
	tests := []struct {
		name    string
		args    []string
		vars    map[string]string
		loadErr error
		wantErr bool
	}{
		{name: "no config file", args: []string{"-R", "work-org/app"}},
		{name: "broken config with GH_TOKEN", args: []string{"-R", "work-org/app"}, vars: map[string]string{"GH_TOKEN": "x"}, loadErr: loadErr},
		{name: "broken config with GH_CONFIG_DIR", args: []string{"auth", "login"}, vars: map[string]string{"GH_CONFIG_DIR": "/x"}, loadErr: loadErr},
		{name: "broken config with gh auth", args: []string{"auth", "status"}, loadErr: loadErr},
		{name: "broken config when routing", args: []string{"-R", "work-org/app"}, loadErr: loadErr, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			load := loaded(nil, tt.loadErr)

			got, err := Decide(tt.args, fakeEnv(tt.vars), load)

			if tt.wantErr {
				if !errors.Is(err, loadErr) {
					t.Fatalf("err = %v, want %v", err, loadErr)
				}
				return
			}
			if err != nil || got.Kind != Passthrough {
				t.Fatalf("Decide = %+v, %v", got, err)
			}
		})
	}
}

// --- Helpers ---

func loaded(cfg *config.Config, err error) func() (*config.Config, error) {
	return func() (*config.Config, error) { return cfg, err }
}

func newConfig(t *testing.T, extra string) *config.Config {
	t.Helper()
	dir := t.TempDir()
	for _, account := range []string{"personal", "work"} {
		folder := filepath.Join(dir, "accounts", account)
		if err := os.MkdirAll(folder, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, "hosts.yml"), []byte("github.com:\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	data := "accounts:\n  personal:\n    owners: [me]\n  work:\n    owners: [Work-Org]\n" + extra
	cfg, err := config.Parse(filepath.Join(dir, "config.yml"), []byte(data))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func fakeEnv(vars map[string]string) target.Env {
	return target.Env{
		Getenv: func(name string) string { return vars[name] },
		Dir:    "/tmp",
		Origin: func(string) (string, bool) { return "", false },
	}
}
