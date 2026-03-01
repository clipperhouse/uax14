package uax14

import (
	"fmt"
	"testing"
)

func TestLineBreakConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping conformance test in short mode")
	}

	cases := len(lineBreakConformanceTests)
	if cases == 0 {
		t.Fatal("no generated conformance test cases")
	}

	mismatches := 0
	const maxExamples = 5
	examples := make([]string, 0, maxExamples)

	for _, tc := range lineBreakConformanceTests {
		gotOffsets, gotKinds, err := runSplitDecisions(tc.input)
		if err != nil {
			mismatches++
			if len(examples) < maxExamples {
				examples = append(examples, fmt.Sprintf("line %d: split decisions failed: %v", tc.lineNo, err))
			}
			continue
		}

		if len(gotOffsets) != len(tc.breakOffsets) {
			mismatches++
			if len(examples) < maxExamples {
				examples = append(examples, fmt.Sprintf("line %d: break count mismatch got=%d want=%d input=%q got=%v want=%v", tc.lineNo, len(gotOffsets), len(tc.breakOffsets), string(tc.input), gotOffsets, tc.breakOffsets))
			}
			continue
		}

		caseMismatch := false
		for i := range gotOffsets {
			if gotOffsets[i] != tc.breakOffsets[i] {
				caseMismatch = true
				mismatches++
				if len(examples) < maxExamples {
					examples = append(examples, fmt.Sprintf("line %d: break mismatch idx=%d got=%d want=%d input=%q got=%v want=%v", tc.lineNo, i, gotOffsets[i], tc.breakOffsets[i], string(tc.input), gotOffsets, tc.breakOffsets))
				}
				break
			}
		}
		if caseMismatch {
			continue
		}

		// Break-kind sanity check: every boundary must have a kind and final break is mandatory (LB3).
		if len(gotKinds) == 0 {
			mismatches++
			if len(examples) < maxExamples {
				examples = append(examples, fmt.Sprintf("line %d: no break kinds produced", tc.lineNo))
			}
			continue
		}
		kindsInvalid := false
		for i, k := range gotKinds {
			if k != breakMandatory && k != breakOpportunity {
				kindsInvalid = true
				mismatches++
				if len(examples) < maxExamples {
					examples = append(examples, fmt.Sprintf("line %d: invalid break kind at idx=%d: %v", tc.lineNo, i, k))
				}
				break
			}
		}
		if kindsInvalid {
			continue
		}
		if gotKinds[len(gotKinds)-1] != breakMandatory {
			mismatches++
			if len(examples) < maxExamples {
				examples = append(examples, fmt.Sprintf("line %d: final break kind is %v, want mandatory", tc.lineNo, gotKinds[len(gotKinds)-1]))
			}
			continue
		}
	}
	if mismatches == 0 {
		t.Logf("line break conformance: %d/%d cases matched", cases, cases)
		return
	}

	t.Logf("line break conformance mismatches: %d/%d", mismatches, cases)
	for _, ex := range examples {
		t.Log(ex)
	}
	t.Fatalf("line break conformance mismatches remain: %d", mismatches)
}

func runSplitDecisions(input []byte) ([]int, []breakKind, error) {
	remaining := input
	offset := 0
	offsets := make([]int, 0, len(input))
	kinds := make([]breakKind, 0, len(input))

	for len(remaining) > 0 {
		advance, kind := NextBreak(remaining)
		if advance <= 0 || advance > len(remaining) {
			return nil, nil, fmt.Errorf("invalid advance: %d", advance)
		}
		offset += advance
		offsets = append(offsets, offset)
		kinds = append(kinds, kind)
		remaining = remaining[advance:]
	}

	return offsets, kinds, nil
}
