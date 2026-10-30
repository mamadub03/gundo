# Gundo — Architecture

> **Status:** Reconstructed from project discussions through October 2026. The Mac-controller / Windows-client MVP was demonstrated; exact source details and any subsequent edits must be checked against the current repository. This is not a claim that every proposed feature has been implemented.

## 1. Goal and boundaries
Gundo is a Go controller/client system for authorized security competitions. Its defining current design is **asynchronous communication through a third-party SaaS API** rather than a direct controller-to-client connection. Today that API is GitHub Issues/comments.

The intended long-term separation is:
- **Operations layer:** operator interface, discovery, client selection, task tracking, results, and eventual web UI.
- **Transport layer:** GitHub API integration now; potentially interchangeable authorized SaaS adapters later.
- **Execution layer:** client lifecycle, health reporting, task dispatch, and a constrained action handler.

These are **logical boundaries**; they do not imply a completed package/plugin architecture. MCP, Meta-glasses control, AI-generated transports, and a multi-SaaS catalog were ideas discussed for later—not confirmed v1 functionality.

## 2. Working MVP data flow

```text
Mac / operator controller                 Windows / remote test client
  discoverClients()                         loadState() / saveState()
  choose online client                      heartbeatLoop()
  create task                               pollTasks()
        |                                          ^
        v                                          |
  GitHub Issue dedicated to THIS client (comments form mailbox)
        ^                                          |
        |                                          v
  waitForResult(TaskID)                    handleTask() allowlist
        ^                                          |
        +----------- GitHub result comment <------+ 
```

**Exactly one Issue per client** is the established MVP choice. Multiple clients have separate Issues, not one shared task thread. A GitHub Issue is a logical mailbox, not proof that its client is currently online.

## 3. Identity, enrollment, and restart
Client state was defined as:

```go
type ClientState struct {
    ClientID    string `json:"client_id"`
    IssueNumber int    `json:"issue_number"`
}
```

Historically, `state.go` stores this under the OS user configuration directory in `gundo/state.json` using `os.UserConfigDir()`.

**First run:** load state → if absent, generate a UUID (`ClientID`) → `createIssue()` → save the ID and GitHub Issue number → initialize client → start heartbeat and task polling.

**Restart:** load the same ID and Issue number → resume the client loops. A missing state file is currently treated as a new installation, potentially leaving an older Issue behind.

**Not yet implemented/verified:** recovering when the Issue is deleted while local state remains. The intended later behavior is to confirm a genuine missing mailbox, preserve `ClientID`, create a replacement Issue, save the new `IssueNumber`, then resume. Never recreate on network failures or rate limiting.

## 4. Message contract
The project uses structured JSON encoded in GitHub Issue comments. Key message classes:

| Type | Origin | Relevant fields | Meaning |
|---|---|---|---|
| `task` | Controller | `Type`, `TaskID`, `TargetClientID`, `Action`, `Timestamp` | Request a supported action on one client |
| `result` | Client | `Type`, `TaskID`, `ClientID`, `Status`, `Output`, `Timestamp` | Return outcome for the same logical task |
| `heartbeat` | Client | `Type`, `ClientID`, `Status`, `Timestamp` | Report liveness |

`TaskID` is shared between task and result. GitHub's comment ID is a separate identifier used for new-message detection/deduplication. `TargetClientID` is checked by the client before accepting a task; per-client Issues are not a substitute for message validation.

The known MVP client actions are **`ping`**, **`get_hostname`**, **`get_os`**, and **`get_user`**. Unsupported actions return an error. The MVP is **not** an unrestricted shell executor.

## 5. Historical Go implementation map
These names were used in the project discussions; inspect the live code to verify them:

| File | Main responsibilities / functions |
|---|---|
| `main.go` | `main()` mode selection; `runClient()`, `runController()`, `discoverClients()`, `runCommandMenu()`, `waitForResult()` |
| `client.go` | Client runtime, `heartbeatLoop()`, `pollTasks()`, `handleTask()` |
| `state.go` | `ClientState`, `getStatePath()`, `loadState()`, `saveState()` |
| `github.go` | `createIssue()`, `getIssue()`, `listIssues()`, `listComments()`, `createComment()` |
| `messages.go` | `GundoMessage` helpers, task-ID creation, message parsing |
| `models.go` | GitHub request/response structs and/or shared models |
| `poller.go` | Older generic `pollComments()` experiment; reported as unused in the active MVP path |
| `message_test.go`, `client_test.go` | Existing test coverage discussed for messages/client behavior |

Historically all these Go source files used `package main`, so the development invocation was `go run . client` or `go run . controller`, **not** `go run main.go`.

## 6. Heartbeat, discovery, and result waiting
- Client heartbeats were sent **immediately and then approximately every 2 minutes**.
- `discoverClients()` listed GitHub Issues with the client title prefix, located the latest matching heartbeat, and determined `ONLINE` if less than roughly **6 minutes** old.
- `pollTasks()` historically polled about every **10 seconds**; `waitForResult()` checked for the matching `TaskID` about every **5 seconds**.
- The terminal controller could select a client, choose a predefined diagnostic action, post a task, then display the matching result.
- The controller's displayed online state was a **snapshot**; a blocking terminal prompt did not continuously refresh it.

These timing values describe the last discussed implementation, not promised configuration defaults.

**Current polling implementation (Phase 2, October 2026):** `pollTasks()` performs a full paginated startup read to establish its baseline, then uses GitHub `since` filtering. A `commentCheckpoint` stores the highest seen comment ID and its GitHub `created_at`; requests start one second before that timestamp, and the ID check filters repeated comments. Empty mailboxes remain unfiltered until their first comment. Every matching page must succeed, with valid creation metadata for new IDs, before retrieval progress advances. Polling intervals and diagnostic actions are unchanged.

The controller retains a separate bookmark and latest valid heartbeat for each discovery mailbox across menu refreshes, recalculating heartbeat age even when no new heartbeat arrives. Each result wait starts from the ID and timestamp returned when its task comment was posted, then polls independently. Checkpoints are in memory only; client state still contains only ClientID and IssueNumber. Failed result POSTs retain their existing logging behavior; durable delivery and retries are separate work.

## 7. Proven MVP milestone
The team verified a **controller running on macOS and a Go client running on a separate physical Windows computer**. The Windows client maintained its own state and GitHub credential; the Mac controller independently discovered the client's Issue, tasked its fixed diagnostic actions, and received correlated results. Restart persistence and the basic heartbeat path were also tested.

This confirms a functional **terminal-operated MVP**, not production readiness, comprehensive resilience, or a verified completed web dashboard.

## 8. Known API issue and next architecture work
A live investigation found `listComments()` returning exactly **30 comments** from an Issue, falsely marking a client offline because newer heartbeats were not retrieved. Changing the request to `?per_page=100` restored operation, but **only postpones the same bug to comment #101**.

Correctness and efficiency are separate:

1. **Pagination (implemented):** follow GitHub's `Link: ... rel="next"` response headers to retrieve all relevant pages when needed. This prevents truncation; failed reads do not return partial success.
2. **Incremental retrieval (implemented):** use independent comment-ID/creation-time bookmarks and overlapping `since` queries. Existing IDs are ignored even if returned again; the current message workflow is append-only. Startup, checkpoint persistence, and durable result delivery are not redesigned by this phase.
3. **Conditional requests (pending approval):** use `ETag` / `If-None-Match` for stable request URLs where appropriate; handle HTTP `304` without decoding a comment body.
4. **Rate-limit resilience (pending approval):** inspect rate-limit and retry headers, distinguish primary and secondary limits, and back off correctly rather than spinning or silently losing messages.

`listComments()` was used by **client task polling**, **controller heartbeat discovery**, and **result waiting**, so changes must preserve all three flows. `listIssues()` also historically used a single `per_page=100` request and needs a separate pagination review.

**Important:** pagination does not make rate limiting disappear. Re-fetching 1,000 historical comments every 10 seconds would consume many requests even with correct paging. Avoid that through incremental/conditional polling.

## 9. Credentials and deployment
GitHub operations historically read `GITHUB_TOKEN` from the environment. The Mac workflow used a token stored in Keychain and exported into a terminal session; the Windows client had a **separate fine-grained GitHub PAT** scoped to the same authorized repository. Tokens should not be persisted in `state.json` or binaries.

Deployment was proven manually with a cross-compiled Windows binary; repeatable packaging/provisioning, token rotation and secrets management remain development work. A future GitHub App may be worth evaluating but is **not** a confirmed MVP dependency.
