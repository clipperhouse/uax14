# Phase 3: Lookup API

## TODOs

- [x] Validate that generated trie lookup is available as an internal API.
- [x] Keep lookup internal-only (no exported `Lookup` yet).
- [x] Add lookup-focused tests for representative classes and defaults.
- [x] Add generic parity tests (`string` vs `[]byte`).
- [x] Add UTF-8 edge-case tests for invalid and incomplete input.

## Notes

- The generated trie provides a raw internal generic lookup:
  - `lookup[T ~string | ~[]byte](in T) (property, int)`
- Phase 3 adds an internal normalized facade used by upcoming algorithm work:
  - `lookupProperty[T ~string | ~[]byte](in T) property`
  - valid unmapped scalars default to `_XX`.
  - invalid/truncated UTF-8 remains `0` (via raw `lookup` semantics).
- This keeps the package surface minimal while Phase 4 (`SplitFunc`) is in progress.
