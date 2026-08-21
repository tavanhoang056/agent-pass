---
name: agent-pass
description: Manage, check quotas, and switch user accounts/profiles for AI coding agents (Antigravity, Codex, etc.).
---

# Agent Pass Skill (agpass)

Use `agpass` to manage, inspect, and switch accounts/credentials for AI coding agents.

## When to use
- When the current agent account has exhausted its quota or hit a rate limit.
- When switching between work, personal, or different client project profiles.
- When you need to inspect available accounts and remaining request allowances.

## CLI Commands for AI Agents (Headless)

### 1. List configured accounts
```bash
agpass list --json
```

### 2. Check current active account
```bash
agpass current antigravity
```

### 3. Check remaining quota
```bash
agpass quota antigravity --json
```

### 4. Switch account (Non-interactive)
```bash
agpass switch antigravity --to <account_name> --json
# or
agpass switch codex <account_name> --json
```

### 5. Add a new account
```bash
agpass add antigravity --name work --email user@example.com --non-interactive --json
```
