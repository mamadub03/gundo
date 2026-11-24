# Gundo Controller — Build & Setup Guide

This guide explains how to build, configure, and operate the **Gundo Controller** — the operator interface for managing remote agents through Gundo C2.

---

## 1. Prerequisites

Before setting up the controller, ensure you have:

1. **Go 1.22+ installed** (`go version` to verify).
2. **A GitHub Repository** to act as the C2 communication channel (e.g., `owner/repo`).
   - GitHub Issues must be enabled in the repository settings (**Settings** → **Features** → **Issues**).
3. **A GitHub Personal Access Token (PAT)**:
   - **Recommended:** Fine-Grained Personal Access Token.
     - **Repository Access:** Selected repositories → Choose your C2 repo.
     - **Permissions:** **Issues** → Read and write.
   - **Classic PAT Alternative:** Scoped to `repo` (for private repositories) or `public_repo` (for public repositories).

---

## 2. Building the Controller

You can run the controller directly with Go or compile a standalone binary.

### Option A: Run Directly with Go
```bash
# Using the unified entrypoint
go run . controller

# Or directly from the controller package
go run ./controller/cmd
```

### Option B: Build a Standalone Executable

#### 1. Build the Unified Binary (Agent, Controller & Setup Dispatcher)
```bash
go build -o gundo .
```
Run the controller using:
```bash
./gundo controller
```

#### 2. Build the Dedicated Controller Binary
```bash
go build -o gundo-controller ./controller/cmd
```
Run directly:
```bash
./gundo-controller
```

### Cross-Compilation Examples
Build for different operator operating systems:

```bash
# macOS (Apple Silicon ARM64)
GOOS=darwin GOARCH=arm64 go build -o gundo-controller-darwin-arm64 ./controller/cmd

# Linux (x86_64)
GOOS=linux GOARCH=amd64 go build -o gundo-controller-linux-amd64 ./controller/cmd

# Windows (x86_64)
GOOS=windows GOARCH=amd64 go build -o gundo-controller.exe ./controller/cmd
```

---

## 3. Configuration & Authentication Setup

The controller requires two pieces of information:
1. **Target GitHub Repository** (e.g., `owner/repo` or full URL `https://github.com/owner/repo/issues`)
2. **GitHub Personal Access Token (PAT)**

Gundo supports three flexible methods to provide credentials:

### Method 1: Interactive Setup Wizard (Saved to OS Keyring)
Run the setup command:
```bash
go run . setup
# or with compiled binary:
./gundo setup
```
You will be prompted:
```text
GitHub repository (owner/repo or URL): mamadub03/gundo
GitHub token (input hidden): **********
```
- The setup command validates both your identity and Issues read access with GitHub before saving.
- Credentials are encrypted and saved in your operating system's native credential store (Apple Keychain on macOS, Windows Credential Manager on Windows, or Secret Service / Keyring on Linux).

### Method 2: Automatic First-Run Prompt
If you run the controller on a fresh system without running setup:
```bash
go run . controller
```
The controller detects missing credentials and automatically prompts you in the terminal, validates them, saves them to the OS keyring, and immediately launches the operator interface.

### Method 3: Environment Variables (Headless / CI / Overrides)
To bypass the keyring or run with temporary credentials:
```bash
export GITHUB_REPO="owner/repo"
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxxxxxxxxx"
go run . controller
```

Or pass them inline for a single execution:
```bash
GITHUB_REPO="owner/repo" GITHUB_TOKEN="ghp_xxxx" ./gundo controller
```

*Note: `GUNDO_REPO` and `GUNDO_TOKEN` are also accepted aliases.*

---

## 4. Operating the Controller

When started, the controller initializes the interactive operator console.

### 1. Agent Discovery Screen
The controller queries GitHub Issues in your target repository:
- Each agent registers an issue with the title `Gundo Client - <UUID>`.
- The controller scans heartbeat comments to determine agent health.
- Status Indicators:
  - **ONLINE (green):** Agent sent a heartbeat within the last 6 minutes.
  - **OFFLINE (gray):** No heartbeat within the threshold.

```text
  GUNDO · DISCOVERY · mamadub03/gundo
  Issues mailbox transport · 1 online

  Use the arrow keys to navigate: ↓ ↑ → ←
  ? Select an agent:
    ▸ Issue #1 · 7a42f6d2-… · ONLINE
      Issue #2 · e1b34a90-… · OFFLINE
      Exit
```

### 2. Command Menu
Select an online agent to open the action menu:
- **Ping** — Check round-trip responsiveness.
- **Get hostname** — Retrieve the agent host name.
- **Get operating system** — Retrieve OS details (Windows, Linux, macOS).
- **Get current user** — Retrieve the user account under which the agent runs.
- **Custom command…** — Enter interactive custom command mode.
- **Shutdown…** — Send a shutdown signal to terminate the remote agent process.
- **Back to agents** — Return to the discovery screen.

### 3. Interactive Custom Command Shell
Selecting **Custom command…** opens an interactive terminal prompt:
```text
  CUSTOM COMMANDS · Agent #1
  Type commands to run on the agent. Type 'exit' or 'quit' to return to menu.

Agent #1 (7a42f6d2-...) > whoami
  Task     3f9a1c20-8022-4a0b-9671-5ad978189876
  Task posted. Waiting for result…
  Result:
  desktop-pc\operator
```
Type `exit` or `quit` to return to the action menu.

---

## 5. Troubleshooting & FAQ

| Problem | Cause | Solution |
| :--- | :--- | :--- |
| **`Authentication setup failed: 401 Unauthorized`** | The token is invalid, expired, or revoked. | Run `go run . setup` to supply a fresh token. |
| **`Authentication setup failed: 404 Not Found`** | Repository name is mistyped, does not exist, or token lacks access to private repository. | Verify repository spelling or verify token scopes. Ensure Issues are enabled under repo Settings. |
| **`Authentication setup failed: 403 Forbidden`** | GitHub rate limit exceeded or token lacks required permissions. | Check token permissions (needs `Issues: Read & Write`). |
| **Agent stays `OFFLINE`** | The agent process is not running, blocked by outbound firewall, or rate limited. | Verify the agent is running on the target machine and can reach `api.github.com`. |
| **OS Keyring access error on headless Linux** | Headless servers without D-Bus / SecretService daemon cannot use the OS keyring. | Export `GITHUB_REPO` and `GITHUB_TOKEN` environment variables instead. |
