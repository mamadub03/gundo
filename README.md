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
