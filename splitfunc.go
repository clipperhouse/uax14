package uax14

import (
	"bufio"
	"unicode"
	"unicode/utf8"
)

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

func NextBreak[T ~string | ~[]byte](data T) (advance int, kind breakKind) {
	if len(data) == 0 {
		return 0, breakMandatory
	}

	// These vars are stateful across loop iterations
	var pos int
	var lastExSP property = 0    // "last excluding SP"
	var beforeLastExSP property  // predecessor of lastExSP, with CM/ZWJ ignored
	var lastExCMZWJ property = 0 // "last excluding CM and ZWJ"

	current, w := lookup(data[pos:])
	if w == 0 {
		pos = len(data)
		return pos, breakMandatory
	}

	// https://www.unicode.org/reports/tr14/#LB2
	// Start of text always advances
	pos += w

	for {
		eot := pos == len(data) // "end of text"

		if eot {
			// https://www.unicode.org/reports/tr14/#LB3
			return pos, breakMandatory
		}

		// Remember previous properties to avoid lookups/lookbacks
		last := current
		prevExCMZWJ := lastExCMZWJ
		if !last.is(_SP) {
			beforeLastExSP = prevExCMZWJ
			lastExSP = last
		}
		if !last.is(_CM | _ZWJ) {
			lastExCMZWJ = last
		}

		current, w = lookup(data[pos:])
		if w == 0 {
			pos = len(data)
			return pos, breakMandatory
		}

		// https://www.unicode.org/reports/tr14/#LB4
		// Break after BK
		if last.is(_BK) {
			return pos, breakMandatory
		}

		// https://www.unicode.org/reports/tr14/#LB5
		// CR × LF; break after CR, LF, NL
		if last.is(_CR) && current.is(_LF) {
			pos += w
			continue
		}
		if last.is(_CR | _LF | _NL) {
			return pos, breakMandatory
		}

		// https://www.unicode.org/reports/tr14/#LB6
		// No break before BK, CR, LF, NL
		if current.is(_BK | _CR | _LF | _NL) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB7
		// No break before SP or ZW
		if current.is(_SP | _ZW) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB8
		// Break after ZW SP*
		if lastExSP.is(_ZW) {
			return pos, breakOpportunity
		}

		// https://www.unicode.org/reports/tr14/#LB8a
		// No break after ZWJ
		if last.is(_ZWJ) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB9
		// Absorb CM and ZWJ into the preceding base character (if eligible)
		// https://www.unicode.org/reports/tr14/#LB10
		// Remaining CM and ZWJ (after BK/CR/LF/NL/SP/ZW or sot) resolve to AL
		if current.is(_CM | _ZWJ) {
			if lastExCMZWJ != 0 && !lastExCMZWJ.is(_BK|_CR|_LF|_NL|_SP|_ZW) {
				pos += w
				continue
			}
			current = _AL
		}

		// https://www.unicode.org/reports/tr14/#LB11
		// No break before WJ
		if (current | lastExCMZWJ).is(_WJ) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB12
		// GL ×: no break after GL
		if lastExCMZWJ.is(_GL) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB12a
		// [^SP BA HY HH] × GL: no break before GL, unless preceded by SP, BA, HY, HH
		if current.is(_GL) && !lastExCMZWJ.is(_SP|_BA|_HY|_HH) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB13
		// × CL, × CP, × EX, × SY: no break before these
		if current.is(_CL | _CP | _EX | _SY) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB14
		// OP SP* ×: no break after OP (with optional SP*)
		if lastExSP.is(_OP) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB15a
		// (sot | BK | CR | LF | NL | OP | QU | GL | SP | ZW) [\p{Pi}&QU] SP* ×
		if lastExSP.is(_PI) &&
			(beforeLastExSP == 0 || beforeLastExSP.is(_BK|_CR|_LF|_NL|_OP|_QU|_GL|_SP|_ZW)) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB15b
		// × [\p{Pf}&QU] (SP | GL | WJ | CL | QU | CP | EX | IS | SY | BK | CR | LF | NL | ZW | eot)
		if current.is(_PF) && current.is(_QU) {
			next := property(0) // 0 means eot here
			if pos+w < len(data) {
				nr, nw := lookup(data[pos+w:])
				if nw > 0 {
					next = nr
				}
			}
			if next == 0 || next.is(_SP|_GL|_WJ|_CL|_QU|_CP|_EX|_IS|_SY|_BK|_CR|_LF|_NL|_ZW) {
				pos += w
				continue
			}
		}

		// https://www.unicode.org/reports/tr14/#LB15c
		// SP ÷ IS NU
		if last.is(_SP) && current.is(_IS) {
			next := property(0)
			if pos+w < len(data) {
				nr, nw := lookup(data[pos+w:])
				if nw > 0 {
					next = nr
				}
			}
			if next.is(_NU) {
				return pos, breakOpportunity
			}
		}

		// https://www.unicode.org/reports/tr14/#LB15d
		// × IS: no break before IS
		if current.is(_IS) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB16
		// (CL | CP) SP* × NS
		if lastExSP.is(_CL|_CP) && current.is(_NS) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB17
		// B2 SP* × B2
		if lastExSP.is(_B2) && current.is(_B2) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB18
		// SP ÷
		if last.is(_SP) {
			return pos, breakOpportunity
		}

		// https://www.unicode.org/reports/tr14/#LB19
		// × [ QU - \p{Pi} ] and [ QU - \p{Pf} ] ×
		if (current.is(_QU) && !current.is(_PI)) || (lastExCMZWJ.is(_QU) && !lastExCMZWJ.is(_PF)) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB19a
		// [^$EastAsian] × QU
		// × QU ( [^$EastAsian] | eot )
		// QU × [^$EastAsian]
		// ( sot | [^$EastAsian] ) QU ×
		if current.is(_QU) || last.is(_QU) {
			next := property(0) // 0 means eot here
			if pos+w < len(data) {
				nr, nw := lookup(data[pos+w:])
				if nw > 0 {
					next = nr
				}
			}

			noBreakBeforeQU := current.is(_QU) && (!last.is(_EA) || !next.is(_EA))
			noBreakAfterQU := last.is(_QU) && (!current.is(_EA) || prevExCMZWJ == 0 || !prevExCMZWJ.is(_EA))
			if noBreakBeforeQU || noBreakAfterQU {
				pos += w
				continue
			}
		}

		// https://www.unicode.org/reports/tr14/#LB31
		// Default break opportunity
		return pos, breakOpportunity
	}
}

func splitFunc[T ~string | ~[]byte](data T, atEOF bool) (advance int, token T, err error) {
	var empty T
	advance, _, _, err = splitDecisionOld(data, atEOF)
	if advance == 0 || err != nil {
		return advance, empty, err
	}
	return advance, data[:advance], nil
}

// splitDecisionOld applies UAX #14 rules for the first break boundary in data.
// It returns the token advance, break kind, and a rule label for diagnostics.
func splitDecisionOld[T ~string | ~[]byte](data T, atEOF bool) (advance int, kind breakKind, rule string, err error) {
	var emptyRule string
	if len(data) == 0 {
		return 0, 0, emptyRule, nil
	}

	leftRaw, w := lookup(data)
	if leftRaw == 0 && w > 0 {
		leftRaw = lookupProperty(data)
	}
	if w == 0 {
		if !atEOF {
			return 0, 0, emptyRule, nil
		}
		return len(data), breakMandatory, "LB3", nil
	}

	pos := w // LB2: sot ×
	left := lbClass(leftRaw, data)
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
		if rightRaw == 0 && rw > 0 {
			rightRaw = lookupProperty(data[pos:])
		}
		if rw == 0 {
			if !atEOF {
				return 0, 0, emptyRule, nil
			}
			return len(data), breakMandatory, "LB3", nil
		}
		right := lbClass(rightRaw, data[pos:])

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
				// Treat CM/ZWJ as part of the preceding class X; do not
				// advance left-tracking state to CM/ZWJ itself.
				pos += rw
				continue
			}
			// LB10: remaining CM/ZWJ resolve to AL.
			right = _AL
		}
		// LB10: remaining CM/ZWJ on the left side also resolve to AL.
		if left.is(_CM | _ZWJ) {
			left = _AL
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

		// LB15c: SP ÷ IS NU
		if left == _SP && right == _IS {
			next := property(0)
			if pos+rw < len(data) {
				nr, nw := lookup(data[pos+rw:])
				if nr == 0 && nw > 0 {
					nr = lookupProperty(data[pos+rw:])
				}
				if nw > 0 {
					next = lbClass(nr, data[pos+rw:])
				}
			}
			if next == _NU {
				return pos, breakOpportunity, "LB15c", nil
			}
		}

		// LB15d: × IS
		if right == _IS {
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

		// LB15a approximation: (sot|BK|CR|LF|NL|OP|QU|GL|SP|ZW) [Pi&QU] SP* ×
		if left == _SP && lb15aNoBreak(data, pos) {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB15b approximation: × [Pf&QU] (SP|GL|WJ|CL|QU|CP|EX|IS|SY|BK|CR|LF|NL|ZW|eot)
		//
		// We do not yet distinguish Pi/Pf subtypes inside QU; this approximation
		// applies the Pf-side behavior to QU when right-context matches.
		if right == _QU {
			next := property(0) // 0 means eot/unknown here.
			if pos+rw < len(data) {
				nr, nw := lookup(data[pos+rw:])
				if nr == 0 && nw > 0 {
					nr = lookupProperty(data[pos+rw:])
				}
				if nw > 0 {
					next = lbClass(nr, data[pos+rw:])
				}
			}
			if isPfQuote(data[pos:]) && (next == 0 || next.is(_SP|_GL|_WJ|_CL|_QU|_CP|_EX|_IS|_SY|_BK|_CR|_LF|_NL|_ZW)) {
				prev, left, pos = step(prev, left, right, pos, rw)
				lastNonSP = updateLastNonSP(lastNonSP, left)
				riRun = updateRIRun(riRun, left)
				continue
			}
		}

		// LB28a: Brahmic orthographic syllable no-break patterns.
		// Note: this currently covers class-based AK/AP/AS/VF/VI interactions.
		leftIsAKLike := left.is(_AK|_AS) || leftBaseIsDottedCircle(data, pos)
		rightIsAKLike := right.is(_AK|_AS) || isDottedCircle(data[pos:])
		next := property(0)
		if pos+rw < len(data) {
			nr, nw := lookup(data[pos+rw:])
			if nr == 0 && nw > 0 {
				nr = lookupProperty(data[pos+rw:])
			}
			if nw > 0 {
				next = lbClass(nr, data[pos+rw:])
			}
		}
		if (left == _AP && rightIsAKLike) ||
			(leftIsAKLike && right.is(_VF|_VI)) ||
			(left == _VI && rightIsAKLike && (prev.is(_AK|_AS) || leftPrecededByDottedCircle(data, pos))) ||
			(leftIsAKLike && rightIsAKLike && next == _VF) {
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

		// LB20a: (sot|BK|CR|LF|NL|SP|ZW|CB|GL) (HY|HH) × (AL|HL)
		if left.is(_HY|_HH) && right.is(_AL|_HL) && (prev == 0 || prev.is(_BK|_CR|_LF|_NL|_SP|_ZW|_CB|_GL)) {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB21: × BA/HH/HY/NS, BB ×
		if right.is(_BA|_HH|_HY|_NS) || left == _BB {
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
		// LB25 additions:
		// NU ... CL/CP × PR/PO
		if left.is(_CL|_CP) && right.is(_PR|_PO) && prev == _NU {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}
		// PR/PO × OP (NU | IS NU)
		if left.is(_PR|_PO) && right == _OP {
			next := property(0)
			next2 := property(0)
			if pos+rw < len(data) {
				nr, nw := lookup(data[pos+rw:])
				if nr == 0 && nw > 0 {
					nr = lookupProperty(data[pos+rw:])
				}
				if nw > 0 {
					next = lbClass(nr, data[pos+rw:])
					if pos+rw+nw < len(data) {
						n2r, n2w := lookup(data[pos+rw+nw:])
						if n2r == 0 && n2w > 0 {
							n2r = lookupProperty(data[pos+rw+nw:])
						}
						if n2w > 0 {
							next2 = lbClass(n2r, data[pos+rw+nw:])
						}
					}
				}
			}
			if next == _NU || (next == _IS && next2 == _NU) {
				prev, left, pos = step(prev, left, right, pos, rw)
				lastNonSP = updateLastNonSP(lastNonSP, left)
				riRun = updateRIRun(riRun, left)
				continue
			}
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

		// LB30 approximation: (AL|HL|NU) × [OP-$EastAsian], CP × (AL|HL|NU).
		// CP East Asian exclusion and full East Asian set coverage are still partial.
		if (left.is(_AL|_HL|_NU) && right == _OP && !isEastAsianOP(data[pos:])) || (left == _CP && right.is(_AL|_HL|_NU)) {
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

		// LB30b: EB × EM, [Extended_Pictographic & Cn] × EM.
		if (left == _EB && right == _EM) || (right == _EM && leftBaseIsExtPictCn(data, pos)) {
			prev, left, pos = step(prev, left, right, pos, rw)
			lastNonSP = updateLastNonSP(lastNonSP, left)
			riRun = updateRIRun(riRun, left)
			continue
		}

		// LB31: default break opportunity.
		return pos, breakOpportunity, "LB31", nil
	}
}

func lbClass[T ~string | ~[]byte](p property, in T) property {
	switch p {
	case _AI, _SG, _XX:
		return _AL
	case _CJ:
		return _NS
	case _SA:
		// SA maps to CM for combining marks, otherwise AL.
		if isCombiningMark(in) {
			return _CM
		}
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

	// NU (SY|IS)* × (PR|PO)
	if left.is(_SY|_IS) && right.is(_PR|_PO) && prev == _NU {
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

func isPfQuote[T ~string | ~[]byte](in T) bool {
	switch x := any(in).(type) {
	case string:
		r, _ := utf8.DecodeRuneInString(x)
		return unicode.Is(unicode.Pf, r)
	case []byte:
		r, _ := utf8.DecodeRune(x)
		return unicode.Is(unicode.Pf, r)
	default:
		return false
	}
}

func isPiQuote[T ~string | ~[]byte](in T) bool {
	switch x := any(in).(type) {
	case string:
		r, _ := utf8.DecodeRuneInString(x)
		return unicode.Is(unicode.Pi, r)
	case []byte:
		r, _ := utf8.DecodeRune(x)
		return unicode.Is(unicode.Pi, r)
	default:
		return false
	}
}

func isCombiningMark[T ~string | ~[]byte](in T) bool {
	switch x := any(in).(type) {
	case string:
		r, _ := utf8.DecodeRuneInString(x)
		return unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r)
	case []byte:
		r, _ := utf8.DecodeRune(x)
		return unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r)
	default:
		return false
	}
}

func isEastAsianOP[T ~string | ~[]byte](in T) bool {
	var r rune
	switch x := any(in).(type) {
	case string:
		r, _ = utf8.DecodeRuneInString(x)
	case []byte:
		r, _ = utf8.DecodeRune(x)
	default:
		return false
	}

	// Common East Asian opening punctuation used by LB30 exclusion.
	switch r {
	case 0x2329, // LEFT-POINTING ANGLE BRACKET
		0x3008, 0x300A, 0x300C, 0x300E, 0x3010, 0x3014, 0x3016, 0x3018,
		0x301A, 0xFE59, 0xFE5B, 0xFE5D, 0xFF08, 0xFF3B, 0xFF5B, 0xFF5F, 0xFF62:
		return true
	default:
		return false
	}
}

func isDottedCircle[T ~string | ~[]byte](in T) bool {
	switch x := any(in).(type) {
	case string:
		r, _ := utf8.DecodeRuneInString(x)
		return r == '\u25cc'
	case []byte:
		r, _ := utf8.DecodeRune(x)
		return r == '\u25cc'
	default:
		return false
	}
}

func leftBaseIsDottedCircle[T ~string | ~[]byte](data T, boundary int) bool {
	i := boundary
	for i > 0 {
		j := i - 1
		for j > 0 && (data[j]&0xC0) == 0x80 {
			j--
		}

		raw, w := lookup(data[j:i])
		if raw == 0 && w > 0 {
			raw = lookupProperty(data[j:i])
		}
		if w == 0 {
			return false
		}
		c := lbClass(raw, data[j:i])
		if c.is(_CM | _ZWJ) {
			i = j
			continue
		}
		return isDottedCircle(data[j:i])
	}
	return false
}

func leftPrecededByDottedCircle[T ~string | ~[]byte](data T, boundary int) bool {
	// Locate the immediate left base class at boundary.
	i := boundary
	for i > 0 {
		j, c, _ := prevRuneClass(data, i)
		if j < 0 {
			return false
		}
		if c.is(_CM | _ZWJ) {
			i = j
			continue
		}

		// Find the preceding non-CM/ZWJ base and test if it's dotted circle.
		k := j
		for k > 0 {
			h, pc, _ := prevRuneClass(data, k)
			if h < 0 {
				return false
			}
			if pc.is(_CM | _ZWJ) {
				k = h
				continue
			}
			return isDottedCircle(data[h:k])
		}
		return false
	}
	return false
}

func leftBaseIsExtPictCn[T ~string | ~[]byte](data T, boundary int) bool {
	i := boundary
	for i > 0 {
		j := i - 1
		for j > 0 && (data[j]&0xC0) == 0x80 {
			j--
		}
		raw, w := lookup(data[j:i])
		if raw == 0 && w > 0 {
			raw = lookupProperty(data[j:i])
		}
		if w == 0 {
			return false
		}
		if raw.is(_CM | _ZWJ) {
			i = j
			continue
		}
		if raw != _XX && raw != _ID {
			return false
		}

		r := firstRune(data[j:i])
		// Conservative approximation of ExtPict-unassigned space used in tests.
		return r >= 0x1F000 && r <= 0x1FFFD
	}
	return false
}

func lb15aNoBreak[T ~string | ~[]byte](data T, boundary int) bool {
	i := boundary
	// Walk left over SP* and CM/ZWJ that attach to the quote cluster.
	for i > 0 {
		j, c, _ := prevRuneClass(data, i)
		if j < 0 {
			return false
		}
		if c == _SP || c.is(_CM|_ZWJ) {
			i = j
			continue
		}
		// Immediate non-space left item must be a Pi quote.
		if c != _QU || !isPiQuote(data[j:i]) {
			return false
		}
		// Context before the quote must be in the allowed set, or sot.
		if j == 0 {
			return true
		}
		k, pc, _ := prevRuneClass(data, j)
		if k < 0 {
			return false
		}
		return pc.is(_BK | _CR | _LF | _NL | _OP | _QU | _GL | _SP | _ZW)
	}
	return false
}

func prevRuneClass[T ~string | ~[]byte](data T, end int) (start int, class property, r rune) {
	if end <= 0 {
		return -1, 0, utf8.RuneError
	}
	j := end - 1
	for j > 0 && (data[j]&0xC0) == 0x80 {
		j--
	}
	raw, w := lookup(data[j:end])
	if raw == 0 && w > 0 {
		raw = lookupProperty(data[j:end])
	}
	if w == 0 {
		return -1, 0, utf8.RuneError
	}
	return j, lbClass(raw, data[j:end]), firstRune(data[j:end])
}

func firstRune[T ~string | ~[]byte](in T) rune {
	switch x := any(in).(type) {
	case string:
		r, _ := utf8.DecodeRuneInString(x)
		return r
	case []byte:
		r, _ := utf8.DecodeRune(x)
		return r
	default:
		return utf8.RuneError
	}
}
