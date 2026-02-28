# Unicode Data Inputs (Phase 1)

## Source of truth

- Line break classes are sourced from `LineBreak.txt`:
  - `https://unicode.org/Public/UNIDATA/LineBreak.txt`
- Conformance cases are sourced from `LineBreakTest.txt`:
  - `https://unicode.org/Public/UCD/latest/ucd/auxiliary/LineBreakTest.txt`

## LB1 class resolution requirements

Per UAX #14 LB1, classes below must be remapped before applying boundary rules:

- `AI` -> `AL`
- `SG` -> `AL`
- `XX` -> `AL`
- `SA` -> `CM` when `General_Category` is `Mn` or `Mc`
- `SA` -> `AL` otherwise
- `CJ` -> `NS` (default behavior)
- `CB` remains conditional and is handled by LB20 (`break before and after unresolved CB`)

## Additional properties used by rules

The default algorithm also depends on:

- `General_Category` (for LB1 and quotation-related conditions)
- `East_Asian_Width` (for `$EastAsian` in LB19a/LB30)
- `Extended_Pictographic` and `Cn` interaction (LB30b)

## Generator parsing notes for later phases

- Parse `LineBreak.txt` as ordered ranges and build a dense class enum for trie generation.
- Keep raw assigned class in generated data, then apply LB1 remapping in runtime lookup (or a generated second-pass table) so algorithm code sees resolved classes.
- Include Unicode version metadata in generated headers to detect drift.
- Keep parser deterministic:
  - stable range ordering
  - stable enum ordering
  - stable code generation output
