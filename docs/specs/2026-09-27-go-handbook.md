# Approved product specification

Personal Russian Senior Go/backend handbook. Minimal local VitePress site for reading; IDE for code. Short recap, mechanism, example, pitfalls/tradeoffs, hidden self-check, practice, sources per topic. No accounts, progress, editor, code runner or database. Sidebar sections, local search, page outline, light/dark themes, responsive layout. Assets local after npm install. Go first, Kafka/PostgreSQL next, then backend engineering, Redis/ELK, Docker, separately launched backend labs; Kubernetes/NoSQL/system design later.

Go release contains 42 topics in eight groups, 20 exercises (10 explain + 10 implement) and five bounded illustrative SVG models. Runtime details versioned; chosen schedule/capacity must not masquerade as language guarantees. Sources are selected and explained, not copied. Date of technical review only after verification. Retain source attribution and bookmark privacy.

Repository becomes site/, examples/, practice/ under original module. Existing material reorganized without archive, with a migration map. Starter tests require exercise tag; checked reference solutions default. Known bad snippets excluded from default package traversal. Canonical exercise README/source included by site build, not duplicated. No spoiler excerpts in search.

Approved dependencies: VitePress 1.6, Vue 3, x/sync. Installed Node 22.22.2 and Go 1.27.1. Pin resolved dependency versions. Node built-in tests and Go testing; no extra test framework. No commits. Execution in managed isolated checkout protects original main checkout.

Full acceptance and tasks are in the adjacent approved implementation plan; per-task ownership/contracts are recorded in the execution ledger.
