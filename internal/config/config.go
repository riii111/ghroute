package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Account struct {
	Name      string
	Owners    []string
	ConfigDir string
}

type Config struct {
	Path     string
	Accounts map[string]Account
	Default  string
	Strict   bool
	byOwner  map[string]string
}

type file struct {
	Accounts map[string]struct {
		Owners    []string `yaml:"owners"`
		ConfigDir string   `yaml:"config_dir"`
	} `yaml:"accounts"`
	Default string `yaml:"default"`
	Strict  bool   `yaml:"strict"`
}

// --- Loading ---

func Path() (string, error) {
	if path := os.Getenv("GHROUTE_CONFIG"); path != "" {
		return path, nil
	}

	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "ghroute", "config.yml"), nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return Parse(path, data)
}

func Parse(path string, data []byte) (*Config, error) {
	var raw file
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&raw); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	cfg := &Config{
		Path:     path,
		Accounts: map[string]Account{},
		Default:  raw.Default,
		Strict:   raw.Strict,
		byOwner:  map[string]string{},
	}
	for name, entry := range raw.Accounts {
		if err := cfg.add(name, entry.Owners, entry.ConfigDir); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}

	if cfg.Default != "" {
		if _, ok := cfg.Accounts[cfg.Default]; !ok {
			return nil, fmt.Errorf("%s: default account %q is undefined", path, cfg.Default)
		}
	}
	return cfg, nil
}

func (c *Config) add(name string, owners []string, configDir string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("invalid account name %q", name)
	}

	dir, err := expandDir(configDir, filepath.Join(filepath.Dir(c.Path), "accounts", name))
	if err != nil {
		return fmt.Errorf("account %q: %w", name, err)
	}
	c.Accounts[name] = Account{Name: name, Owners: owners, ConfigDir: dir}

	for _, owner := range owners {
		key := strings.ToLower(owner)
		if other, ok := c.byOwner[key]; ok {
			return fmt.Errorf("owner %q is mapped to both %q and %q", owner, other, name)
		}
		c.byOwner[key] = name
	}
	return nil
}

func expandDir(dir, fallback string) (string, error) {
	switch {
	case dir == "":
		return fallback, nil
	case dir == "~" || strings.HasPrefix(dir, "~/"):
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(dir, "~")), nil
	case filepath.IsAbs(dir):
		return dir, nil
	default:
		return "", fmt.Errorf("config_dir %q must be absolute or start with ~/", dir)
	}
}

// --- Lookup ---

func (c *Config) AccountFor(owner string) (Account, bool) {
	name, ok := c.byOwner[strings.ToLower(owner)]
	if !ok {
		return Account{}, false
	}
	return c.Accounts[name], true
}

func (c *Config) SortedAccounts() []Account {
	accounts := make([]Account, 0, len(c.Accounts))
	for _, account := range c.Accounts {
		accounts = append(accounts, account)
	}
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].Name < accounts[j].Name })
	return accounts
}
