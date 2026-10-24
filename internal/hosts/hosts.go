package hosts

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func GitHubUser(configDir string) (string, error) {
	path := filepath.Join(configDir, "hosts.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var entries map[string]struct {
		User string `yaml:"user"`
	}
	if err := yaml.Unmarshal(data, &entries); err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}

	user := entries["github.com"].User
	if user == "" {
		return "", fmt.Errorf("%s: no github.com user", path)
	}
	return user, nil
}
