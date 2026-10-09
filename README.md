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
| `claudio copy --to-profile=<name> --from-local [--with-history]` | Create a profile from your local installation's configuration (`~/.claude`) |
| `claudio copy --to-profile=<name> --from=<profile> [--with-history]` | Create a profile from another profile's configuration |
| `claudio restore` | Copy the origin account back to the local installation (`~/.claude`) |
| `claudio login <name>` | Open Claude with the given profile to authenticate |
| `claudio <name> [-- args...]` | Launch Claude with that profile |
| `claudio exec <name> -- <command> [args...]` | Run any command with `CLAUDE_CONFIG_DIR` set to that profile |
| `claudio switch` | Interactive profile selector |
| `claudio list` | List all profiles |
| `claudio current` | Show which profile applies to the current directory |
| `claudio pin <name>` | Pin a profile to the current directory |
| `claudio pin unpin` | Remove the pin |
| `claudio rename <old> <new>` | Rename a profile |
| `claudio remove <name>` | Delete a profile |
| `claudio model <name>` | Show the default model for a profile |
| `claudio model <name> <model>` | Set the default model for a profile |
| `claudio model <name> --unset` | Remove the default model |
| `claudio doctor` | Check the setup |
| `claudio manage` | Open the interactive Task Hub |

## Interactive manager

Run `claudio manage` to open the full-screen Task Hub. It includes:

- **Launch Claude:** choose a profile and start a Claude session.
- **Profile Library:** create profiles, log in, rename, launch, and remove profiles.
- **Project Routing:** see the profile resolved for the current directory, pin or unpin a project, and add or remove path rules.
- **Check setup:** inspect Claude availability, configuration, profile directories, and the active route.

Use the arrow keys and Enter to navigate. Press `?` for the key guide. Existing commands such as `claudio create`, `claudio switch`, and `claudio <profile>` remain available.

For a static illustrative frame (with sample profile names and paths):

```bash
claudio manage --frame --screen home --cols 100 --rows 30
```

## Copying an existing configuration

If you already use Claude Code in `~/.claude`, copy it into a profile:

```bash
claudio copy --to-profile=personal --from-local
```

This copies the configuration of `~/.claude` and `~/.claude.json` (settings, `CLAUDE.md`, plugins, skills, hooks and MCP servers) into `~/.claudio/profiles/personal/claude/` and marks the profile as the **origin account**. History, conversations and sessions are personal data of an account, so they are copied only if you ask for them, typically when the profile will use the same account:

```bash
claudio copy --to-profile=personal --from-local --with-history
```

The account identity is never copied; Claude fills it in when you log in. What counts as configuration, history or disposable cache is defined in [`claude.toml`](claude.toml) (`[copy]`). Paths that pointed into `~/.claude`, such as hook commands and plugin locations, are rewritten to the profile. From then on the profile is independent: nothing is shared or synced with `~/.claude`, which is left untouched and keeps working with plain `claude`.

To start a profile from another profile's configuration:

```bash
claudio copy --to-profile=work-2 --from=work
```

Login tokens are stored in the system keychain per config directory and are never copied, so log in once in the new profile (`claudio login <name>`).

To copy the origin account back to the local installation:

```bash
claudio restore
```

Existing `~/.claude` and `~/.claude.json` are backed up as `<name>.bak.<timestamp>` before being replaced. To restore a specific profile instead of the origin account:

```bash
claudio restore --from-profile=personal
```

## Default model per profile

Assign a default Claude model to any profile so it launches with that model automatically:

```bash
claudio model personal claude-sonnet-4-6
claudio model work claude-opus-4-5

claudio model personal          # show current default
claudio model personal --unset  # remove the default
```

When a profile has a default model set, claudio prepends `--model <model>` every time it launches Claude with that profile. If you pass `--model` yourself, your value takes precedence.

## Profiles

Each profile is a fully independent Claude Code config directory under `~/.claudio/profiles/<name>/claude/`, with its own account, history, settings, plugins, skills, hooks and MCP servers. `claudio create` leaves it empty and Claude builds its layout on first launch, exactly as it does for `~/.claude`, so profiles always match the Claude Code version you run.

Anything you install from a profile session stays in that profile, because Claude runs with `CLAUDE_CONFIG_DIR` pointing at it:

```bash
claudio work                               # then /plugin, /mcp, /config inside the session
claudio work plugin install drawio@drawio  # Claude CLI subcommands work too
claudio work mcp add <name> -- <command>
```

Third-party installers reach a profile only if they honor `CLAUDE_CONFIG_DIR`. `claudio exec` runs any command with it set to a profile, without launching Claude, so you never have to set the variable yourself:

```bash
claudio exec work -- npx skills add <owner/repo> -g
claudio exec work -- mnemo setup refresh --agent=claudecode
```

It replaces any `CLAUDE_CONFIG_DIR` already in the environment, so it also targets the right profile when run from inside another profile's session.

Installers that ignore `CLAUDE_CONFIG_DIR` keep writing to `~/.claude` or `~/.claude.json`, even through `claudio exec`. Some honor it only in part: `mnemo setup` puts its skill and `CLAUDE.md` block in the profile but its MCP server in `~/.claude.json`, and `codegraph install` writes only to `~/.claude.json`. Check the profile afterwards with `claudio work mcp list` and register any missing MCP server with `claudio work mcp add`:

```bash
claudio work mcp add -s user codegraph -- codegraph serve --mcp
```

### Skills

Global skills installed with [`npx skills`](https://skills.sh) work per profile. The `skills` CLI stores each skill once in `~/.agents/skills/` and symlinks it into `$CLAUDE_CONFIG_DIR/skills/`, so each profile gets its own link to the same skill:

```bash
claudio exec work -- npx skills add <owner/repo> -g
claudio exec personal -- npx skills add <owner/repo> -g
```

Project skills (without `-g`) go into the project's `.claude/skills/` and are shared by every profile that opens the project.

### Agent skill

claudio ships a skill, `claudio-profiles`, that tells Claude how to install into profiles: when you ask it to add an MCP server, a skill, a plugin or run a tool's installer, it picks the target profile and uses `claudio exec` or `claudio <profile> mcp|plugin` instead of writing to `~/.claude` or copying files by hand. Install it in each profile where you want it:

```bash
claudio exec work -- npx skills add jmeiracorbal/claudio -g
```

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

claudio sets `CLAUDE_CONFIG_DIR` to the profile directory before launching Claude. Claude also loads `.claude/CLAUDE.md` and `.claude/rules/` from every directory above the working directory as project instructions, and your home directory is one of them, so the local installation's `~/.claude/CLAUDE.md` would leak into every profile. claudio prevents it by passing `--settings` with `claudeMdExcludes` on every launch; the patterns live in [`claude.toml`](claude.toml) (`[launch]`). A `--settings` you pass yourself takes precedence.

Project-level `.claude` directories work as normal. Claude Code merges project settings on top of global settings and runs project hooks alongside global hooks. If a project's `.claude` has its own `settings.json` or `hooks/`, those will override or extend the profile's config, which is standard Claude Code behavior. `claudio doctor` flags any local `.claude` directories where this could be unexpected.

## Troubleshooting

### After `claudio copy`, Claude asks me to log in again

This is expected. Claude Code stores login tokens in the system keychain, keyed by config directory, and they are not copied. Your configuration (and history, with `--with-history`) is copied; only the login needs to be done once:

```bash
claudio login personal
```

### `claudio copy` fails with "profile already exists"

`claudio copy` always creates a new profile. Pick another name, or remove the existing profile first:

```bash
claudio remove <name>
claudio copy --to-profile=<name> --from-local
```

## License

[MIT](LICENSE)
