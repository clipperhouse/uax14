package uax14

import "testing"

func TestSplitDecision_HardBreaks(t *testing.T) {
	got := collectDecisions("A\r\nB")
	if len(got) != 2 {
		t.Fatalf("got %d tokens, want 2", len(got))
	}
	if got[0].token != "A\r\n" || got[0].kind != breakMandatory || got[0].rule != "LB5" {
		t.Fatalf("token0 = %#v, want token=%q kind=%v rule=%q", got[0], "A\r\n", breakMandatory, "LB5")
	}
	if got[1].token != "B" || got[1].kind != breakMandatory || got[1].rule != "LB3" {
		t.Fatalf("token1 = %#v, want token=%q kind=%v rule=%q", got[1], "B", breakMandatory, "LB3")
	}
}

func TestSplitDecision_CombiningAndZWJ(t *testing.T) {
	comb := collectDecisions("A\u0301B")
	if len(comb) != 2 {
		t.Fatalf("combining: got %d tokens, want 2", len(comb))
	}
	if comb[0].token != "A\u0301" || comb[0].rule != "LB31" {
		t.Fatalf("combining token0 = %#v, want token=%q rule=%q", comb[0], "A\u0301", "LB31")
	}

	spaceComb := collectDecisions(" \u0301B")
	if len(spaceComb) != 3 {
		t.Fatalf("space+combining: got %d tokens, want 3", len(spaceComb))
	}
	if spaceComb[0].token != " " || spaceComb[0].rule != "LB18" {
		t.Fatalf("space+combining token0 = %#v, want token=%q rule=%q", spaceComb[0], " ", "LB18")
	}

	zwj := collectDecisions("A\u200dB")
	if len(zwj) != 1 {
		t.Fatalf("zwj: got %d tokens, want 1", len(zwj))
	}
	if zwj[0].token != "A\u200dB" || zwj[0].rule != "LB3" {
		t.Fatalf("zwj token0 = %#v, want token=%q rule=%q", zwj[0], "A\u200dB", "LB3")
	}
}

func TestSplitDecision_RIEmojiNumeric(t *testing.T) {
	ri := collectDecisions("🇺🇸🇨🇦")
	if len(ri) != 2 {
		t.Fatalf("ri: got %d tokens, want 2", len(ri))
	}
	if ri[0].token != "🇺🇸" || ri[0].rule != "LB30a" {
		t.Fatalf("ri token0 = %#v, want token=%q rule=%q", ri[0], "🇺🇸", "LB30a")
	}

	emoji := collectDecisions("👍🏻x")
	if len(emoji) != 2 {
		t.Fatalf("emoji: got %d tokens, want 2", len(emoji))
	}
	if emoji[0].token != "👍🏻" {
		t.Fatalf("emoji token0 = %#v, want token=%q", emoji[0], "👍🏻")
	}

	numeric := collectDecisions("1,2 3/4")
	if len(numeric) != 2 {
		t.Fatalf("numeric: got %d tokens, want 2", len(numeric))
	}
	if numeric[0].token != "1,2 " || numeric[0].rule != "LB18" {
		t.Fatalf("numeric token0 = %#v, want token=%q rule=%q", numeric[0], "1,2 ", "LB18")
	}
}

type decision struct {
	token string
	kind  breakKind
	rule  string
}

func collectDecisions(s string) []decision {
	out := make([]decision, 0, len(s))
	remaining := s

	for len(remaining) > 0 {
		advance, kind, rule, err := splitDecision(remaining, true)
		if err != nil {
			panic(err)
		}
		if advance <= 0 || advance > len(remaining) {
			panic("invalid split advance")
		}
		out = append(out, decision{
			token: remaining[:advance],
			kind:  kind,
			rule:  rule,
		})
		remaining = remaining[advance:]
	}

	return out
}
