# Gundo — Development Plan

> **Living backlog, last reconstructed October 2026.** Check the repository and tests before changing statuses. This is the only project-wide development checklist; no separate v1 release checklist or per-feature harness is required.

## MVP — demonstrated / discussed as working
- [x] Go controller/client modes with GitHub Issues/comments transport.
- [x] One GitHub Issue mailbox per client installation.
- [x] Persistent local `ClientID` and `IssueNumber` across restarts.
- [x] First-run Issue creation; controller discovery of client Issues.
- [x] Heartbeat comments and online/offline calculation.
- [x] Terminal client selection and fixed diagnostic task menu.
- [x] Structured task/result messages correlated by `TaskID`.
- [x] Allowlisted actions: `ping`, `get_hostname`, `get_os`, `get_user`.
- [x] Cross-machine test: Mac controller ↔ separate physical Windows client.
- [x] Separate GitHub credentials on operator and client machines.
- [x] Temporary `per_page=100` workaround for the 30-comment truncation bug.

These reflect the **last confirmed conversation milestone**, not a fresh source-code audit. The terminal workflow was confirmed; a completed web UI was not independently verified in the later code walkthroughs.

## Current focus — make comment transport correct and API-efficient

### P0 — Comment pagination correctness
- [x] Audit every `listComments()` caller (`pollTasks`, `discoverClients`, `waitForResult`); also inspect the unused `pollComments()` loop.
- [x] Support pagination beyond 100 comments by following GitHub `Link` `rel="next"` rather than stopping after the first page.
- [x] Preserve ordering and expose errors rather than silently returning incomplete history.
- [x] Test mocked Issues with 0, 1, 30, 99, 100, 101, 250, and 251 comments and multi-page failures.
- [ ] Confirm all three consumers see newer tasks, heartbeats, and results.

**Acceptance:** A new comment on an Issue with more than 100 historical comments is no longer invisible. No unrelated transport redesign.

**Verification (October 30, 2026):** Pagination is implemented with explanatory code comments. In-memory HTTP tests verify ordered complete retrieval, task/heartbeat/result fixtures beyond page one, missing/malformed links, repeated links, response-body closure, and subsequent-page HTTP/network/JSON failures. `gofmt`, `go test ./...`, `go vet ./...`, and `go build -o /private/tmp/gundo-pagination-verification-20261030 ./...` passed using the local toolchain with network dependency lookups disabled and `GITHUB_TOKEN` removed from the check processes. No real credentials or live GitHub requests were used. End-to-end consumer confirmation on the existing Mac-controller/Windows-client environment remains pending; the fixtures test retrieval and message parsing, not execution of the consumer loops.

### P1 — Incremental and conditional polling
- [x] Design per-consumer in-memory bookmarks so routine polls do **not** reread every historical page.
- [ ] Preserve comment/task delivery semantics across polling boundaries and restarts; investigate offline tasks and result retry behavior.
- [x] Evaluate GitHub comments `since` with update-time semantics and overlap/ID filtering safeguards.
- [ ] Use `ETag` / `If-None-Match` and handle `304 Not Modified` where request semantics permit.
- [x] Test zero-change polls, simultaneous timestamps, repeated/changed old IDs, and pagination during active posting.

**Acceptance:** Existing comment history does not make steady-state API usage grow linearly with Issue age, and newly posted messages are not missed.

**Phase 2 verification (October 30, 2026):** Implemented `since` filtering with a one-second overlap before the newest seen comment's GitHub `created_at`, retaining the existing comment-ID check. Bookmarks advance only after complete successful retrieval and metadata validation, independently for task polling, each discovery mailbox, and each result wait. Startup still skips existing tasks; bookmarks are not persisted. Discovery retains its latest valid heartbeat and recalculates its age. Result waiting starts from the task POST response's ID/timestamp, avoiding a separate baseline read. Mocked tests cover same-second arrivals, empty mailboxes, missing timestamps, 100/101/250/1,000 new comments, failed-page recovery, task filtering, cached heartbeat expiry, and immediate results. A request-count test verifies ten initialization requests for 1,000 comments followed by one request per recent/quiet poll. `gofmt`, `go test ./...`, `go vet ./...`, `go build -o /private/tmp/gundo-incremental-verification-20261030 ./...`, and `git diff --check` passed. Check processes had `GITHUB_TOKEN` removed and dependency network lookups disabled; no real credentials or live GitHub requests were used. Manual Mac/Windows verification remains pending. ETags (Phase 3) and rate-limit resilience (Phase 4) are not implemented.

### P2 — Rate-limit and HTTP resilience
- [ ] Centralize/reuse an `http.Client` with request timeouts and appropriate cancellation.
- [ ] Handle `401`, `403`, `404`, `429`, `5xx`, and network failures distinctly.
- [ ] Respect `Retry-After`, rate-limit remaining/reset headers, and secondary-limit guidance; avoid busy retries.
- [ ] Use bounded backoff with jitter when appropriate; do not blindly retry non-idempotent POSTs that may have succeeded.
- [ ] Review heartbeat frequency and total comment POST volume across clients; consider reducing unnecessary writes.
- [ ] Add mocked tests for throttling, exhaustion, recovery, and timeout paths.

**Acceptance:** The client/controller degrade predictably during GitHub errors or rate limits, without tight loops, duplicate operations, or silent failures.

## Remaining v1 reliability and polish

### Task and client lifecycle
- [ ] Give `waitForResult()` a timeout/cancellation and clear error messages; avoid indefinite blocking.
- [ ] Confirm task processing semantics at restart: current startup baseline may skip tasks sent while the client was offline.
- [ ] Avoid losing task results when a POST fails; evaluate idempotent TaskID processing and durable last-seen progress.
- [ ] Make `saveState()` robust (validated/atomic writes and appropriate local permissions); consider Issue-created/state-save-failed recovery.
- [ ] **Later:** handle a genuinely deleted/invalid mailbox: keep ClientID, replace IssueNumber only after confirmed missing-Issue detection, and persist updated state. Never recreate on temporary GitHub failures.
- [ ] Handle stale/offline clients and old Issues cleanly; improve refresh/status UX.

### GitHub discovery and credentials
- [ ] Review `listIssues()` pagination, issue filtering, and behavior when there are more than 100 mailbox Issues.
- [ ] Add clear startup checks for missing/invalid `GITHUB_TOKEN` and insufficient repo permissions.
- [ ] Document minimum fine-grained PAT permissions, expiration, rotation, and revocation; avoid tokens in logs/repo.
- [ ] Choose deployment-time secrets provisioning approach (environment, protected secret store, or Ansible Vault if used). Evaluate GitHub App authentication only when warranted.

### Deployment, quality, and operator UX
- [ ] Build reproducible client/controller binaries for target operating systems; document supported architectures and setup.
- [ ] Add automated tests for API pagination, message correlation, heartbeat status, restart/state handling, and failure cases.
- [ ] Establish baseline build/test checks in Codex/CI (`gofmt`, `go test ./...`, `go vet ./...`, `go build ./...`).
- [ ] Improve friendly client labels, CLI output, and error visibility.
- [ ] Confirm scope and implementation status of the simple web dashboard; complete it if still pending, after transport reliability.
- [ ] Update README with actual run, token, test, and troubleshooting steps.

## After v1 — ideas, not commitments
- Modular transport interface with additional authorized SaaS adapters.
- More capable operator UI and centralized task history/monitoring.
- MCP integration / wearable interface exploration, beginning with low-risk read-only status operations.
- Assisted transport-adapter development, contingent on security review and a clear use case.

## Working with Codex
1. Ask Codex to read `AGENTS.md`, this file, and relevant docs and **inspect the live code**.
2. For the current checkbox, request a read-only investigation and a concise implementation/test plan.
3. Review the plan and approve a bounded scope.
4. Have Codex implement and run observable tests; inspect its diff and explanation.
5. Check completed items and record any new architectural decision in `docs/DECISIONS.md`.

**Current starting point:** Pagination and Phase 2 incremental polling are implemented and automated checks pass; live Mac/Windows verification remains pending. Stop here: conditional requests (Phase 3) and rate-limit resilience (Phase 4) each require separate approval before implementation. Deleted-Issue recovery is intentionally later.
