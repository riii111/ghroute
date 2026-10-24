package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/riii111/ghroute/internal/config"
	"github.com/riii111/ghroute/internal/hosts"
	"github.com/riii111/ghroute/internal/realgh"
	"github.com/riii111/ghroute/internal/route"
	"github.com/riii111/ghroute/internal/target"
)

var Version = "dev"

const usage = `ghroute runs gh as the account mapped to the target repository's owner.

Install it as an executable named gh that comes before the real gh on PATH.

Usage:
  ghroute which [--] <gh arguments>   show the account for a gh command
  ghroute doctor                      verify each account's token
  ghroute version
`

var tokenVariables = []string{"GH_CONFIG_DIR", "GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"}

// --- Entry point ---

func Main(argv []string, stdout, stderr io.Writer) int {
	if strings.TrimSuffix(filepath.Base(argv[0]), ".exe") == "gh" {
		return runGh(argv[1:], stderr)
	}

	if len(argv) < 2 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch argv[1] {
	case "which":
		return which(trimDoubleDash(argv[2:]), stdout, stderr)
	case "doctor":
		return doctor(stdout, stderr)
	case "version", "--version":
		fmt.Fprintln(stdout, "ghroute", Version)
		return 0
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "ghroute: unknown command %q\n\n%s", argv[1], usage)
		return 2
	}
}

func trimDoubleDash(args []string) []string {
	if len(args) > 0 && args[0] == "--" {
		return args[1:]
	}
	return args
}

// --- gh stand-in ---

func runGh(args []string, stderr io.Writer) int {
	gh, err := realgh.Find()
	if err != nil {
		fmt.Fprintln(stderr, "ghroute:", err)
		return 127
	}

	decision, err := decide(args)
	if err != nil {
		fmt.Fprintln(stderr, "ghroute:", err)
		return 1
	}

	env := os.Environ()
	if decision.Kind == route.UseAccount {
		env = append(env, "GH_CONFIG_DIR="+decision.Account.ConfigDir)
	}
	return execGh(gh, args, env, stderr)
}

func decide(args []string) (route.Decision, error) {
	cfg, _, err := loadConfig()
	if err != nil {
		return route.Decision{}, err
	}
	return route.Decide(args, cfg, target.ProcessEnv())
}

func loadConfig() (*config.Config, string, error) {
	path, err := config.Path()
	if err != nil {
		return nil, "", err
	}
	cfg, err := config.Load(path)
	return cfg, path, err
}

// --- which ---

func which(args []string, stdout, stderr io.Writer) int {
	decision, err := decide(args)
	if err != nil {
		fmt.Fprintln(stderr, "ghroute:", err)
		return 1
	}

	if decision.Owner.Name != "" {
		fmt.Fprintf(stdout, "owner:   %s (from %s)\n", decision.Owner.Name, decision.Owner.Source)
	}
	if decision.Kind == route.Passthrough {
		fmt.Fprintf(stdout, "account: active gh account (%s)\n", decision.Reason)
		return 0
	}
	fmt.Fprintf(stdout, "account: %s\n", decision.Account.Name)
	fmt.Fprintf(stdout, "config:  %s\n", decision.Account.ConfigDir)
	return 0
}

// --- doctor ---

func doctor(stdout, stderr io.Writer) int {
	gh, err := realgh.Find()
	if err != nil {
		fmt.Fprintln(stderr, "ghroute:", err)
		return 1
	}
	cfg, path, err := loadConfig()
	if err != nil {
		fmt.Fprintln(stderr, "ghroute:", err)
		return 1
	}

	fmt.Fprintf(stdout, "gh:     %s\n", gh)
	if cfg == nil {
		fmt.Fprintf(stdout, "config: %s (missing)\n", path)
		return 1
	}
	fmt.Fprintf(stdout, "config: %s\n", path)

	code := 0
	for _, account := range cfg.SortedAccounts() {
		if problem := checkAccount(gh, account); problem != "" {
			code = 1
			fmt.Fprintf(stdout, "FAIL  %s: %s\n", account.Name, problem)
			continue
		}
		fmt.Fprintf(stdout, "ok    %s: %s\n", account.Name, account.ConfigDir)
	}
	return code
}

func checkAccount(gh string, account config.Account) string {
	user, err := hosts.GitHubUser(account.ConfigDir)
	if err != nil {
		return err.Error()
	}
	login, err := authenticatedLogin(gh, account.ConfigDir)
	if err != nil {
		return err.Error()
	}

	if !strings.EqualFold(login, user) {
		return fmt.Sprintf("token belongs to %s, not %s", login, user)
	}
	return ""
}

func authenticatedLogin(gh, configDir string) (string, error) {
	cmd := exec.Command(gh, "api", "user", "--jq", ".login")
	cmd.Env = append(withoutVariables(os.Environ(), tokenVariables), "GH_CONFIG_DIR="+configDir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh api user: %s", firstLine(stderr.String(), err.Error()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func withoutVariables(env, names []string) []string {
	kept := make([]string, 0, len(env))
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		if !slices.Contains(names, key) {
			kept = append(kept, entry)
		}
	}
	return kept
}

func firstLine(text, fallback string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return fallback
	}
	line, _, _ := strings.Cut(text, "\n")
	return line
}
