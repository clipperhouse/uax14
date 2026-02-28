# Phase 1: Research and Rule Mapping

This document tracks concrete Phase 1 tasks from `PLAN.md` and links to the implementation-ready research artifacts.

## TODOs

- [x] Create UAX #14 implementation worksheet mapping each LB rule to:
  - boundary predicate form
  - lookbehind requirements
  - lookahead requirements
  - break outcome (`must`, `may`, `no`)
- [x] Confirm line break class source-of-truth from Unicode data files.
- [x] Document LB1 resolution/remapping for `AI`, `CB`, `CJ`, `SA`, `SG`, and `XX`.
- [x] Pin conformance asset URLs in repo docs:
  - `https://unicode.org/Public/UNIDATA/LineBreak.txt`
  - `https://unicode.org/Public/UCD/latest/ucd/auxiliary/LineBreakTest.txt`
- [x] Capture implementation notes for rule ordering and state shape needed by split/iterator phases.

## Deliverables

- Rule worksheet: `docs/rule-worksheet.md`
- Unicode data and conformance inputs: `docs/unicode-data.md`
