# ghroute

ghroute runs [GitHub CLI](https://cli.github.com/) as the account mapped to the owner of the target repository.

It sits in front of `gh` on `PATH` and starts the real `gh` with a per-account `GH_CONFIG_DIR`. Unlike `gh auth switch`, it changes no global state, so parallel `gh` calls, such as from several AI coding agents, never switch each other's account. It needs no shell hook, so non-interactive shells work too.

## Install

```sh
go install github.com/riii111/ghroute@latest
ln -s "$(go env GOPATH)/bin/ghroute" ~/.local/bin/gh
```

Put the `gh` symlink before the real `gh` on `PATH`. ghroute skips itself and runs the next `gh`. Set `GHROUTE_GH` to choose the `gh` binary.

macOS and Linux are supported.

## Setup

Log in to each account in its own config directory:

```sh
GH_CONFIG_DIR=~/.config/ghroute/accounts/personal gh auth login
GH_CONFIG_DIR=~/.config/ghroute/accounts/work gh auth login
```

Map owners to accounts in `~/.config/ghroute/config.yml`:

```yaml
accounts:
  personal:
    owners: [your-login]
  work:
    owners: [acme-corp, acme-labs]
    # config_dir: ~/.config/gh-work   # defaults to accounts/<name> next to this file

# default: personal   # account for unmapped owners and commands without a repository
# strict: true        # stop instead of using gh's active account when nothing applies
```

Owner names are case-insensitive. Set `GHROUTE_CONFIG` to read the config from another path.

Check that each account's token belongs to the user in its `hosts.yml`:

```sh
ghroute doctor
```

## How the account is chosen

The owner comes from the first of these that applies:

1. `-R` / `--repo` (the last one wins)
2. `GH_REPO`
3. A whole argument that is a `https://github.com/OWNER/...` URL
4. The `origin` remote of the current directory
5. The current directory under `$GHQ_ROOT/github.com/OWNER/` (default `~/ghq`)

Then:

| Situation | Result |
|---|---|
| The owner is mapped | Runs as the mapped account |
| The mapped account has no `hosts.yml` | Stops with an error |
| No owner, or an unmapped owner, with `default` set | Runs as the default account |
| Same, with `strict: true` | Stops with an error |
| Same, otherwise | Runs as gh's active account |

`gh` runs unchanged when `GH_CONFIG_DIR`, `GH_TOKEN` or `GITHUB_TOKEN` is already set, for `gh auth` commands other than `gh auth git-credential`, and when there is no config file.

To see what a command would do:

```console
$ ghroute which pr view 123 -R acme-corp/api
owner:   acme-corp (from --repo)
account: work
config:  /Users/you/.config/ghroute/accounts/work
```

## License

MIT
