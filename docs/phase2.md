# Phase 2: Unicode Data + Trie Codegen

## TODOs

- [x] Add `go generate` entrypoint for Unicode data generation.
- [x] Implement parser for `LineBreak.txt` records (single code point + ranges).
- [x] Support version-pinned source URL with optional local override input.
- [x] Generate stable bit-flag `Class` enum output.
- [x] Generate UTF-8 trie lookup table + lookup function from the parsed LineBreak data.
- [x] Keep generated artifacts deterministic (stable class ordering + Unicode version header).
- [x] Cache Unicode source files locally under a versioned path to avoid repeated network fetches.
- [ ] Add tests for parser/generator determinism.

## Notes

- This phase starts with the smallest executable slice and avoids premature package scaffolding.
- Current generated artifact is trie-only (no range-table fallback path).
- Generator entrypoint lives at repo root (`generate.go`) and runs with `-C internal/gen`.
- Generated trie output is `trie.go` at repo root for direct use by public API.
- Unicode version is pinned in `internal/gen/main.go` via `unicodeVersion` constant.
- Generator cache paths include the Unicode version:
  - `internal/gen/cache/<unicodeVersion>/LineBreak.txt`
  - `internal/gen/cache/<unicodeVersion>/LineBreakTest.txt`
- Use `go generate ./...` for cached/offline regeneration after initial download.
- Use `go generate ./...` with `-refresh` (via `go run -C internal/gen . -refresh`) to refresh cache from network.
