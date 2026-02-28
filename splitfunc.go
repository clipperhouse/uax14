package uax14

import "bufio"

// SplitFunc segments line-break tokens for bufio.Scanner.
var SplitFunc bufio.SplitFunc = splitFunc[[]byte]

type breakKind uint8

const (
	breakOpportunity breakKind = iota + 1
	breakMandatory
)

func (p property) is(classes property) bool {
	return (p & classes) != 0
}

func splitFunc[T ~string | ~[]byte](data T, atEOF bool) (advance int, token T, err error) {
	var empty T
	advance, _, _, err = splitDecision(data, atEOF)
	if advance == 0 || err != nil {
		return advance, empty, err
	}
	return advance, data[:advance], nil
}

// splitDecision applies UAX #14 rules for the first break boundary in data.
// It returns the token advance, break kind, and a rule label for diagnostics.
func splitDecision[T ~string | ~[]byte](data T, atEOF bool) (advance int, kind breakKind, rule string, err error) {
	var emptyRule string
	if len(data) == 0 {
		return 0, 0, emptyRule, nil
	}

	leftRaw, w := lookup(data)
	if w == 0 {
		if !atEOF {
			return 0, 0, emptyRule, nil
		}
		return len(data), breakMandatory, "LB3", nil
	}

	pos := w // LB2: sot ×
	left := lbClass(leftRaw)
	prev := property(0)
	lastNonSP := left
	if left == _SP {
		lastNonSP = 0
	}
	riRun := 0
	if left == _RI {
		riRun = 1
	}

	for {
		if pos == len(data) {
			if !atEOF {
				return 0, 0, emptyRule, nil
			}
			return pos, breakMandatory, "LB3", nil
		}

		rightRaw, rw := lookup(data[pos:])
		if rw == 0 {
			if !atEOF {
				return 0, 0, emptyRule, nil
			}
			return len(data), breakMandatory, "LB3", nil
		}
		right := lbClass(rightRaw)

		// LB4: BK !
		if left == _BK {
			return pos, breakMandatory, "LB4", nil
		}

		// LB5: CR × LF, CR/LF/NL !
		if left == _CR && right == _LF {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}
		if left.is(_CR | _LF | _NL) {
			return pos, breakMandatory, "LB5", nil
		}

		// LB6: × (BK | CR | LF | NL)
		if right.is(_BK | _CR | _LF | _NL) {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB7: × SP, × ZW
		if right.is(_SP | _ZW) {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB8: ZW SP* ÷
		if lastNonSP == _ZW {
			return pos, breakOpportunity, "LB8", nil
		}

		// LB8a: ZWJ ×
		if left == _ZWJ {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB9/LB10: combining mark and ZWJ behavior.
		if right.is(_CM | _ZWJ) {
			// LB9: X × (CM|ZWJ), where X is not BK/CR/LF/NL/SP/ZW.
			if !left.is(_BK | _CR | _LF | _NL | _SP | _ZW) {
				prev, left, pos = step(prev, left, right, pos, rw)
				lastNonSP = updateLastNonSP(lastNonSP, left)
				riRun = updateRIRun(riRun, left)
				continue
			}
			// LB10: remaining CM/ZWJ resolve to AL.
			right = _AL
		}

		// LB11: × WJ, WJ ×
		if left == _WJ || right == _WJ {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB12: GL ×
		if left == _GL {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB12a: [^SP BA HY HH] × GL
		if right == _GL && !left.is(_SP|_BA|_HY|_HH) {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB13: × CL, × CP, × EX, × SY
		if right.is(_CL | _CP | _EX | _SY) {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB14: OP SP* ×
		if lastNonSP == _OP {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB16: (CL | CP) SP* × NS
		if right == _NS && lastNonSP.is(_CL|_CP) {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB17: B2 SP* × B2
		if right == _B2 && lastNonSP == _B2 {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB18: SP ÷
		if left == _SP {
			return pos, breakOpportunity, "LB18", nil
		}

		// LB19 approximation: suppress break around unresolved quotes.
		if left == _QU || right == _QU {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB20: ÷ CB, CB ÷
		if left == _CB || right == _CB {
			return pos, breakOpportunity, "LB20", nil
		}

		// LB21: × BA/HH/HY/NS, BB ×
		if right.is(_BA | _HH | _HY | _NS) || left == _BB {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB21a: HL (HY|HH) × [^HL]
		if prev == _HL && left.is(_HY|_HH) && right != _HL {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB21b: SY × HL
		if left == _SY && right == _HL {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB22: × IN
		if right == _IN {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB23: (AL|HL) × NU, NU × (AL|HL)
		if (left.is(_AL|_HL) && right == _NU) || (left == _NU && right.is(_AL|_HL)) {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB23a: PR × (ID|EB|EM), (ID|EB|EM) × PO
		if (left == _PR && right.is(_ID|_EB|_EM)) || (left.is(_ID|_EB|_EM) && right == _PO) {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB24: (PR|PO) × (AL|HL), (AL|HL) × (PR|PO)
		if (left.is(_PR|_PO) && right.is(_AL|_HL)) || (left.is(_AL|_HL) && right.is(_PR|_PO)) {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB25 subset: numeric punctuation and sign interactions.
		if isLB25NoBreak(prev, left, right) {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB26: Korean syllable no-breaks.
		if (left == _JL && right.is(_JL|_JV|_H2|_H3)) ||
			(left.is(_JV|_H2) && right.is(_JV|_JT)) ||
			(left.is(_JT|_H3) && right == _JT) {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB27: Korean blocks with prefix/postfix.
		if (left.is(_JL|_JV|_JT|_H2|_H3) && right == _PO) || (left == _PR && right.is(_JL|_JV|_JT|_H2|_H3)) {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB28: (AL|HL) × (AL|HL)
		if left.is(_AL|_HL) && right.is(_AL|_HL) {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB29: IS × (AL|HL)
		if left == _IS && right.is(_AL|_HL) {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB30 approximation: (AL|HL|NU) × OP, CP × (AL|HL|NU).
		// East Asian exclusions are deferred to the conformance phase.
		if (left.is(_AL|_HL|_NU) && right == _OP) || (left == _CP && right.is(_AL|_HL|_NU)) {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB30a: RI parity pair rule.
		if left == _RI && right == _RI {
			if riRun%2 == 1 {
				prev, left, pos = step(prev, left, right, pos, rw)
				lastNonSP = updateLastNonSP(lastNonSP, left)
				riRun = updateRIRun(riRun, left)
				continue
			}
			return pos, breakOpportunity, "LB30a", nil
		}

		// LB30b: EB × EM (Extended_Pictographic Cn extension deferred).
		if left == _EB && right == _EM {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB31: default break opportunity.
		return pos, breakOpportunity, "LB31", nil
	}
}

func lbClass(p property) property {
	switch p {
	case _AI, _SG, _XX:
		return _AL
	case _CJ:
		return _NS
	case _SA:
		// Full SA remap needs General_Category data; AL fallback for now.
		return _AL
	default:
		return p
	}
}

func isLB25NoBreak(prev, left, right property) bool {
	// HY × NU, IS × NU, NU × NU
	if right == _NU && left.is(_HY|_IS|_NU) {
		return true
	}

	// NU × (SY|IS|CL|CP)
	if left == _NU && right.is(_SY|_IS|_CL|_CP) {
		return true
	}

	// (PR|PO) × NU, NU × (PR|PO)
	if (left.is(_PR|_PO) && right == _NU) || (left == _NU && right.is(_PR|_PO)) {
		return true
	}

	// NU (SY|IS|CL|CP)* × NU (common decimal/grouping patterns).
	if right == _NU && left.is(_SY|_IS|_CL|_CP) && prev == _NU {
		return true
	}

	return false
}

func step(prev, left, right property, pos, width int) (property, property, int) {
	return left, right, pos + width
}

func updateLastNonSP(lastNonSP, p property) property {
	if p != _SP {
		return p
	}
	return lastNonSP
}

func updateRIRun(run int, p property) int {
	if p == _RI {
		return run + 1
	}
	return 0
}
