# Repository agent map

This repository selects its workflow from the chat's active root model:
`gpt-6-astra` uses Astra/Sol delegation; any other root model performs all work
itself without subagents. Root model and reasoning effort remain user-selected
per session. The detailed routing, task contract, and lifecycle are in
[docs/agent-workflow.md](docs/agent-workflow.md).

## Operating rules

- Routing applies to the root agent, not to an already-dispatched worker.
  Use reliable current session/model context; do not infer the root model from
  worker defaults, previous chats, or self-identification without evidence.
  If the active root model is unknown, work independently without delegation.
- Only an Astra root uses the subagent workflow. Every other root model owns
  research, design, implementation, validation, and review itself. Do not
  spawn implementers, explorers, or reviewers, or create other chats as a
  substitute for subagents in this mode. This policy takes precedence over
  skill recommendations to delegate; direct user instructions take priority.
- Astra owns the overall plan, task decomposition, dependencies, acceptance
  criteria, cross-task architecture, escalation, review, and final acceptance.
  Routine scoped implementation is delegated to `sol-worker`; read-only
  answers and repository research do not require delegation.
- For non-trivial changes the responsible root model follows
  RESEARCH -> DESIGN -> PLAN -> EXECUTION -> VALIDATION -> REVIEW
  -> FINAL INTEGRATION REVIEW. Wait at APPROVAL only when the user requests
  it or an applicable rule requires it.
- Every Sol task has an explicit scope, constraints, dependencies, decision
  authority, acceptance criteria, and required validation.
- Astra dispatches implementation with `agent_type="sol-worker"` and
  `fork_turns="none"`, supplying a self-contained task contract. This isolates
  conversation history; the worker still shares the repository workspace.
- Sol may research and choose architecture, API, contract, and schema changes
  independently within task scope when they preserve compatibility and the
  approved product requirements. Sol explains decisions and validation in its
  report. Breaking changes require explicit Astra approval before implementation;
  approved breaking changes must be recorded in the task contract or rework
  instructions. Sol must not expand scope or delegate further. Adding
  dependencies still requires explicit user authorization.
- Astra reviews the actual diff and test evidence; only Astra marks delegated work
  accepted.
- Review fixes return through `followup_task` to the same Sol worker session.
  Do not replace the worker for ordinary rework.
- Sol escalates material requirement ambiguity, unapproved breaking changes,
  cross-task architecture conflicts, required changes beyond scope, unauthorized
  dependencies, or failures that invalidate the agreed constraints. Compatible
  decisions within task scope do not require individual approval.

## Plan file convention

- The responsible root model creates a new, uniquely named
  `YYYY-MM-DD-topic.md` file in `docs/plans/` for each plan; never create or overwrite a shared
  `docs/active-plan.md`.
- Link every plan from [docs/README.md](docs/README.md). Keep a completed plan
  at the same path and update its status.
- This repository convention overrides any skill or tool default for plan
  paths, including `docs/active-plan.md` and `docs/superpowers/plans/`.

## Repository map

- [Agent workflow](docs/agent-workflow.md) — roles, task contract, lifecycle,
  rework, escalation, and validation.

## Validation

Choose checks that exist in this repository and fit the task. For documentation
or configuration changes, parse affected TOML, check local Markdown links, and
inspect `git diff --check`, `git diff --cached --check`, and status. For Go
changes, run the smallest relevant checks, then `go test ./...` and
`go build ./...` when applicable. Report existing exercise failures separately
and do not fix them outside scope. Report every command and its exact outcome.
This repository has no standard `make` targets or pinned linter.

## Change boundaries

Preserve user edits and staging state. Do not reset, checkout, commit, or add
dependencies without explicit authorization. Keep product behavior unchanged
for documentation or harness-only tasks. Do not hide unrelated cleanup in a
task.
