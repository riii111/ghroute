package config

import (
	"path/filepath"
	"strings"
	"testing"
)

const validConfig = `
accounts:
  personal:
    owners: [me]
  work:
    owners: [Work-Org, other-org]
    config_dir: /srv/gh/work
default: personal
`

func TestParseResolvesAccounts(t *testing.T) {
	tests := []struct {
		name      string
		owner     string
		wantFound bool
		wantDir   string
	}{
		{name: "default config dir", owner: "me", wantFound: true, wantDir: filepath.FromSlash("/cfg/ghroute/accounts/personal")},
		{name: "explicit config_dir", owner: "other-org", wantFound: true, wantDir: "/srv/gh/work"},
		{name: "case-insensitive owner", owner: "work-org", wantFound: true, wantDir: "/srv/gh/work"},
		{name: "unmapped owner", owner: "someone", wantFound: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Parse(filepath.FromSlash("/cfg/ghroute/config.yml"), []byte(validConfig))
			if err != nil {
				t.Fatal(err)
			}

			account, found := cfg.AccountFor(tt.owner)

			if found != tt.wantFound || account.ConfigDir != tt.wantDir {
				t.Errorf("AccountFor(%q) = %q, %v; want %q, %v", tt.owner, account.ConfigDir, found, tt.wantDir, tt.wantFound)
			}
		})
	}
}

func TestParseRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{name: "duplicate owner", data: "accounts:\n  a:\n    owners: [x]\n  b:\n    owners: [X]\n", want: "mapped to both"},
		{name: "undefined default", data: "accounts:\n  a:\n    owners: [x]\ndefault: b\n", want: "undefined"},
		{name: "unknown field", data: "accounts:\n  a:\n    owner: [x]\n", want: "not found"},
		{name: "relative config_dir", data: "accounts:\n  a:\n    config_dir: rel/dir\n", want: "must be absolute"},
		{name: "path in account name", data: "accounts:\n  ../a:\n    owners: [x]\n", want: "invalid account name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse("config.yml", []byte(tt.data))

			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestParseAcceptsEmptyFile(t *testing.T) {
	_, err := Parse("config.yml", nil)

	if err != nil {
		t.Fatal(err)
	}
}
