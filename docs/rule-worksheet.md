# UAX #14 Rule Worksheet (Phase 1)

This worksheet converts UAX #14 rules into implementation-facing entries for a boundary scanner.

Conventions used below:

- Boundary under evaluation is between `left` and `right`.
- Outcomes:
  - `must` = mandatory break (`!`)
  - `may` = break opportunity (`÷`)
  - `no` = prohibited break (`×`)
- "Lookbehind" and "lookahead" are relative to the current boundary.

## Pre-boundary setup

- `LB1` (class assignment/remap): apply before boundary scanning.
  - Outcome: n/a (classification step)
  - Required remaps: see `docs/unicode-data.md`.

## Boundary rules in application order

### Text start/end and hard breaks

- `LB2`: `sot ×`
  - Predicate: boundary is at start of text
  - Lookbehind: none
  - Lookahead: first code point
  - Outcome: `no`

- `LB3`: `! eot`
  - Predicate: boundary is at end of text
  - Lookbehind: last code point
  - Lookahead: none
  - Outcome: `must`

- `LB4`: `BK !`
  - Predicate: `left` class is `BK`
  - Lookbehind: 1 class
  - Lookahead: none
  - Outcome: `must`

- `LB5`: `CR × LF`, `CR !`, `LF !`, `NL !`
  - Predicate:
    - if `left=CR` and `right=LF` -> `no`
    - else if `left in {CR, LF, NL}` -> `must`
  - Lookbehind: 1 class
  - Lookahead: 1 class (for `CR × LF`)
  - Outcome: `no` / `must`

- `LB6`: `× (BK | CR | LF | NL)`
  - Predicate: `right in {BK, CR, LF, NL}`
  - Lookbehind: none
  - Lookahead: 1 class
  - Outcome: `no`

### Explicit non-break/break controls

- `LB7`: `× SP`, `× ZW`
  - Predicate: `right in {SP, ZW}`
  - Lookahead: 1 class
  - Outcome: `no`

- `LB8`: `ZW SP* ÷`
  - Predicate: nearest non-`SP` class on left side is `ZW`
  - Lookbehind: variable (skip spaces)
  - Lookahead: current `right`
  - Outcome: `may`

- `LB8a`: `ZWJ ×`
  - Predicate: `left=ZWJ`
  - Lookbehind: 1 class
  - Outcome: `no`

### Combining marks

- `LB9`: `X (CM | ZWJ)*` treated as `X` (except certain `X`)
  - Predicate: preprocessing/scan-time ignore rule
  - Lookbehind/lookahead: variable
  - Outcome: n/a (normalization of rule inputs)

- `LB10`: remaining `CM`/`ZWJ` -> `AL`
  - Predicate: preprocessing fallback
  - Outcome: n/a

### Joiners and glue

- `LB11`: `× WJ`, `WJ ×`
  - Predicate: either side is `WJ`
  - Lookbehind/lookahead: 1 class both sides
  - Outcome: `no`

- `LB12`: `GL ×`
  - Predicate: `left=GL`
  - Lookbehind: 1 class
  - Outcome: `no`

- `LB12a`: `[^SP BA HY HH] × GL`
  - Predicate: `right=GL` and `left not in {SP, BA, HY, HH}`
  - Lookbehind/lookahead: 1 class
  - Outcome: `no`

### Opening/closing punctuation cluster

- `LB13`: `× CL`, `× CP`, `× EX`, `× SY`
  - Predicate: `right in {CL, CP, EX, SY}`
  - Outcome: `no`

- `LB14`: `OP SP* ×`
  - Predicate: nearest non-`SP` class on left is `OP`
  - Lookbehind: variable (skip spaces)
  - Outcome: `no`

- `LB15a`: `(sot|BK|CR|LF|NL|OP|QU|GL|SP|ZW) [Pi&QU] SP* ×`
  - Predicate:
    - unresolved initial quotation before boundary
    - context before quotation is in listed start/open classes
    - optional spaces between quotation and boundary
  - Lookbehind: multi-step (skip spaces; inspect quoted char and prior context)
  - Outcome: `no`

- `LB15b`: `× [Pf&QU] (SP|GL|WJ|CL|QU|CP|EX|IS|SY|BK|CR|LF|NL|ZW|eot)`
  - Predicate: unresolved final quotation on right with listed right context
  - Lookahead: multi-step (inspect right quotation and following class/eot)
  - Outcome: `no`

- `LB15c`: `SP ÷ IS NU`
  - Predicate: `left=SP` and `right=IS` and `next(right)=NU`
  - Lookahead: 2 classes
  - Outcome: `may`

- `LB15d`: `× IS`
  - Predicate: `right=IS`
  - Outcome: `no`

- `LB16`: `(CL | CP) SP* × NS`
  - Predicate: left non-space is `CL`/`CP`, right is `NS`
  - Lookbehind: variable (skip spaces)
  - Lookahead: 1 class
  - Outcome: `no`

- `LB17`: `B2 SP* × B2`
  - Predicate: left non-space is `B2` and `right=B2`
  - Lookbehind: variable (skip spaces)
  - Lookahead: 1 class
  - Outcome: `no`

- `LB18`: `SP ÷`
  - Predicate: `left=SP`
  - Outcome: `may`

### Special cases

- `LB19`: `× [QU-Pi]`, `[QU-Pf] ×`
  - Predicate:
    - no break before non-initial unresolved quote
    - no break after non-final unresolved quote
  - Lookbehind/lookahead: 1 class with quote subtype checks
  - Outcome: `no`

- `LB19a`: East Asian context-sensitive unresolved quote handling
  - Predicate: around `QU`, suppress breaks unless both sides are East Asian context
  - Lookbehind/lookahead: up to 1 class each side plus `sot/eot`
  - Outcome: `no`

- `LB20`: `÷ CB`, `CB ÷`
  - Predicate: either side is unresolved `CB`
  - Lookbehind/lookahead: 1 class
  - Outcome: `may`

- `LB20a`: `(sot|BK|CR|LF|NL|SP|ZW|CB|GL) (HY|HH) × (AL|HL)`
  - Predicate: word-initial hyphen context before `AL/HL`
  - Lookbehind: 2 logical items (pre-context + `HY/HH`)
  - Lookahead: 1 class (`AL/HL`)
  - Outcome: `no`

- `LB21`: `× BA`, `× HH`, `× HY`, `× NS`, `BB ×`
  - Predicate: right in listed non-starters/hyphens or left is `BB`
  - Outcome: `no`

- `LB21a`: `HL (HY|HH) × [^HL]`
  - Predicate: Hebrew letter + hyphen, before non-Hebrew
  - Lookbehind: 2 classes
  - Lookahead: 1 class with negation
  - Outcome: `no`

- `LB21b`: `SY × HL`
  - Predicate: solidus before Hebrew letter
  - Lookbehind/lookahead: 1 class each
  - Outcome: `no`

- `LB22`: `× IN`
  - Predicate: `right=IN`
  - Outcome: `no`

### Numeric and alphabetic sequences

- `LB23`: `(AL|HL) × NU`, `NU × (AL|HL)`
  - Predicate: digit-letter adjacency in either direction
  - Outcome: `no`

- `LB23a`: `PR × (ID|EB|EM)`, `(ID|EB|EM) × PO`
  - Predicate: numeric prefix/postfix with ideographs or emoji ideograph-like classes
  - Outcome: `no`

- `LB24`: `(PR|PO) × (AL|HL)`, `(AL|HL) × (PR|PO)`
  - Predicate: prefix/postfix adjacent to alphabetics
  - Outcome: `no`

- `LB25`: numeric cluster protection (multiple patterns)
  - Predicate: prevent breaks inside number expressions, including:
    - `NU (SY|IS)*` before `PO`/`PR` (with or without `CL`/`CP`)
    - `PO`/`PR` before `OP NU`, `OP IS NU`, or `NU`
    - `HY × NU`, `IS × NU`
    - `NU (SY|IS)* × NU`
  - Lookbehind/lookahead: variable spans with repeat groups
  - Outcome: `no`

### Korean and script-specific shaping

- `LB26`: Korean syllable block no-break patterns
  - Predicate:
    - `JL × (JL|JV|H2|H3)`
    - `(JV|H2) × (JV|JT)`
    - `(JT|H3) × JT`
  - Outcome: `no`

- `LB27`: `(JL|JV|JT|H2|H3) × PO`, `PR × (JL|JV|JT|H2|H3)`
  - Predicate: treat Korean syllable block behavior as `ID` in prefix/postfix contexts
  - Outcome: `no`

- `LB28`: `(AL|HL) × (AL|HL)`
  - Predicate: alphabetic word interior
  - Outcome: `no`

- `LB28a`: Brahmic orthographic syllable no-break patterns
  - Predicate:
    - `AP × (AK|U+25CC|AS)`
    - `(AK|U+25CC|AS) × (VF|VI)`
    - `(AK|U+25CC|AS) VI × (AK|U+25CC)`
    - `(AK|U+25CC|AS) × (AK|U+25CC|AS) VF`
  - Lookbehind/lookahead: up to 2 classes
  - Outcome: `no`

- `LB29`: `IS × (AL|HL)`
  - Predicate: numeric punctuation before alphabetic
  - Outcome: `no`

- `LB30`: `(AL|HL|NU) × [OP-$EastAsian]`, `[CP-$EastAsian] × (AL|HL|NU)`
  - Predicate: parenthetical delimiters around alnum/symbol words, excluding East Asian opener/closer subset
  - Outcome: `no`

- `LB30a`: RI pairing parity rule
  - Predicate: break between `RI` and `RI` only when preceding RI run length is even
  - Lookbehind: variable-length RI run with parity tracking
  - Outcome: `may` (if even), otherwise `no`

- `LB30b`: `EB × EM`, `[Extended_Pictographic & Cn] × EM`
  - Predicate: emoji base/potential-emoji before modifier
  - Lookbehind/lookahead: 1 class plus property checks
  - Outcome: `no`

- `LB31`: default fallback (`ALL ÷`, `÷ ALL`)
  - Predicate: no prior rule matched
  - Outcome: `may`

## Implementation-state implications for next phases

- Rule application must be strictly ordered.
- Scanner needs support for:
  - skipping spaces for several rules (`SP*`)
  - peeking ahead up to 2+ classes
  - variable-length lookbehind (`LB8`, `LB14`, `LB16`, `LB17`, `LB30a`)
  - side-channel property predicates (quote subtype, East Asian set, Extended_Pictographic)
- Internal decision type should preserve:
  - `must` vs `may` for API exposure (`MustBreak`, `CanBreak`)
  - the rule label for debugging/test diagnostics (e.g., `LB25`)
