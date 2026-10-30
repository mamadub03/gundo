# Gundo — Architecture Decisions

> This file records decisions we actually discussed and the rationale behind them. It is not a feature wish list. Verify exact implementation against the current code before making changes.

## D1 — One GitHub Issue per client (adopted)
**Decision:** Each client installation gets a dedicated Issue that serves as its GitHub mailbox. **Why:** Client messages and lifecycle are separated without a shared global comment stream. **Do not:** replace this with a single shared Issue as a casual simplification.

## D2 — Persistent client identity, separate mailbox identity (adopted)
**Decision:** `ClientID` is stable across ordinary process restarts; `IssueNumber` identifies the remote mailbox; both live in client-local state. **Why:** Restarting a process should not create a new logical client. A lost/deleted Issue and a lost local identity are different failures. **Implication:** Future mailbox recovery keeps `ClientID` and replaces only `IssueNumber` after a verified missing-Issue condition.

## D3 — Creating the Issue is enrollment (adopted)
**Decision:** A successful initial `createIssue()` establishes the client's mailbox and discoverability. **Why:** A separate `register()` message would duplicate what Issue creation already establishes. **Qualification:** Enrollment is not proof of current availability; heartbeats supply liveness.

## D4 — Controller discovers clients remotely (adopted)
**Decision:** The controller discovers client Issues and interprets heartbeats from GitHub rather than reading any remote client's `state.json`. **Why:** The controller and client can run on separate systems with separate credentials and local files.

## D5 — Three identifier roles, not one (adopted)
**Decision:** Keep `ClientID` (installation), `IssueNumber` (mailbox), `TaskID` (logical operation), and GitHub comment ID (transport event) conceptually distinct. **Why:** Results must correlate with the right task; deduplication of comments is not the same as deduplication of tasks.

## D6 — Async SaaS transport and logical layer separation (adopted direction)
**Decision:** The MVP exchanges instructions/results through GitHub's API rather than maintaining a direct controller-to-client socket. Operations, transport, and client execution have separate responsibilities. **Why:** A transport adapter should eventually be replaceable without rewriting the operator workflow or diagnostic action handler. **Current limit:** A complete plug-in adapter interface/multi-SaaS suite is a *future design goal*, not a verified implementation.

## D7 — Fixed diagnostic actions for the MVP (adopted)
**Decision:** Initial client actions are `ping`, `get_hostname`, `get_os`, and `get_user`; unsupported actions fail cleanly. **Why:** It gives a testable task/result lifecycle without introducing arbitrary remote execution or destructive operations. Maintenance tasks must preserve this boundary.

## D8 — Credentials belong outside client state (adopted)
**Decision:** Use an externally supplied GitHub credential (`GITHUB_TOKEN` in the current design), with separate credentials per operator/client as appropriate. **Why:** Identity state should not hold secrets. **Later options:** improved provisioning, rotation, minimum scopes, and potentially GitHub App installation tokens; not assumptions about the current implementation.

## D9 — Correctness before API optimization (current engineering sequence)
**Decision:** Fix truncated comment retrieval first, then incremental/conditional polling, then comprehensive rate-limit/retry behavior. **Why:** `per_page=100` alone fails on larger Issues, while naive pagination alone can increase request volume. The final design must avoid missed messages **and** repetitive history scans.

## D10 — Preserve learning and review in Codex workflow (adopted)
**Decision:** Use `AGENTS.md` + architecture/decisions/plan as the only permanent development scaffold. Agent workflow: inspect → propose → review → implement → test → explain diff. **Why:** The agent should understand the existing system without requiring giant pasted chat histories, while the developer retains control and learns the changes. No per-feature harness directory is required.

## Not yet adopted as completed v1 functionality
A polished web dashboard, alternative SaaS transports (Calendar/Sheets/Reddit/etc.), generalized plugin system, MCP/Meta-glasses controller, AI-generated transport adapters, automated deployment and deleted-Issue recovery were discussed as possibilities or follow-up work. **Do not describe them as shipped without checking the live repository.**
