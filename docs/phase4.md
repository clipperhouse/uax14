# Phase 4: SplitFunc core algorithm + conformance harness

## TODOs

- [x] Add `SplitFunc` implementation in package root for `bufio.Scanner` integration.
- [x] Add internal split-decision path that returns:
  - boundary advance
  - break kind (`mandatory` vs `opportunity`)
  - matched rule label (e.g. `LB5`, `LB30a`) for diagnostics/tests
- [x] Implement explicit handling for:
  - hard-break controls (`BK`, `CR`, `LF`, `NL`)
  - combining marks and `ZWJ` behavior (`LB8a`, `LB9`, `LB10`)
  - regional indicator parity (`LB30a`)
  - emoji modifier attachment (`LB30b`, `EB × EM`)
  - numeric punctuation interactions (core `LB25` patterns)
- [x] Add focused tests for the behaviors above.
- [x] Add conformance harness test:
  - consumes generated Go fixtures for `LineBreakTest.txt` cases (no runtime parsing/downloading)
  - validates `splitDecision` break offsets against Unicode expectations
  - checks break-kind invariants (final boundary is mandatory)

## Notes

- `splitfunc.go` follows UAX #14 rule ordering and keeps per-rule labels in return values.
- `SplitFunc` is a thin wrapper over internal `splitDecision`, preserving a `bufio.SplitFunc` surface while making break-kind state available to upcoming iterator work.
- `internal/gen/main.go` now generates `linebreak_conformance_generated_test.go` from `LineBreakTest.txt`:
  - each case includes UTF-8 input bytes, expected break offsets, source line number, and comment
  - test execution is fully offline once generation has been run
- `linebreak_conformance_test.go` logs mismatch counts and representative examples, and fails whenever mismatches are present.
- The conformance test skips in `-short` mode.
- LB1 remapping is applied at scan time:
  - `AI`, `SG`, `XX` -> `AL`
  - `CJ` -> `NS`
  - `SA` currently falls back to `AL` until generator/runtime wiring includes `General_Category` (`Mn`/`Mc`) for full conformance behavior.
- Remaining conformance-sensitive items are now tracked as rule-completion backlog:
  - quote subtype disambiguation (`LB15*`, `LB19a`)
  - East Asian exclusions in `LB30`
  - full `LB25` pattern family
  - `Extended_Pictographic & Cn` condition in `LB30b`
