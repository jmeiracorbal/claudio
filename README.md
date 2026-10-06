<h1 align="center">
  <img src="assets/gallo-claudio.svg" alt="Gallo Claudio, mascota de claudio" width="100" align="absmiddle">
  claudio
</h1>

[![Release](https://img.shields.io/github/v/release/jmeiracorbal/claudio)](https://github.com/jmeiracorbal/claudio/releases/latest)
[![Go](https://img.shields.io/badge/go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/github/license/jmeiracorbal/claudio)](LICENSE)
[![macOS](https://img.shields.io/badge/macOS-arm64%20%7C%20amd64-000000?logo=apple)](https://github.com/jmeiracorbal/claudio/releases/latest)
[![Linux](https://img.shields.io/badge/Linux-arm64%20%7C%20amd64-FCC624?logo=linux&logoColor=black)](https://github.com/jmeiracorbal/claudio/releases/latest)

Multiple Claude Code accounts on the same machine.

```bash
claudio create work
claudio login work
claudio work
```

## Install

```bash
curl -sSfL https://raw.githubusercontent.com/jmeiracorbal/claudio/main/install.sh | sh
```

Build from source:

```bash
git clone https://github.com/jmeiracorbal/claudio
cd claudio
go build -o claudio .
sudo mv claudio /usr/local/bin/
```

Requires Go 1.26+ when building from source, and [Claude Code](https://claude.ai/code) in PATH.

## Quick start

```bash
claudio create personal
claudio create work

claudio login personal
claudio login work

claudio personal
claudio work
```

## Commands

| Command | Description |
|---|---|
| `claudio create <name>` | Create a new profile |
| `claudio create <name> --isolated` | Create a profile with its own settings and hooks |
| `claudio migrate <name>` | Migrate your existing `~/.claude` account into a claudio profile |
| `claudio restore` | Restore the origin account back to `~/.claude` |
| `claudio login <name>` | Open Claude with the given profile to authenticate |
| `claudio <name> [-- args...]` | Launch Claude with that profile |
| `claudio switch` | Interactive profile selector |
| `claudio list` | List all profiles |
| `claudio current` | Show which profile applies to the current directory |
| `claudio pin <name>` | Pin a profile to the current directory |
| `claudio pin unpin` | Remove the pin |
| `claudio rename <old> <new>` | Rename a profile |
| `claudio remove <name>` | Delete a profile |
| `claudio doctor` | Check the setup |
| `claudio manage` | Open the interactive Task Hub |

## Interactive manager

Run `claudio manage` to open the full-screen Task Hub. It includes:

- **Launch Claude:** choose a profile and start a Claude session.
- **Profile Library:** create shared or isolated profiles, log in, rename, launch, and remove profiles.
- **Project Routing:** see the profile resolved for the current directory, pin or unpin a project, and add or remove path rules.
- **Check setup:** inspect Claude availability, configuration, profile directories, and the active route.

Use the arrow keys and Enter to navigate. Press `?` for the key guide. Existing commands such as `claudio create`, `claudio switch`, and `claudio <profile>` remain available.

For a static illustrative frame (with sample profile names and paths):

```bash
claudio manage --frame --screen home --cols 100 --rows 30
```

## Migrating an existing account

If you already have a Claude Code account in `~/.claude` and want to bring it under claudio:

```bash
claudio migrate personal
```

This copies `~/.claude` into `~/.claudio/profiles/personal/claude/` and marks it as the **origin account**. The original `~/.claude` is left untouched. From that point, use `claudio personal` instead of `claude` directly.

To migrate from a non-standard location:

```bash
claudio migrate work --from-external-path=/path/to/.claude
```

To reverse the migration and restore the account back to `~/.claude`:

```bash
claudio restore
```

If `~/.claude` already exists it is backed up as `~/.claude.bak.<timestamp>` before being replaced. To restore a specific profile instead of the origin account:

```bash
claudio restore --from-profile=personal
```

## Profiles

Each profile gets its own directory under `~/.claudio/profiles/<name>/claude/`. Auth credentials are stored there, isolated from other profiles and from `~/.claude`.

By default, `settings.json` and `hooks/` are symlinked from `~/.claude` so all profiles share your global configuration. Use `--isolated` when a profile needs its own settings or hooks.

## Multiple terminals

Each `claudio <name>` call is independent. There is no global state that one terminal can change for another.

```
Terminal A: claudio work      # work account
Terminal B: claudio personal  # personal account
```

## Project pinning

```bash
cd ~/projects/client-x
claudio pin client-x
```

Creates `.claudio-account` in the current directory. Add it to `.gitignore` if you don't want it committed.

You can also set path-based rules in `~/.claudio/config.json`:

```json
{
  "rules": [
    { "path": "~/projects/work/**", "profile": "work" },
    { "path": "~/projects/personal/**", "profile": "personal" }
  ]
}
```

When running `claudio` with no profile name, the resolution order is:

1. `.claudio-account` file (walks up to the git root)
2. Directory rules (first match)
3. If there is one profile, use that profile; otherwise show the interactive selector

## Settings and local `.claude` directories

claudio sets `CLAUDE_CONFIG_DIR` to the profile directory before launching Claude. `~/.claude` stays the source of truth for global settings: both `settings.json` and `hooks/` are symlinked from there into each profile.

Project-level `.claude` directories work as normal. Claude Code merges project settings on top of global settings and runs project hooks alongside global hooks. If a project's `.claude` has its own `settings.json` or `hooks/`, those will override or extend the profile's config, which is standard Claude Code behavior. `claudio doctor` flags any local `.claude` directories where this could be unexpected.

## License

[MIT](LICENSE)
