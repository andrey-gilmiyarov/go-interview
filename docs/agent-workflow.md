# Lead/Luna agent workflow

The root agent is the Lead: Astra is recommended when selected in the ChatGPT
app, while the root model and reasoning effort remain per-session choices. The
Lead owns requirements, research, architecture and product decisions, planning,
dispatch, review, and final integration. Luna implements one bounded task.

## Responsibilities and decision boundary

The Lead resolves requirements, selects and records architecture, product,
public API, and schema decisions, checks dependencies, defines acceptance
criteria, and owns review and integration. Luna may implement those decisions
only when the task contract explicitly approves them and includes them in
scope. Luna returns `BLOCKED` if an unexpected decision or out-of-scope change
is required. Do not add software dependencies without explicit user
authorization.

Luna reads the complete contract, inspects relevant context, implements within
its ownership, runs the requested validation, checks the diff, and reports
exact outcomes. Luna implements only approved decisions within contract scope.
It does not independently select architecture, alter requirements or public
contracts beyond scope, perform unrelated cleanup, or delegate to subagents.

## Lifecycle and ownership

```text
USER -> RESEARCH -> DESIGN -> PLAN
  -> APPROVAL (only when requested or required) -> LUNA EXECUTION
  -> VALIDATION -> READY_FOR_REVIEW -> LEAD REVIEW
LEAD REVIEW -> REWORK -> LUNA EXECUTION (same session via followup_task)
LEAD REVIEW -> ACCEPTED -> FINAL INTEGRATION REVIEW
```

For non-trivial work, the Lead completes research, design, and planning before
execution. If the user requests plan approval, or an applicable rule requires
approval, wait before execution; otherwise skip that gate. A Luna
`READY_FOR_REVIEW` report maps the task to REVIEW. The Lead reviews the actual
diff and evidence. If changes are needed, the Lead sends REWORK to the same
worker with `followup_task`; Luna fixes the review items, validates again, and
returns an updated report. The Lead reviews again. Only the Lead may mark work
ACCEPTED, after which the Lead performs the final integration review.

Task states are `PLANNED`, `READY`, `IN_PROGRESS`, `REVIEW`, `REWORK`,
`BLOCKED`, `ACCEPTED`, and `CANCELLED`.

## Plan file convention

The Lead creates a new, uniquely named `YYYY-MM-DD-topic.md` file in
`docs/plans/` for each plan. Never create or overwrite a shared
`docs/active-plan.md`. Link every plan from [docs/README.md](README.md), and
keep a completed plan at the same path while updating its status. This
repository convention overrides skill and tool defaults for plan paths,
including `docs/active-plan.md` and `docs/superpowers/plans/`.

Parallel tasks may run only after their dependencies are satisfied and when
their file ownership does not overlap. The Lead serializes or coordinates
changes to shared files. Every task contract states exact ownership, and each
worker preserves other workers' edits and staging state.

## Task contract and dispatch

Dispatch implementation tasks explicitly with `agent_type="luna-worker"` and
`fork_turns="none"`. The contract must be complete enough to execute without
guessing:

```yaml
task:
  id: T1
  title: Short bounded objective
  objective: Exact behavior to implement
context:
  approved_decisions: [chosen approach and constraints]
  relevant_components: [paths or packages]
ownership:
  files: [exact paths this worker may change]
scope:
  forbidden: [out-of-scope files or behaviors]
constraints: [compatibility, dependency, and safety rules]
dependencies: [prerequisite task IDs and their state]
acceptance_criteria: [observable outcomes]
validation: [commands to run]
escalation_conditions: [conditions requiring the Lead]
```

The Lead confirms prerequisites are satisfied before dispatch and obtains
explicit user authorization before adding software dependencies. The contract
is followed by applicable `AGENTS.md`, repository docs, and existing code
conventions. Reports include changed files, implementation summary, each
validation command and result, assumptions, deviations, and blockers.

## Escalation and review

Luna returns `BLOCKED` rather than guessing when requirements conflict, an
unapproved architecture/product/API/schema decision is needed, a prerequisite
is missing, tasks conflict, validation invalidates the approved design, or
scope must expand. The report gives evidence, safe partial work, and the
smallest decision needed from the Lead.

The Lead reviews scope, design fit, error handling, edge cases, compatibility,
tests, maintainability, and diff cleanliness. Passing checks are evidence, not
acceptance. The Lead retains review and acceptance authority; Luna owns
implementation through ordinary review and rework. The Lead sends fixes to the
same Luna session; replacement is reserved for an unavailable worker or an
intentional Lead decision.

## Runtime configuration

`.codex/config.toml` registers the explicit role as `luna-worker`. The
`default_subagent_model` and `default_subagent_reasoning_effort` settings are
fallbacks for spawned agents and do not replace explicit implementation
dispatch. `.codex/agents/luna-worker.toml` keeps the worker on `gpt-6-luna`
with `max` reasoning. The project does not pin the root model or reasoning
effort; the app selects those per session. Runtime settings support the policy
but do not replace it.

## Validation and change boundaries

Choose checks that exist in this repository and fit the task. For documentation
or configuration changes, parse affected TOML, check local Markdown links, and
inspect staged and unstaged diffs. For Go changes, run the smallest relevant
checks, then `go test ./...` and `go build ./...` when applicable. Report
pre-existing exercise failures separately and do not fix them outside scope.
This repository has no standard `make` targets or pinned linter. Report every
command and its exact outcome.

Preserve user edits and staging state. Do not reset, checkout, commit, or add
dependencies without explicit authorization. Keep product behavior unchanged
for documentation or harness-only tasks. Do not claim a check passed unless it
was run and succeeded.
