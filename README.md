# agent-pass 🎟️

> **The ultra-fast multi-account & quota manager for AI coding agents** (*Google Antigravity*, *OpenAI Codex*, and more).

[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey.svg)]()

`agent-pass` (`agpass`) is a single-purpose, lightweight CLI tool engineered to solve a common developer bottleneck: **switching accounts, credentials, and managing quotas across AI coding agents** without repeated logins or messy credential overrides.

Built with **Go**, `agent-pass` compiles into a single, standalone binary with zero external runtime requirements. It offers a sleek dark-themed terminal UI for developers and full headless JSON commands for autonomous AI agents.

---

## 📑 Table of Contents

- [The Problem & Solution](#-the-problem--solution)
- [Key Features](#-key-features)
- [Installation](#-installation)
  - [Method 1: Standard Go Install (Recommended)](#method-1-standard-go-install-recommended)
  - [Method 2: One-Line Web Installer](#method-2-one-line-web-installer)
  - [Method 3: Build from Source](#method-3-build-from-source)
- [Usage Guide](#-usage-guide)
  - [For Humans (Interactive TUI)](#1-for-human-developers-interactive-tui)
  - [For AI Agents (Headless & JSON)](#2-for-ai-coding-agents-headless--json)
- [AI Agent Skill Integration](#-ai-agent-skill-integration)
- [Project Structure](#-project-structure)
- [Configuration](#-configuration)
- [Supported Agents](#-supported-agents)
- [License](#-license)

---

## 🎯 The Problem & Solution

- **The Problem:** Developers often maintain multiple accounts for AI coding assistants (e.g., personal vs. work accounts, secondary accounts to bypass quota/rate limits). Logging out and re-authenticating disrupts coding workflows.
- **The Solution:** `agent-pass` isolates credential sessions into organized profiles, enabling instantaneous, safe switching in a single command or keystroke.

---

## ⚡ Key Features

- ⚡ **Instant Switching**: Swap agent sessions in under 50 milliseconds.
- 🎨 **Modern Terminal UI**: Color-coded progress bars, box-drawing cards, and arrow-key interactive menus (inspired by *oh-my-pi*).
- 🤖 **AI-Agent Native**: Full headless mode (`--json`, `--non-interactive`) enabling AI agents to autonomously manage their own accounts when rate limits occur.
- 🔒 **Safe State Preservation**: Automatically backs up active credentials before applying the target session.
- 🌐 **Global PATH Ready**: Built-in self-registration command (`agpass setup-path`) to invoke `agpass` anywhere.
- 🪶 **Zero Runtime Bloat**: Single standalone binary (~10MB), no Node.js or Python runtimes required.

---

## 📦 Installation

### Method 1: Standard Go Install (Recommended)
If you have Go (1.20+) installed:
```bash
go install agent-pass@latest
```
*(Make sure `$GOPATH/bin` or `%USERPROFILE%\go\bin` is in your system PATH).*

---

### Method 2: One-Line Web Installer

#### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/agent-pass/agent-pass/main/install.ps1 | iex
```

#### macOS / Linux
```bash
curl -fsSL https://raw.githubusercontent.com/agent-pass/agent-pass/main/install.sh | bash
```

---

### Method 3: Build from Source
```bash
# Clone the repository
git clone https://agent-pass.git
cd agent-pass

# Build using build scripts or Makefile
# Windows:
.\scripts\build.ps1
# Unix:
make build

# Register globally in PATH
.\bin\agpass setup-path
```

---

## 📖 Usage Guide

### 1. For Human Developers (Interactive TUI)

```bash
# Display help and ASCII art banner
agpass

# List all configured agents and accounts in a tree view
agpass list

# View quota usage with color-coded progress bars
agpass quota

# Interactively choose and switch accounts with arrow keys (↑/↓)
agpass switch antigravity

# Add a new account profile
agpass add antigravity
```

---

### 2. For AI Coding Agents (Headless & JSON)

AI coding agents (Antigravity, Codex, Claude Code, Cursor) can invoke `agpass` headlessly without interactive keyboard input:

#### List Accounts (JSON)
```bash
agpass list --json
```

#### Get Active Account
```bash
agpass current antigravity --json
```

#### Check Remaining Quota (JSON)
```bash
agpass quota antigravity --json
```

#### Switch Account Headlessly
```bash
agpass switch antigravity --to work --json
# or
agpass switch codex -a dev-team --json
```

#### Add Account Headlessly
```bash
agpass add antigravity --name personal --email user@example.com --non-interactive --json
```

---

## 🧠 AI Agent Skill Integration

`agent-pass` includes an autonomous skill definition (`SKILL.md`) that allows AI coding assistants to automatically detect rate limits and switch accounts.

To install the skill into your global and workspace agent directories:
```bash
agpass install-skill
```

This installs the skill to:
- `~/.gemini/config/skills/agent-pass/SKILL.md` *(Antigravity Global Config)*
- `.agents/skills/agent-pass/SKILL.md` *(Workspace Config)*

---

## 📁 Project Structure

```
agent-pass/
├── cmd/                    # Cobra CLI commands (root, list, switch, quota, add, current, setup, skill)
├── internal/
│   ├── agents/             # Agent registry and switch engine (Antigravity, Codex)
│   ├── config/             # YAML configuration manager (~/.agpass/config.yaml)
│   ├── quota/              # Quota checker logic
│   └── ui/                 # Lipgloss styles, banner, and Bubbletea TUI selectors
├── scripts/                # Development build scripts (build.ps1, build.sh)
├── skills/                 # AI Agent skill definitions (SKILL.md)
├── Makefile                # Standard Go Makefile
├── install.ps1             # Windows installer & PATH registrar
├── install.sh              # Unix installer & PATH registrar
├── go.mod                  # Go module definition
├── go.sum                  # Go checksums
├── .gitignore              # Ignored build artifacts and local configs
└── README.md               # Project documentation
```

---

## ⚙️ Configuration

Profiles and states are persisted in `~/.agpass/config.yaml`:

```yaml
agents:
  antigravity:
    active: work
    accounts:
      - name: work
        email: dev@company.com
        config_dir: ~/.gemini
      - name: personal
        email: user@gmail.com
        config_dir: ~/.gemini
  codex:
    active: primary
    accounts:
      - name: primary
        config_dir: ~/.codex
```

---

## 🤖 Supported Agents

| Agent | Default Config Directory | Status |
| :--- | :--- | :--- |
| **Google Antigravity** | `~/.gemini/` | ✅ Supported |
| **OpenAI Codex** | `~/.codex/` | ✅ Supported |
| **Claude Code** | `~/.claude/` | 🚧 Roadmap |
| **Cursor** | Local state storage | 🚧 Roadmap |

---

## 📄 License

MIT License © 2026 agent-pass