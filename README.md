# Gundo

Gundo is a Go-based controller/agent platform built for authorized
collegiate red-team and cyber defense competitions.

## Architecture

Controller
    ↓
GitHub Issues / Comments
    ↑
Agent

GitHub Issues act as persistent agent mailboxes, while comments carry
structured task and result messages.

## Project Structure

main.go       - controller entry point
github.go     - GitHub transport
messages.go   - Gundo message protocol
models.go     - shared data models
poller.go     - comment polling logic
agent/        - agent implementation

## Credential setup and launching

Gundo stores its GitHub token in macOS Keychain or Windows Credential Manager
for the OS user running Gundo. It no longer reads or sets `GITHUB_TOKEN`.
Use a token scoped to the configured repository with Issues read/write access.
The repository remains `mamadub03/gundo-test-repo`, configured in the source.

Run from the directory containing `go.mod`:

```sh
go run . setup
go run . client
# On the controller machine instead:
go run . controller
```

With built binaries, use `./gundo setup` on Mac or `.\gundo.exe setup` on Windows,
then the corresponding `client` or `controller` mode. Run setup under the same
OS account that will launch Gundo. With no stored token, either mode prompts
for one before enrollment or polling. Input is hidden; no token argument or
environment export is needed. A missing terminal causes setup to fail clearly.

Setup performs two read-only requests: authenticated identity (`GET /user`),
then one page with at most one Issue from the configured repository. Each
request has a ten-second timeout and redirects are rejected. Successful setup
proves authentication and Issues read access, **not** write permission. It does
not create an Issue. Only the normal first client launch enrolls a new client.
A rejected candidate token is not saved. Locked/inaccessible storage does not
trigger automatic replacement or an environment/plaintext fallback.

Run `setup` again to replace an expired/revoked token. This does not change
ClientID, IssueNumber, or `state.json`. Restart running processes afterward:
each launch loads the stored token once into memory. Existing saved tokens
are not preflight-validated on every launch; normal API errors still apply.
Old environment settings are not removed automatically and are now ignored.
Existing manually-created Keychain entries are not imported automatically.

The credential entry uses service `gundo` and account
`github:mamadub03/gundo-test-repo`. Both modes under the same OS user use that
entry. Keychain access may require OS approval/unlocking. Credential-store
protection does not prevent extraction by a sufficiently privileged local user;
the process needs the usable token in memory.

## Verification

Automated tests use fake storage, fake token input, and mocked HTTP responses;
they must not access real credential entries or GitHub. Standard checks:

```sh
gofmt -l *.go
go test ./...
go vet ./...
go build ./...
```

Manual Mac-controller/Windows-client verification (not performed automatically):

1. Run `setup` on each machine as its intended OS user. Confirm input is hidden
   and successful setup does not create an Issue or change existing client state.
2. Launch each mode without supplying `GITHUB_TOKEN`. Existing installations
   should retain their ClientID/IssueNumber; a fresh client should create one Issue.
3. Send `ping` from the controller and verify the result. This checks write access.
4. Stop and relaunch both: no token prompt should be needed while the store is
   accessible. OS credential-access prompts may still appear.
5. Enter a deliberately invalid replacement through `setup`; validation should
   fail and the previous stored credential should continue working on relaunch.

Windows cross-compilation checks compilation only; it does not verify Windows
Credential Manager behavior. No automatic retry, mailbox recovery, ETags, or
background service installation is included in credential setup.

## Operator console

Run the existing `controller` mode for the banner and interactive agent/action
menus. Use arrow keys and Enter, or `/` to search the visible entries. Ctrl+C
backs out of a selection (exits from the agent list). Refresh and Quit are menu
entries. All agents are shown initially; select Hide offline agents to narrow the list. Online agents sort first, then by Issue number. Compact rows show a green Online/Offline label and the agent
Issue number; the selected-agent view shows the full ID and heartbeat age. No aliases are stored.

Agent status remains Online/Offline and is a discovery snapshot, not a live
background refresh. A failed mailbox read is logged and remains Offline, as
before. Toggling the offline view uses the current snapshot without another
GitHub request. Empty discovery results still allow refresh or quit.

Action submission shows the selected action, target Issue and TaskID, followed by
static waiting text and the final result/output. Shutdown requires typing `yes`;
an empty custom-command input cancels that selection. Result output has terminal
control sequences removed. There is no spinner, progress percentage, execution
acknowledgment, or in-app cancellation of result waiting in this increment.

Set `NO_COLOR` to disable colors. Set `GUNDO_PLAIN=1` to use numbered menus;
non-terminal input/output and `TERM=dumb` also use the plain fallback. Enter `0`
to back out of a numbered menu. Existing launch commands and credentials apply.

Console verification: mocked tests cover selection mapping, cancellation without
posting, sanitized output, and a full mocked ping/result path. A local PTY preview
with fake agents verified arrow-down selection and result rendering. Mac and
Windows builds pass; actual Windows terminal appearance and live GitHub use still
need manual verification.

Interactive console navigation redraws the screen when entering/leaving an agent,
refreshing, or toggling the list. Within an agent session, action output and errors remain visible while the next
action menu appears immediately; no extra Enter prompt is required. Plain/redirected output retains its transcript without clearing
or extra acknowledgement prompts. The banner uses slanted green ASCII lettering.
