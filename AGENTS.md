# Repository agent map

This repository uses a hierarchical Lead/Luna workflow. Astra is the
recommended Lead model when selected in the ChatGPT app; the root model and
reasoning effort remain per-session choices. Luna implements bounded tasks.
The workflow is defined in [docs/agent-workflow.md](docs/agent-workflow.md).

## Operating rules

- The Lead owns requirements, research, architecture and product decisions,
  task decomposition, dependencies, acceptance criteria, dispatch, and review.
- For implementation, dispatch explicitly with `agent_type="luna-worker"` and
  `fork_turns="none"`. Give Luna a complete contract with objective, approved
  decisions, exact file ownership, constraints, dependencies, criteria, and
  required validation. Runtime defaults do not replace explicit role dispatch.
- The Lead decides architecture, product, public API, and schema changes. Luna
  may implement those decisions only when they are explicitly approved in the
  scoped contract. Unexpected or out-of-scope changes require `BLOCKED`.
- Do not add software dependencies without explicit user authorization.
- Parallel tasks require satisfied dependencies and non-overlapping file
  ownership. The Lead serializes or coordinates shared-file changes. Preserve
  other workers' edits and staging state.
- For non-trivial work, follow RESEARCH -> DESIGN -> PLAN. Wait at APPROVAL
  only when the user requests it or an applicable rule requires it.
- A Luna `READY_FOR_REVIEW` report moves the task to REVIEW. If changes are
  needed, send REWORK to the same worker with `followup_task`; do not replace
  the worker for ordinary fixes. The Lead alone marks work ACCEPTED, then
  performs the final integration review.
- Read-only answers and repository research do not require delegation.

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
