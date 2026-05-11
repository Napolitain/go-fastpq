# AGENTS.md Instructions For go-fastpq

## Project Profile

`go-fastpq` is a Go library package that provides bucket-based priority queues for bounded, fill-then-drain, and sparse priority workloads. Lower numeric priorities are higher priority, and equal-priority values must preserve FIFO order.

- Project shape: Go library/package.
- Selected language: Go.
- Minimum module target: Go 1.22 from `go.mod`.
- Current CI runtime: Go 1.26.x in `.github/workflows/ci.yml`.
- Bootstrap source used during generation: Go program-tester guidance from the temporary bootstrap bundle. The bundle is intentionally removed after bootstrapping.

## Default Workflow

1. Read `README.md`, `BENCHMARK.md`, `go.mod`, `.golangci.yml`, and the touched Go files before changing behavior.
2. Identify which queue variant is affected: `Queue`, `BulkQueue`, `FillDrainQueue`, `SparseQueue`, shared bucket helpers, tests, benchmarks, CI, or docs.
3. State any material uncertainty before editing. Ask the user only for facts that affect API shape, compatibility, correctness semantics, benchmark targets, or performance constraints.
4. Preserve the public contracts documented in `README.md`: fixed queues reject invalid priority counts, fixed-priority APIs constrain priorities to `[0, N)`, sparse queues accept non-negative priorities, equal priorities are FIFO, and queues are not synchronized.
5. Encode important claims as tests, table cases, benchmarks, or documented quality gates instead of prose-only assertions.
6. Keep changes narrow. Do not mix unrelated queue behavior, benchmark changes, and docs cleanup unless one change requires the other.
7. Use modern, actively maintained Go tooling that is compatible with the repository. Before changing Go versions, linter versions, benchmark tooling, or dependencies, verify current official release information and document the compatibility reason.

## Operational Guidelines

### Correctness And Invariants

- Treat priority ordering, FIFO stability, length accounting, empty-state behavior, and error identity as core invariants.
- Exercise boundary cases explicitly: zero or negative priority counts, negative priorities, `priority == 0`, `priority == N-1`, `priority == N`, empty queues, singleton queues, repeated priorities, sparse high priorities, and refill-after-drain flows.
- For queue variants with different contracts, test the difference directly. For example, `BulkQueue` rejects pushes during a non-empty drain, while `FillDrainQueue` optimizes for throughput and allows pushes during drain with caller-managed priority validity.
- Keep `errors.Is` compatibility for exported sentinel errors.
- If a hot-path implementation intentionally panics for invalid caller input, document that contract and cover it with tests.

### Go Implementation

- Prefer simple, allocation-aware Go over abstraction that obscures hot paths.
- Preserve generic type behavior and zero-value clearing where values might retain references.
- Keep public APIs small and documented. Exported identifiers require useful comments.
- Avoid adding dependencies unless they materially improve correctness, verification, or performance. If a dependency is needed, compare runtime cost, build cost, memory behavior, maturity, and maintenance status before adopting it.
- Do not add synchronization to queue types unless the task is explicitly about concurrency; the current contract says callers must protect shared queues.

### Tests

- Unit tests must be deterministic and assert exact behavior, not just absence of errors.
- Name tests after the invariant or boundary they defend.
- Prefer table tests when they make edge coverage clearer.
- Add regression cases for any discovered bug before or with the fix.
- When changing optimized behavior, compare against a simple oracle or existing queue variant where feasible.

### Benchmarks And Performance

- Performance is part of the API surface. Benchmark changes that affect push/pop complexity, memory retention, allocation behavior, sparse page management, bucket compaction, or fill-drain paths.
- Use the benchmark workload families documented in `BENCHMARK.md`: `FillDrain`, `SteadyState`, and `SparseReused`.
- Run benchmarks with `-benchmem` and inspect allocation changes.
- Keep benchmark dimensions compatible with the shared Go/C++ benchmark shape unless the task explicitly changes the benchmark model.
- If benchmark runtime is too high for an inner loop, reduce with `FASTPQ_BENCH_MAX_ITEMS` and state that the run is a smoke benchmark, not a full regression assessment.

### Tooling Policy

- Keep `go.mod` tidy and do not lower the declared Go version without a compatibility reason.
- CI currently pins `actions/setup-go` to `1.26.x`; when updating this, verify the current stable Go line from official Go sources.
- Use the repository-pinned linter command from `README.md` and CI unless deliberately updating tooling.
- Prefer `gofmt` for formatting because that is the current CI gate. Do not introduce `gofumpt` as a required gate unless CI and docs are updated together.

## Quality Gate Loop

Run this loop before closing implementation work:

1. Confirm the current task, changed files, and affected queue variant.
2. Identify correctness, compatibility, performance, and integration risks introduced by the change.
3. Add or update tests, documentation, benchmarks, or checks for those risks.
4. Run the selected gates below.
5. If any selected gate fails, fix the issue and rerun the failed gate plus any dependent gates.
6. Document commands that could not run, including the blocker and the command to run later.

### Selected Gates

| Gate | Command or check | Applies to | Required result |
| --- | --- | --- | --- |
| Format | `gofmt -l .` | Go source and tests | No files listed |
| Module tidy check | `go mod tidy` then `git diff --exit-code -- go.mod go.sum` | Module metadata | Module files stay tidy |
| Lint | `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 run ./...` | All packages and tests | Exits 0 |
| Build | `go build ./...` | All packages | Exits 0 |
| Tests and coverage | `go test -v "-coverprofile=coverage.out" "-covermode=atomic" ./...` | All packages | Exits 0 and writes coverage profile |
| Dead code | `go run golang.org/x/tools/cmd/deadcode@v0.45.0 -test ./...` | Package plus tests | Exits 0 |
| Benchmarks | `go test -run '^$' -bench 'Benchmark(FillDrain|SteadyState|SparseReused)' -benchmem` | Benchmark suite | Exits 0; no unexplained allocation or runtime regression |

When the full benchmark matrix is too slow for the current change, run a smoke benchmark with a documented limit, for example:

```bash
FASTPQ_BENCH_MAX_ITEMS=100000 go test -run '^$' -bench 'Benchmark(FillDrain|SteadyState|SparseReused)' -benchmem
```

### Omitted Gates

| Gate | Reason omitted |
| --- | --- |
| Property-based testing | Not selected for this bootstrap pass; add when invariants need generated-input coverage. |
| Fuzzing | Not selected for this bootstrap pass; no parser or hostile byte-input surface currently exists. |
| Mutation testing | Not selected for this bootstrap pass; no Go mutation tool is configured in the repo. |
| Race detection | Not selected for this bootstrap pass; queues are documented as unsynchronized and caller-protected. Run `go test -race ./...` if concurrency code is introduced. |
| Dependency and security audit | Not selected for this bootstrap pass. Run `govulncheck ./...` when adding or updating dependencies. |
| Formal/model checking | Not selected for this bootstrap pass; use small exact oracles in tests where practical. |

## Commit Message Policy

Create git commits automatically as work reaches coherent checkpoints unless the user explicitly asks not to commit, the worktree contains user-owned changes that cannot be safely separated, or verification has not reached a reasonable commit point. Keep commits granular: one commit per independently reviewable feature, fix, test, documentation update, refactor, performance change, build change, CI change, chore, or revert.

Every commit must use a Conventional Commits subject line and include a descriptive body. The body is mandatory because commit descriptions are used for internal logging.

Use this structure:

```text
<type>[optional scope]: <concise imperative subject>

Task:
- <what was requested and what changed>

Why:
- <why this matters for the repository or workflow>

Approach:
- <how the work was done and why that approach was preferred>

Verification:
- <commands run and results, or explicit reason verification was not run>
```

Valid subject types include `feat`, `fix`, `test`, `docs`, `refactor`, `perf`, `build`, `ci`, `chore`, and `revert`. Avoid vague subjects such as `fix`, `update`, or `changes`, and do not mix unrelated concerns in one commit.
