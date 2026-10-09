---
name: claudio-profiles
description: Use whenever the user wants to install, add or configure anything that lives in the Claude Code config directory (MCP servers, skills via `npx skills`, plugins, hooks, global CLAUDE.md, or a third-party installer such as `mnemo setup` or `codegraph install`) and claudio profiles are in use, i.e. CLAUDE_CONFIG_DIR points under ~/.claudio/profiles/ or the user names a claudio profile. Routes the install to the right profile with `claudio exec` / `claudio <profile> mcp|plugin` instead of env-var prefixes or copying files by hand.
---

# Installing into claudio profiles

claudio runs Claude with `CLAUDE_CONFIG_DIR=~/.claudio/profiles/<name>/claude`. Each profile has its own MCP servers, skills, plugins, hooks and global `CLAUDE.md`. Anything written to `~/.claude` or `~/.claude.json` only affects the local installation, not any profile.

## 1. Find the target profile

```bash
echo "$CLAUDE_CONFIG_DIR"   # ~/.claudio/profiles/<name>/claude → current profile is <name>
claudio list                # all profiles
```

The target is the profile the user names. If they name none, it is the current profile; say which one before installing. If the user wants it in several profiles, repeat the command per profile.

## 2. Pick the command

| What | Command |
|---|---|
| MCP server | `claudio <profile> mcp add [-s user] <name> -- <command>` |
| Plugin | `claudio <profile> plugin install <plugin>` |
| Global skill (`npx skills`) | `claudio exec <profile> -- npx skills add <owner/repo> -g` |
| Third-party installer | `claudio exec <profile> -- <installer> [args]` (e.g. `mnemo setup refresh --agent=claudecode`, `codegraph install`) |
| Per-project setup (`codegraph init`, `mnemo init`, project skills without `-g`) | Run normally in the project; it is not tied to a profile |

`claudio exec` runs any program with `CLAUDE_CONFIG_DIR` set to that profile, replacing any inherited value, so it is correct both outside Claude and from inside another profile's session.

Do not:
- prefix commands with `CLAUDE_CONFIG_DIR=...` by hand,
- create symlinks, copy skill folders or edit `settings.json`/`.claude.json`/`CLAUDE.md` inside a profile directory by hand,
- run an installer without `claudio exec` when the target is not the current profile.

## 3. Verify where it landed

Third-party installers reach a profile only if they honor `CLAUDE_CONFIG_DIR`. After running one, check the profile:

```bash
claudio <profile> mcp list
ls ~/.claudio/profiles/<profile>/claude/skills
```

If the installer wrote to `~/.claude` or `~/.claude.json` instead, tell the user it ignores `CLAUDE_CONFIG_DIR`, and offer the equivalent through claudio (usually `claudio <profile> mcp add ...` with the same command, args and env the installer used). Ask before editing profile files directly.

Changes apply to new sessions of that profile; tell the user to restart Claude with `claudio <profile>`.
