# Gundo Agent — Build & Deployment Guide (Option B)

This guide explains how to compile, deploy, and enroll the **Gundo Agent** on remote target systems without triggering OS password modals or Apple Keychain prompts.

---

## 1. Overview & Architecture

Gundo agents communicate asynchronously with the controller by using GitHub Issues as mailboxes. 
- **Zero OS Keyring Prompts**: The agent never accesses Apple Keychain or Windows Credential Manager on the target machine.
- **Self-Cleaning Staging File**: The agent ingests configuration from a temporary setup file and immediately wipes/deletes the setup file from disk.
- **Persistent Local State**: The agent saves its identity and connection credentials into a restricted local state file (`state.json`) so it automatically reconnects across system reboots.

---

## 2. Building the Agent Binary

Build the agent on your operator machine for the target OS:

```bash
# Windows Agent (x86_64)
GOOS=windows GOARCH=amd64 go build -o gundo-agent.exe ./agent/cmd

# Linux Agent (x86_64)
GOOS=linux GOARCH=amd64 go build -o gundo-agent-linux ./agent/cmd

# macOS Agent (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o gundo-agent-darwin ./agent/cmd
```

---

## 3. Deployment & Enrollment (Option B Workflow)

### Step 1: Drop the Temporary Setup File
Create a temporary setup file named `gundo_setup.json` in the system temporary directory or the agent's folder:
- **Linux / macOS:** `/tmp/gundo_setup.json`
- **Windows:** `%TEMP%\gundo_setup.json` (or in the same directory as `gundo-agent.exe`)

**File Content:**
```json
{
  "repo": "mamadub03/gundo",
  "token": "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
```
*(You can use `owner/repo` or full URLs like `https://github.com/owner/repo/issues`).*

### Step 2: Run the Agent
Execute the agent binary on the target machine:
```bash
./gundo-agent
# or on Windows:
.\gundo-agent.exe
```

### Step 3: What Happens Automatically
1. **Detection & Ingestion:** The agent detects the temporary setup file and reads the `repo` and `token`.
2. **Mailbox Registration:** It contacts GitHub to create a new dedicated Issue titled `Gundo Client - <UUID>`.
3. **State Persistence:** It saves its configuration with restricted file permissions (`0600`) to:
   - **Windows:** `%APPDATA%\gundo\state.json`
   - **Linux:** `~/.config/gundo/state.json`
   - **macOS:** `~/Library/Application Support/gundo/state.json`
4. **Auto-Cleanup:** The agent **immediately deletes** the temporary `gundo_setup.json` file from disk.
5. **Heartbeat & Task Polling:** The agent starts sending periodic heartbeats and listens for commands.

---

## 4. Reboots & Subsequent Runs

On subsequent machine boots or process restarts:
- The agent reads directly from its local `state.json`.
- It does **not** look for a temp file and does **not** prompt for credentials.
- It immediately resumes checking for tasks in its existing GitHub Issue mailbox.

---

## 5. Alternative: Environment Variable Enrollment

If you prefer not to write a temporary file to disk, you can also launch the agent with environment variables on the first run:

```bash
GUNDO_REPO="mamadub03/gundo" GUNDO_TOKEN="ghp_xxxx" ./gundo-agent
```
On the first run, the agent will persist these into `state.json` and subsequent runs can omit the environment variables.

---

## 6. Teardown & Removal

To cleanly remove the agent from the target machine:
1. Terminate the running agent process.
2. Delete the persistent state directory:
   - **Linux:** `rm -rf ~/.config/gundo`
   - **macOS:** `rm -rf ~/Library/Application\ Support/gundo`
   - **Windows:** `Remove-Item -Recurse -Force "$env:APPDATA\gundo"`
3. Close the corresponding client Issue on GitHub.
