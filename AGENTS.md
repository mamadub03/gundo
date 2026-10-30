# Gundo — Repository Instructions for Coding Agents

## Purpose
Gundo is a Go controller/client prototype for **authorized collegiate red-team/blue-team exercises**. Its current transport is GitHub Issues and issue comments. Longer-term goals include a modular transport layer and a web operator interface, but **do not assume those are implemented**.

Read `docs/ARCHITECTURE.md` for the system, `docs/DECISIONS.md` for settled choices, and `PLANS.md` for priorities. **Live source code and tests take precedence over historical documentation.** If the docs disagree with the repository, report the discrepancy and propose a correction.

## Non-negotiable project conventions
- **One client installation owns one GitHub Issue/mailbox.** Do not consolidate clients into a shared Issue.
- `ClientID` is the persistent identity of the client; `IssueNumber` is its mailbox; `TaskID` correlates a task and its result; GitHub comment IDs are transport-level IDs.
- Initial Issue creation is client enrollment. Do not invent a redundant registration workflow without a demonstrated need.
- Controller/client communicate indirectly through GitHub. The controller does **not** need the client's local state file.
- Retain the existing fixed, non-destructive diagnostic action allowlist unless a reviewed scope change explicitly calls for otherwise. Do not introduce remote shell, persistence, evasion, or destructive actions as part of maintenance work.
- Do not embed GitHub tokens in code, test fixtures, tracked configuration, logs, or `state.json`. Prefer environment-provided credentials (`GITHUB_TOKEN` in the current design).
- Treat GitHub errors distinctly: `404` is not equivalent to `401`, `403/429`, timeout, or `5xx`. Do not turn temporary failures into mailbox recreation or duplicate operations.

## How to work in this repository
1. **Inspect before editing.** Locate relevant functions, their callers, tests, and current behavior. Do not assume filenames or behavior from these docs are fully up to date.
2. **Explain the problem and propose the smallest plan** for meaningful changes. Name affected files, behavior changes, tests, risks, and non-goals. For architectural changes, wait for the developer's approval.
3. **Work in small, reviewable increments.** Avoid unrelated refactors, invented abstractions, or sweeping file reorganizations.
4. **Test what you change.** Run appropriate checks, such as `gofmt`, `go test ./...`, `go vet ./...`, and `go build ./...`; report what actually ran and what failed. Use mocked HTTP/API responses for transport tests. Do not use live tokens or create real GitHub issues in routine tests.
5. **Show the diff and explain why.** Summarize the execution path affected, important Go concepts, failure cases, and any tradeoffs so the developer can learn the implementation.
6. **Update documentation selectively.** Change `PLANS.md` when a milestone changes; update architecture or decisions only for durable changes. Do not generate a separate plan file or harness for each feature.
7. **Stop at the requested scope.** Do not automatically proceed to the next backlog item after finishing a task.

## First task in a new Codex session
Read these four files, inspect the actual repository, and identify documentation/code mismatches before implementing a substantial change. The current reliability priority is **correct comment pagination followed by incremental/conditional polling and rate-limit handling**; deleted-Issue recovery is later.

## Safety and operating discipline
- Only test on systems, repositories, and accounts explicitly authorized for the exercise.
- Never expose credentials or private repository data in summaries or logs.
- No production deployment, destructive Git actions, remote execution, or external service mutations without an explicit request and appropriate review.
