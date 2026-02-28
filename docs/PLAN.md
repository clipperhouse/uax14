# UAX #14 Implementation Plan

## Scope and starting point

- This repo is greenfield (`SPEC.md` + `go.mod`), so start by creating a minimal package structure aligned with `uax29` style and cleaner `displaywidth`-style generation.
- Anchor all behavior to UAX #14 default algorithm and Unicode data files; defer optional tailoring knobs until after conformance.
- Treat API generic input support (`~string | ~[]byte`) as a design requirement from day one, not a post-hoc refactor.
- Treat the public API as provisional until conformance harness integration is in place; optimize for testability and correctness over early API freeze.
- Set a performance bar up front: target `uax29`-class throughput and zero allocations in steady-state iteration paths.
- Key references: [`SPEC.md`](SPEC.md), UAX #14 algorithm, UAX #44 data model.

## Phase 1: Research and rule mapping

- Status: in progress (started)
- [x] Parse UAX #14 into an implementation worksheet: each LB rule (`LB1`, `LB2`, ...) mapped to concrete predicate logic, required lookbehind/lookahead, and break outcome (`must`, `may`, `no`).
- [x] Confirm class source-of-truth from `LineBreak.txt`; document handling of derived/default values and classes requiring algorithmic remapping.
- [x] Validate conformance assets exist and pin them in docs:
  - `https://unicode.org/Public/UNIDATA/LineBreak.txt`
  - `https://unicode.org/Public/UCD/latest/ucd/auxiliary/LineBreakTest.txt`
- Implementation artifacts:
  - `docs/phase1.md`
  - `docs/unicode-data.md`
  - `docs/rule-worksheet.md`

## Phase 2: Unicode data + trie codegen

- Status: in progress (started)
- [x] Implement generator to fetch and parse Unicode data (version-pinned URL + optional local file override).
- [x] Emit line break class enum as bit-flags (`1 << iota`) to support fast set checks.
- [x] Emit UTF-8 trie lookup table for byte-oriented lookup.
- [x] Keep generated artifacts stable (sorted output, header with Unicode version) to reduce diff noise.
- Implementation artifacts:
  - `docs/phase2.md`
  - `generate.go`
  - `trie.go` (generated)
  - `internal/gen/main.go`

## Phase 3: Lookup API

- Status: in progress (started)
- [x] Implement internal property lookup backed by generated trie (`lookupProperty[T ~string | ~[]byte](in T) property` wrapping generated `lookup`).
- [x] Keep lookup internal-only for now; do not expose a public `Lookup` API yet.
- [x] Add focused tests for representative code points across major LB classes, defaults, generic parity, and UTF-8 edge cases.
- Implementation artifacts:
  - `docs/phase3.md`
  - `lookup_test.go`

## Phase 4: SplitFunc core algorithm + conformance harness

- Status: in progress (started)
- Implement `bufio.SplitFunc` as the source of break decisions:
  - [x] Track current boundary position explicitly (matching spec semantics).
  - [x] Apply LB rules in spec order and keep rule labels in code comments/branch names so logic reads like UAX #14.
  - [x] Return both boundary offset and break kind (`must` vs `opportunity`) through internal state.
- [x] Include explicit handling for CR/LF/NL, combining marks/ZWJ behavior, regional indicators, emoji classes, and numeric punctuation interactions.
- [x] Keep this path allocation-free: no per-token heap objects, no string/byte copying, and no closure captures on hot loops.
- Conformance harness (now part of Phase 4):
  - [x] Parse `LineBreakTest.txt` and assert produced boundaries + mandatory-status invariants via internal split decisions.
  - [x] Land the harness now even if not yet passing; treat failures as rule-completion backlog.
  - [ ] Add an explicit API fitness checkpoint after parser + harness are working: "Does current iterator/API shape allow straightforward execution and assertion of conformance cases?" If not, revise API before stabilizing docs.
- Implementation artifacts:
  - `splitfunc.go`
  - `splitfunc_test.go`
  - `linebreak_conformance_test.go`
  - `linebreak_conformance_generated_test.go`
  - `docs/phase4.md`

## Phase 5: Iterator + public API

- Status: pending (blocked on initial conformance harness signal)
- Build iterator around split decisions:
  - `NewIterator[T ~string | ~[]byte](in T)`
  - `Next() bool`
  - `Current() T` (preserve caller’s input type)
  - `MustBreak() bool`
  - `CanBreak() bool`
- Internally normalize to `[]byte` for algorithm execution, but preserve typed slices/ranges so `Current()` does not force caller-side conversions.
- Ensure `Current()` is a view/slice of original input data, not a copied value.
- Keep API close to your sketch in `SPEC.md`; prefer correctness and clarity over early options.
- Add package docs and one short example demonstrating loop usage.

## Phase 6: Remaining test strategy and quality gates

- Status: pending
- Table tests for tricky LB rule interactions.
- Regression tests for UTF-8 edge cases (invalid sequences, long runs, mixed scripts).
- Add API parity tests that run the same corpus through `string` and `[]byte` generic instantiations and assert identical boundaries/break-kind semantics.
- Add lightweight benchmarks (ASCII, CJK, emoji-heavy) to track iterator throughput and allocations.
- Add explicit allocation assertions (e.g., `testing.AllocsPerRun`) for core iterator flows, with expected value `0` after warmup.
- CI target: `go test ./...` with generation check (`go generate` no-diff guard).

## Phase 7: Incremental delivery order

- Milestone 1: codegen + lookup + class tests.
- Milestone 2: initial SplitFunc covering structural rules and simple classes.
- Milestone 3: Phase 4 conformance harness integrated and running (currently failing until deferred rules are completed).
- Milestone 4: full LB rule coverage + conformance pass against `LineBreakTest.txt`.
- Milestone 5: iterator/API polish, docs, benchmarks, and release prep.

## Risks and mitigations

- Rule-order regressions: keep one-rule-per-block style and rule-tagged tests.
- Unicode version drift: pin version in generator and expose it in generated headers/tests.
- Ambiguous API semantics (`Current`, break-kind timing): lock with example tests documenting exact behavior.
- Generic API complexity/leakage: keep one internal core engine and thin typed wrappers to avoid duplicate logic for `string` vs `[]byte`.
- Hidden heap escapes from innocuous refactors: track with allocation benchmarks and escape-analysis checks during development.
