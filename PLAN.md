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

## Phase 2: Repo/file skeleton

- Add initial structure:
  - `internal/gen/main.go`
  - `internal/data/linebreak_classes.go` (generated)
  - `lookup.go`
  - `split.go`
  - `iterator.go`
  - `doc.go`
  - `internal/tests/linebreak_conformance_test.go`
- Add `go:generate` wiring so data generation is deterministic and reproducible.

## Phase 3: Unicode data + trie codegen

- Implement generator to fetch and parse Unicode data (version-pinned URL + optional local cache).
- Emit:
  - Line break class enum as bit-flags (`1 << iota`) to support fast set checks.
  - Trie lookup table via `x/text` triegen for UTF-8 byte lookup.
- Keep generated artifacts stable (sorted output, header with Unicode version) to reduce diff noise.

## Phase 4: Lookup API

- Implement internal lookup: `func lookup(p []byte) Class` (or rune-based equivalent) backed by generated trie.
- Add generic facade where useful, e.g. `Lookup[T ~string | ~[]byte](in T)` delegating to shared byte-oriented core.
- Expose public API only if needed (`Lookup`), otherwise keep minimal surface and let iterator/split drive usage.
- Add focused tests for representative code points across major LB classes and defaults.

## Phase 5: SplitFunc core algorithm

- Implement `bufio.SplitFunc` as the source of break decisions:
  - Track current boundary position explicitly (matching spec semantics).
  - Apply LB rules in spec order and keep rule labels in code comments/branch names so logic reads like UAX #14.
  - Return both boundary offset and break kind (`must` vs `opportunity`) through internal state.
- Include explicit handling for CR/LF/NL, combining marks/ZWJ behavior, regional indicators, emoji classes, and numeric punctuation interactions.
- Keep this path allocation-free: no per-token heap objects, no string/byte copying, and no closure captures on hot loops.

## Phase 6: Iterator + public API

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

## Phase 7: Test strategy and quality gates

- Conformance tests:
  - Parse `LineBreakTest.txt` and assert produced boundaries + mandatory status.
  - Add an explicit API fitness checkpoint after parser + harness are working: "Does current iterator/API shape allow straightforward execution and assertion of conformance cases?" If not, revise API before stabilizing docs.
  - Table tests for tricky LB rule interactions.
- Regression tests for UTF-8 edge cases (invalid sequences, long runs, mixed scripts).
- Add API parity tests that run the same corpus through `string` and `[]byte` generic instantiations and assert identical boundaries/break-kind semantics.
- Add lightweight benchmarks (ASCII, CJK, emoji-heavy) to track iterator throughput and allocations.
- Add explicit allocation assertions (e.g., `testing.AllocsPerRun`) for core iterator flows, with expected value `0` after warmup.
- CI target: `go test ./...` with generation check (`go generate` no-diff guard).

## Phase 8: Incremental delivery order

- Milestone 1: codegen + lookup + class tests.
- Milestone 2: initial SplitFunc covering structural rules and simple classes.
- Milestone 3: full LB rule coverage + conformance pass against `LineBreakTest.txt`.
- Milestone 4: iterator/API polish, docs, benchmarks, and release prep.

## Risks and mitigations

- Rule-order regressions: keep one-rule-per-block style and rule-tagged tests.
- Unicode version drift: pin version in generator and expose it in generated headers/tests.
- Ambiguous API semantics (`Current`, break-kind timing): lock with example tests documenting exact behavior.
- Generic API complexity/leakage: keep one internal core engine and thin typed wrappers to avoid duplicate logic for `string` vs `[]byte`.
- Hidden heap escapes from innocuous refactors: track with allocation benchmarks and escape-analysis checks during development.
