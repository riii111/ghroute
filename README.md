# ghroute

Use the right GitHub account for every `gh` command, automatically.

ghroute picks the account from the repository each command targets. Your company's repositories run as your work account, and your own run as your personal account. There is no `gh auth switch`, so terminals and AI coding agents working in parallel never switch each other's account.

## Install

```sh
go install github.com/riii111/ghroute@latest
ln -s "$(go env GOPATH)/bin/ghroute" ~/.local/bin/gh
```

Make sure `~/.local/bin` comes before the real `gh` on your `PATH`. ghroute supports macOS and Linux.

## Set up

Log in once per account:

```sh
GH_CONFIG_DIR=~/.config/ghroute/accounts/personal gh auth login
GH_CONFIG_DIR=~/.config/ghroute/accounts/work gh auth login
```

List which owners each account handles in `~/.config/ghroute/config.yml`:

```yaml
accounts:
  personal:
    owners: [your-login]
  work:
    owners: [acme-corp]
```

Check that each account logs in as the right user:

```sh
ghroute doctor
```

## Use

Run `gh` as usual. ghroute finds the repository from `-R`, a GitHub URL in the arguments, or the current directory.

To see which account a command will use:

```console
$ ghroute which pr view 123 -R acme-corp/api
owner:   acme-corp (from --repo)
account: work
config:  /Users/you/.config/ghroute/accounts/work
```

When no account matches, `gh` runs as your current `gh` account. To change that, add one of these to the config:

- `default: personal` runs unmatched commands as that account.
- `strict: true` stops unmatched commands instead.

## License

MIT
