---
name: agent-pass
description: Manage, check quotas, and switch user accounts/profiles for AI coding agents (Antigravity, Codex, etc.).
---

# Agent Pass Skill (`agpass`)

This skill guides AI coding agents on how to autonomously inspect, manage, and switch user accounts/credentials using the `agpass` CLI.

## 🎯 When to Use This Skill

Activate this workflow when:
1. **Quota / Rate Limit Exceeded**: You encounter a 429 Too Many Requests, quota exhausted, or token limit error.
2. **Account Switching Requested**: The user explicitly asks to switch agent accounts (e.g., "switch Antigravity to my personal account", "use my work profile for Codex").
3. **Account / Quota Inspection**: The user asks what account is currently active or how much quota remains.

---

## 🛠️ Execution Protocol for AI Agents (Headless)

Always use non-interactive commands with `--json` flags to receive structured, machine-readable output.

### Workflow 1: Automatic Quota Failover
When hitting a quota limit on an agent:

1. **Check current active account**:
   ```bash
   agpass current antigravity --json
   ```
2. **List all available accounts**:
   ```bash
   agpass list --json
   ```
   *Parse the output JSON array to find accounts where `is_active` is `false`.*
3. **Switch to an alternate account**:
   ```bash
   agpass switch antigravity --to <alternate_account_name> --json
   ```
   *Verify that the returned JSON has `"success": true`.*
4. **Notify the user** that the account has been switched to continue the coding task seamlessly.

---

### Workflow 2: Inspecting Accounts & Quotas
To inspect available profiles and quotas:

```bash
# Get full list of agents and configured accounts
agpass list --json

# Get quota overview for a specific agent
agpass quota antigravity --json
```

---

### Workflow 3: Registering a New Account
To register a new profile without blocking on interactive prompts:

```bash
agpass add antigravity --name <account_name> --email <user_email> --non-interactive --json
```

---

## 📋 JSON Output Reference

### `agpass switch <agent> --to <account> --json`
```json
{
  "success": true,
  "agent": "antigravity",
  "previous_account": "work",
  "current_account": "personal",
  "message": "Successfully switched Antigravity to personal"
}
```

### `agpass list --json`
```json
[
  {
    "agent": "antigravity",
    "active_account": "work",
    "accounts": [
      {
        "name": "work",
        "is_active": true,
        "email": "work@example.com",
        "config_dir": "/path/to/config"
      },
      {
        "name": "personal",
        "is_active": false,
        "email": "personal@example.com",
        "config_dir": "/path/to/config"
      }
    ]
  }
]
```

---

## ⚠️ Safety & Fallback Rules
- **Do not modify files in `~/.agpass/` manually**: Always use `agpass` CLI commands to ensure backups are preserved.
- **Single account fallback**: If `agpass list --json` returns only 1 account for an agent, notify the user that no alternate accounts are registered and provide the command to add one (`agpass add <agent>`).