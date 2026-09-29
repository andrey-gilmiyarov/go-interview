# Senior Go Handbook implementation plan

Approved by user 2026-09-27. Spec: ../specs/2026-09-27-go-handbook.md.

## Global constraints
Russian self-contained two-level articles; VitePress 1.6 + Vue 3; Go 1.27.1 stdlib + x/sync only. Local static site without code execution, accounts or progress. No commits, staging, resets, new unrelated dependencies or publication. Preserve user edits. Lead owns contracts/review; explicitly dispatch luna-worker, fork_turns none. Workers must not spawn subagents.

## Task 1: Site foundation
Create root npm commands and pinned dependencies, metadata-driven navigation/search, theme, content checker/node tests, first slices article. localhost only, no remote required assets. Search must strip hidden answers/code solutions. Accept after build and focused tests.

## Task 2: Content and source contracts
42 topics across language(7), api(5), concurrency(7), runtime(5), stdlib(6), testing(5), patterns(4), tooling(3). Source catalog and complete coverage mappings for 100 Go Mistakes and 50 Shades. No raw bookmark export or authentication URLs.

## Task 3: Practice and migration
20 exercises: ten trace/explain and ten implement/debug. Starter tests exercise build tag, solutions same shared checks untagged; compile clean stubs. Migrate old scenarios after tests pass, map every old source including extensionless middleware main. Replace Gin/Resty/Testify with stdlib. Keep deliberate unsafe fragments outside default build.

## Task 4: Language and API
Twelve substantial articles plus ten trace exercises with version-aware guarantees and allowed nondeterminism.

## Task 5: Concurrency and runtime
Twelve substantial articles and bounded concurrent exercises with cancellation, invalid limits, empty inputs, failures and lifecycle tests.

## Task 6: Backend library, testing, patterns, tooling
Eighteen substantial articles and remaining implementation exercises. httptest, reproducible benchmarks, no public-site network in checks.

## Task 7: Five models
Slices, interfaces/typed nil, defer, channels/select and worker cancellation. Vue+SVG, explicit scenarios/assumptions, step/reset controls and accessible state explanations. Pure state logic covered by node:test.

## Task 8: Integration and review
42 complete topics, 20 exercises, 5 models. Validate metadata/local links, build, browser search/spoilers/navigation/keyboard/narrow layout/themes. go test ./..., go test -race ./..., go build ./..., go vet ./..., npm run docs:check, npm run test:site, npm run docs:build, git diff --check, git diff --cached --check, status. Record exact outcomes. Backend sections remain roadmap, not implemented prematurely.
