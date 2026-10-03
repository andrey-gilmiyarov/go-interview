# Astra/Sol agent workflow

This is the canonical workflow for repository changes. The user selects the
root model and reasoning effort in the app for each session. GPT-6 Astra uses
Sol workers; every other root model performs the complete task itself.

## Model routing

Routing applies only to the root chat. An already-dispatched Sol worker follows
its task contract and remains subject to Astra review.

| Active root model | Workflow |
| --- | --- |
| `gpt-6-astra` | Astra plans, dispatches `sol-worker`, reviews, and accepts. |
| Any other model, including Sol, Terra, or Luna | The selected model researches, designs, implements, validates, and reviews itself; no subagents. |
| Model not reliably known | Independent execution without delegation. |

Determine the active root model from reliable current session/model context.
Do not infer it from project worker defaults, a previous chat, a model named in
an ordinary task, or unsupported self-identification. Re-evaluate routing when
reliable session context establishes a model change. The independent mode must
not spawn implementers, explorers, or reviewers, or create chats as a substitute.
This repository policy takes precedence over skill recommendations to delegate;
direct user instructions take priority.

The route is an instruction policy: `agents.enabled = true` makes tools
available, but TOML does not conditionally disable them based on the root model.
The fallback model controls an actual spawn, not whether delegation is allowed.
A runtime-enforced gate would require host support beyond this configuration.

## Responsibilities

Astra owns the overall requirements, plan, task decomposition, dependencies,
acceptance criteria, cross-task architecture, escalation, review, and final
integration. Astra delegates routine scoped implementation to Sol; read-only
answers and repository research do not require delegation.

Sol owns research and implementation for its delegated task. It checks
prerequisites, inspects relevant callers and stored-data contracts, chooses an
approach, implements in its owned files, updates relevant tests/docs, validates,
reviews its diff, and reports exact outcomes.

Within scope and unless the contract restricts authority further, Sol may make
compatible architecture, API, contract, and schema decisions independently.
This includes choosing among materially different implementation approaches.
Sol explains significant choices, alternatives, tradeoffs, and compatibility
validation in its report. Individual approval is not required for each
compatible choice.

Sol must preserve approved product requirements, task constraints, existing
caller and stored-data compatibility, and observable guarantees. It must not
expand scope, change cross-task architecture contrary to Astra's plan, add
software dependencies without explicit user authorization, perform unrelated
cleanup, or spawn/delegate to other agents.

In independent execution mode, the selected root model owns all these stages
itself within the user's authorization and the repository's product boundaries.
The delegated-task acceptance and escalation rules below apply to Astra/Sol
work, not to a separate Astra reviewer in independent mode.

## Compatibility approval

Sol obtains explicit Astra approval before implementing incompatible changes.
Examples include removing or renaming public API fields, changing required
input/output semantics, invalidating persisted data, destructive schema
migrations, and weakening existing behavioral guarantees. If compatibility
cannot be established, Sol escalates rather than assumes it.

The proposal identifies affected callers/data, alternatives, and a migration
or compatibility plan. Astra's approval must appear in the task contract or
direct rework instructions with scope and migration/compatibility requirements.
Once approved, Sol can implement that change without repeated approval for
already-settled decisions. Astra approval cannot override the user's product
boundaries or replace explicit user authorization for dependencies.

## Lifecycle and ownership

For non-trivial work, the responsible root model completes research, design,
and planning before execution. If the user requests plan approval or an
applicable rule requires approval, execution waits for that approval; otherwise
continue within the authorized scope.

```text
Astra mode:
USER -> ASTRA RESEARCH / DESIGN / PLAN -> APPROVAL (when required)
  -> ASTRA SELECTS READY TASK -> SOL RESEARCH / IMPLEMENTATION -> VALIDATION
  -> READY_FOR_REVIEW -> ASTRA REVIEW
       -> REWORK -> SAME SOL SESSION -> VALIDATION -> READY_FOR_REVIEW
       -> ACCEPTED -> ASTRA SELECTS NEXT READY TASK
  -> ASTRA FINAL INTEGRATION REVIEW (after all tasks are accepted)

Independent mode:
USER -> SELECTED MODEL RESEARCH / DESIGN / PLAN -> APPROVAL (when required)
  -> IMPLEMENTATION -> VALIDATION -> SELF-REVIEW -> HANDOFF
```

Delegated task states are `PLANNED`, `READY`, `IN_PROGRESS`, `REVIEW`, `REWORK`,
`BLOCKED`, `ACCEPTED`, and `CANCELLED`. The report status `READY_FOR_REVIEW`
moves the task to Astra `REVIEW`; only Astra marks delegated work accepted.
Independent tasks may run in parallel only after prerequisites are satisfied
and file ownership does not overlap. The same Sol session owns a task through
ordinary review/rework; replace it only if unavailable or Astra intentionally
replaces the task.

## Plan file convention

The responsible root model creates a new, uniquely named
`YYYY-MM-DD-topic.md` file in `docs/plans/` for each plan. Never create or overwrite a shared
`docs/active-plan.md`. Link every plan from [docs/README.md](README.md), and
keep a completed plan at the same path while updating its status. This
repository convention overrides skill and tool defaults for plan paths,
including `docs/active-plan.md` and `docs/superpowers/plans/`.

## Task contract

Every Astra dispatch supplies a complete contract:

```yaml
task:
  id: T3
  title: Short scoped objective
  objective: Exact behavior to implement
context:
  overall_design: Astra's architecture and product constraints
  relevant_components: [paths or packages]
scope:
  allowed: [owned files or directories]
  forbidden: [out-of-scope files or behaviors]
decision_authority:
  autonomous: [compatible architecture, API, contract, and schema choices in scope]
  requires_astra_approval: [breaking changes and cross-task architecture changes]
  approved_breaking_changes: [] # Record explicit approvals and migration requirements here.
constraints: [compatibility, dependency, and safety rules]
dependencies: [prerequisite task IDs]
acceptance_criteria: [observable outcomes]
validation: [commands to run]
escalation_conditions: [conditions requiring Astra]
```

Respect direct user instructions and repository product/change boundaries.
Within them, direct Astra task/rework instructions take priority, followed by
the contract, applicable `AGENTS.md`, repository docs, and existing conventions.
The report lists changed files, implementation, significant decisions and
compatibility evidence, approved breaking changes, every validation outcome,
assumptions, deviations, and blockers.

## Escalation, review, and rework

Sol returns `BLOCKED` for affected work when requirements have material
ambiguity, an incompatible change lacks approval, compatibility cannot be
established, scope must expand, cross-task architecture or file ownership
conflicts, a dependency lacks user authorization, prerequisites are missing,
or tests invalidate the agreed constraints. Sol preserves safe partial work,
continues independent safe work where possible, and explains the evidence and
smallest decision needed from Astra. Compatible choices in scope are not
blockers by themselves.

Astra reviews the actual diff, contract scope, architecture fit, compatibility,
error handling, concurrency/transaction semantics, tests, security, and diff
cleanliness. Passing tests are evidence, not acceptance. Ordinary feedback goes
through `followup_task` to the same Sol session. Sol addresses must-fix items,
reruns validation, and returns an updated `READY_FOR_REVIEW` report. Feedback
does not implicitly authorize breaking changes or scope expansion. Astra
reviews again and performs the final cross-task integration review.

## Repository runtime

The project configuration is `.codex/config.toml`; the active worker profile
is `.codex/agents/sol-worker.toml`, named `sol-worker`. The project does not pin
root `model` or `model_reasoning_effort`.

Agents are enabled, with `max_concurrent_threads_per_session = 4` excluding
the primary thread. Fallback routing uses `gpt-6.1-sol` / `high`. The explicit
`[agents.sol-worker]` registration resolves `agents/sol-worker.toml` relative
to `.codex/config.toml`; the profile also pins `gpt-6.1-sol` / `high`.

Astra dispatches `agent_type="sol-worker"` with `fork_turns="none"` and the
complete contract. `fork_turns` is a `spawn_agent` argument, not a TOML key:
it omits the parent's conversation history, not the shared repository
workspace. Workers must preserve user edits, other workers' edits, and staging
state. Runtime settings support this policy without enforcing model-dependent
routing, scope, planning, review, or acceptance.

The previous Luna profile is preserved in
[the archived profile](../.codex/archived-agents/luna-worker.toml), outside the
`.codex/agents/` discovery directory, and is not registered in project config.
These settings configure development agents; documentation and harness changes
keep product behavior unchanged.

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
