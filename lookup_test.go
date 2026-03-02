package uax14

import (
	"testing"
)

func TestLookup_RepresentativeClasses(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want property
	}{
		{name: "AL ASCII letter", in: "A", want: _AL},
		{name: "SP space", in: " ", want: _SP},
		{name: "LF line feed", in: "\n", want: _LF},
		{name: "CR carriage return", in: "\r", want: _CR},
		{name: "NU ASCII digit", in: "0", want: _NU},
		{name: "OP opening punctuation", in: "(", want: _OP},
		{name: "CP closing punctuation", in: ")", want: _CP},
		{name: "HY hyphen", in: "-", want: _HY},
		{name: "IS infix separator", in: ",", want: _IS},
		{name: "SY solidus", in: "/", want: _SY},
		{name: "ZW zero width space", in: "\u200b", want: _ZW},
		{name: "ZWJ zero width joiner", in: "\u200d", want: _ZWJ},
		{name: "CM combining mark", in: "\u0301", want: _CM},
		{name: "ID CJK ideograph", in: "中", want: _ID},
		{name: "RI regional indicator", in: "🇺", want: _RI},
		{name: "AL default unassigned", in: "\u0378", want: _AL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := lookupProperty(tt.in)
			if !got.is(tt.want) {
				t.Fatalf("lookupProperty(%q) = %#x, want to include %#x", tt.in, got, tt.want)
			}
		})
	}
}

func TestLookup_StringAndBytesParity(t *testing.T) {
	tests := []string{
		"A",
		"\n",
		"中",
		"\u200d",
		"🇺",
		"\u0378",
	}

	for _, in := range tests {
		gotS, _ := lookupProperty(in)
		gotB, _ := lookupProperty([]byte(in))
		if gotS != gotB {
			t.Fatalf("lookupProperty parity mismatch for %q: string=%#x bytes=%#x", in, gotS, gotB)
		}
	}
}

func TestLookup_EastAsianWidthBit(t *testing.T) {
	got, _ := lookupProperty("中")
	if !got.is(_EA) {
		t.Fatalf("lookupProperty(%q) should include _EA", "中")
	}
	got, _ = lookupProperty("A")
	if !got.is(_EA) {
		t.Fatalf("lookupProperty(%q) should not include _EA", "A")
	}
}

func TestLookup_RawUTF8EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		in    []byte
		want  property
		wantN int
	}{
		{
			name:  "truncated two-byte sequence",
			in:    []byte{0xC3},
			want:  0,
			wantN: 0,
		},
		{
			name:  "truncated three-byte sequence",
			in:    []byte{0xE2, 0x82},
			want:  0,
			wantN: 0,
		},
		{
			name:  "invalid continuation byte",
			in:    []byte{0xE2, 0x28, 0xA1},
			want:  0,
			wantN: 1,
		},
		{
			name:  "illegal starter byte",
			in:    []byte{0x80},
			want:  0,
			wantN: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, n := lookup(tt.in)
			if got != tt.want || n != tt.wantN {
				t.Fatalf("lookup(%v) = (%#x, %d), want (%#x, %d)", tt.in, got, n, tt.want, tt.wantN)
			}
		})
	}
}
