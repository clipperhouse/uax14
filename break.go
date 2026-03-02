package uax14

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
	var lastExSP property          // "last excluding SP"
	var beforeLastExSP property    // predecessor of lastExSP, with CM/ZWJ ignored
	var lastExCMZWJ property       // "last excluding CM and ZWJ"
	var lastExCMZWJSP property     // "last excluding CM and ZWJ and SP"
	var lastExSYIS property        // "last excluding SY and IS", with CM/ZWJ ignored
	var beforeLastExSYIS property  // predecessor of lastExSYIS
	var regionalIndicatorCount int // count of consecutive RI (excluding CM/ZWJ)

	current, w := lookupProperty(data[pos:])
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

		// Remember previous properties to avoid lookbacks
		last := current
		prevExCMZWJ := lastExCMZWJ
		if !last.is(_SP) {
			beforeLastExSP = prevExCMZWJ
			lastExSP = last
		}
		if !last.is(_CM | _ZWJ) {
			lastExCMZWJ = last
			if last.is(_RI) {
				regionalIndicatorCount++
			} else {
				regionalIndicatorCount = 0
			}
		}
		if !last.is(_SP | _CM | _ZWJ) {
			lastExCMZWJSP = last
		}
		if !lastExCMZWJ.is(_SY | _IS) {
			beforeLastExSYIS = lastExSYIS
			lastExSYIS = lastExCMZWJ
		}

		current, w = lookup(data[pos:])
		if w == 0 {
			pos = len(data)
			return pos, breakMandatory
		}
		if current == 0 {
			current = _AL
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
		if lastExCMZWJSP.is(_OP) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB15a
		// (sot | BK | CR | LF | NL | OP | QU | GL | SP | ZW) [\p{Pi}&QU] SP* ×
		if lastExCMZWJSP.is(_PI) &&
			(beforeLastExSP == 0 || beforeLastExSP.is(_BK|_CR|_LF|_NL|_OP|_QU|_GL|_SP|_ZW)) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB15b
		// × [\p{Pf}&QU] (SP | GL | WJ | CL | QU | CP | EX | IS | SY | BK | CR | LF | NL | ZW | eot)
		if current.is(_PF) && current.is(_QU) {
			var next property
			if pos+w < len(data) {
				next, _ = lookupProperty(data[pos+w:])
			}
			if next == 0 || next.is(_SP|_GL|_WJ|_CL|_QU|_CP|_EX|_IS|_SY|_BK|_CR|_LF|_NL|_ZW) {
				pos += w
				continue
			}
		}

		// https://www.unicode.org/reports/tr14/#LB15c
		// SP ÷ IS NU
		if last.is(_SP) && current.is(_IS) {
			var next property
			if pos+w < len(data) {
				next, _ = lookupProperty(data[pos+w:])
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
		if lastExCMZWJSP.is(_CL|_CP) && current.is(_NS) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB17
		// B2 SP* × B2
		if lastExCMZWJSP.is(_B2) && current.is(_B2) {
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
			var next property
			if pos+w < len(data) {
				next, _ = lookupProperty(data[pos+w:])
			}

			noBreakBeforeQU := current.is(_QU) && (!last.is(_EA) || !next.is(_EA))
			noBreakAfterQU := last.is(_QU) && (!current.is(_EA) || prevExCMZWJ == 0 || !prevExCMZWJ.is(_EA))
			if noBreakBeforeQU || noBreakAfterQU {
				pos += w
				continue
			}
		}

		// https://www.unicode.org/reports/tr14/#LB20
		// ÷ CB, CB ÷
		if (current | lastExCMZWJ).is(_CB) {
			return pos, breakOpportunity
		}

		// https://www.unicode.org/reports/tr14/#LB20a
		// (sot | BK | CR | LF | NL | SP | ZW | CB | GL) (HY | HH) × (AL | HL)
		if last.is(_HY|_HH) && current.is(_AL|_HL) &&
			(prevExCMZWJ == 0 || prevExCMZWJ.is(_BK|_CR|_LF|_NL|_SP|_ZW|_CB|_GL)) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB21
		// × BA
		// × HH
		// × HY
		// × NS
		// BB ×
		if current.is(_BA|_HH|_HY|_NS) || lastExCMZWJ.is(_BB) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB21a
		// HL (HY | HH) × [^HL]
		if prevExCMZWJ.is(_HL) && last.is(_HY|_HH) && !current.is(_HL) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB21b
		// SY × HL
		if lastExCMZWJ.is(_SY) && current.is(_HL) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB22
		// × IN
		if current.is(_IN) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB23
		// (AL | HL) × NU
		// NU × (AL | HL)
		if (lastExCMZWJ.is(_AL|_HL) && current.is(_NU)) || (lastExCMZWJ.is(_NU) && current.is(_AL|_HL)) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB23a
		// PR × (ID | EB | EM)
		// (ID | EB | EM) × PO
		if (lastExCMZWJ.is(_PR) && current.is(_ID|_EB|_EM)) || (lastExCMZWJ.is(_ID|_EB|_EM) && current.is(_PO)) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB24
		// (PR | PO) × (AL | HL)
		// (AL | HL) × (PR | PO)
		if (lastExCMZWJ.is(_PR|_PO) && current.is(_AL|_HL)) || (lastExCMZWJ.is(_AL|_HL) && current.is(_PR|_PO)) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB25
		// NU ( SY | IS )* CL × PO
		// NU ( SY | IS )* CP × PO
		// NU ( SY | IS )* CL × PR
		// NU ( SY | IS )* CP × PR
		// NU ( SY | IS )* × PO
		// NU ( SY | IS )* × PR
		// PO × OP NU
		// PO × OP IS NU
		// PO × NU
		// PR × OP NU
		// PR × OP IS NU
		// PR × NU
		// HY × NU
		// IS × NU
		// NU ( SY | IS )* × NU
		if lastExCMZWJ.is(_NU) && current.is(_SY|_IS|_CL|_CP) {
			pos += w
			continue
		}
		if current.is(_PO|_PR) &&
			((lastExCMZWJ.is(_NU)) || (lastExCMZWJ.is(_SY|_IS) && lastExSYIS.is(_NU))) {
			pos += w
			continue
		}
		if lastExCMZWJ.is(_CL|_CP) && current.is(_PO|_PR) && beforeLastExSYIS.is(_NU) {
			pos += w
			continue
		}
		if lastExCMZWJ.is(_PO|_PR) && current.is(_OP) {
			var next, next2 property
			if pos+w < len(data) {
				next, _ = lookupProperty(data[pos+w:])
				if pos+w+w < len(data) {
					next2, _ = lookupProperty(data[pos+w+w:])
				}
			}
			if next.is(_NU) || (next.is(_IS) && next2.is(_NU)) {
				pos += w
				continue
			}
		}
		if current.is(_NU) &&
			(lastExCMZWJ.is(_PO|_PR|_HY|_IS|_NU) ||
				(lastExCMZWJ.is(_SY|_IS|_CL|_CP) && lastExSYIS.is(_NU))) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB26
		// JL × (JL | JV | H2 | H3)
		// (JV | H2) × (JV | JT)
		// (JT | H3) × JT
		if (lastExCMZWJ.is(_JL) && current.is(_JL|_JV|_H2|_H3)) ||
			(lastExCMZWJ.is(_JV|_H2) && current.is(_JV|_JT)) ||
			(lastExCMZWJ.is(_JT|_H3) && current.is(_JT)) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB27
		// (JL | JV | JT | H2 | H3) × PO
		// PR × (JL | JV | JT | H2 | H3)
		if (lastExCMZWJ.is(_JL|_JV|_JT|_H2|_H3) && current.is(_PO)) ||
			(lastExCMZWJ.is(_PR) && current.is(_JL|_JV|_JT|_H2|_H3)) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB28
		// (AL | HL) × (AL | HL)
		if lastExCMZWJ.is(_AL|_HL) && current.is(_AL|_HL) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB28a
		// AP × (AK | [◌] | AS)
		// (AK | [◌] | AS) × (VF | VI)
		// (AK | [◌] | AS) VI × (AK | [◌])
		// (AK | [◌] | AS) × (AK | [◌] | AS) VF
		if (lastExCMZWJ.is(_AP) && current.is(_AK|_AS|_DC)) ||
			(lastExCMZWJ.is(_AK|_AS|_DC) && current.is(_VF|_VI)) ||
			(lastExCMZWJ.is(_VI) && current.is(_AK|_AS|_DC) && prevExCMZWJ.is(_AK|_AS|_DC)) {
			pos += w
			continue
		}
		if lastExCMZWJ.is(_AK|_AS|_DC) && current.is(_AK|_AS|_DC) {
			if pos+w < len(data) {
				next, _ := lookupProperty(data[pos+w:])
				if next.is(_VF) {
					pos += w
					continue
				}
			}
		}

		// https://www.unicode.org/reports/tr14/#LB29
		// IS × (AL | HL)
		if lastExCMZWJ.is(_IS) && current.is(_AL|_HL) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB30
		// (AL | HL | NU) × [OP-$EastAsian]
		// [CP-$EastAsian] × (AL | HL | NU)
		if (lastExCMZWJ.is(_AL|_HL|_NU) && current.is(_OP) && !current.is(_EA)) ||
			(lastExCMZWJ.is(_CP) && !lastExCMZWJ.is(_EA) && current.is(_AL|_HL|_NU)) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB30a
		// sot (RI RI)* RI × RI
		// [^RI] (RI RI)* RI × RI
		if lastExCMZWJ.is(_RI) && current.is(_RI) {
			odd := regionalIndicatorCount%2 == 1
			if odd {
				pos += w
				continue
			}
		}

		// https://www.unicode.org/reports/tr14/#LB30b
		// EB × EM
		// [\p{Extended_Pictographic}&\p{Cn}] × EM
		if (lastExCMZWJ.is(_EB) || lastExCMZWJ.is(_EPU)) && current.is(_EM) {
			pos += w
			continue
		}

		// https://www.unicode.org/reports/tr14/#LB31
		// ALL ÷
		// ÷ ALL
		return pos, breakOpportunity
	}
}
