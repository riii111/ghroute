package route

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/riii111/ghroute/internal/config"
	"github.com/riii111/ghroute/internal/target"
)

type Kind int

const (
	Passthrough Kind = iota
	UseAccount
)

type Decision struct {
	Kind    Kind
	Reason  string
	Owner   target.Owner
	Account config.Account
}

var overridingVariables = []string{"GH_CONFIG_DIR", "GH_TOKEN", "GITHUB_TOKEN"}

func Decide(args []string, env target.Env, load func() (*config.Config, error)) (Decision, error) {
	if reason, ok := passthroughReason(args, env); ok {
		return Decision{Kind: Passthrough, Reason: reason}, nil
	}

	cfg, err := load()
	if err != nil {
		return Decision{}, err
	}
	if cfg == nil {
		return Decision{Kind: Passthrough, Reason: "no config file"}, nil
	}

	owner, found := target.Resolve(args, env)
	account, err := selectAccount(cfg, owner, found)
	if err != nil {
		return Decision{}, err
	}
	if account == nil {
		return Decision{Kind: Passthrough, Owner: owner, Reason: unmappedReason(owner, found)}, nil
	}

	if _, err := os.Stat(filepath.Join(account.ConfigDir, "hosts.yml")); err != nil {
		return Decision{}, fmt.Errorf("account %q: %w", account.Name, err)
	}
	return Decision{Kind: UseAccount, Owner: owner, Account: *account}, nil
}

func passthroughReason(args []string, env target.Env) (string, bool) {
	for _, name := range overridingVariables {
		if env.Getenv(name) != "" {
			return name + " is set", true
		}
	}
	if len(args) > 0 && args[0] == "auth" && (len(args) < 2 || args[1] != "git-credential") {
		return "gh auth command", true
	}
	return "", false
}

func selectAccount(cfg *config.Config, owner target.Owner, found bool) (*config.Account, error) {
	if found {
		if account, ok := cfg.AccountFor(owner.Name); ok {
			return &account, nil
		}
	}

	switch {
	case cfg.Default != "":
		account := cfg.Accounts[cfg.Default]
		return &account, nil
	case cfg.Strict && found:
		return nil, fmt.Errorf("strict: owner %q (from %s) is not mapped", owner.Name, owner.Source)
	case cfg.Strict:
		return nil, errors.New("strict: no repository owner found")
	default:
		return nil, nil
	}
}

func unmappedReason(owner target.Owner, found bool) string {
	if found {
		return fmt.Sprintf("owner %q is not mapped", owner.Name)
	}
	return "no repository owner found"
}
