// Package main generates line-break trie data and conformance test cases.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/clipperhouse/uax14/internal/gen/triegen"
)

const (
	unicodeVersion          = "17.0.0"
	defaultLineBreakURL     = "https://unicode.org/Public/" + unicodeVersion + "/ucd/LineBreak.txt"
	defaultGeneralCategoryURL = "https://unicode.org/Public/" + unicodeVersion + "/ucd/extracted/DerivedGeneralCategory.txt"
	defaultEastAsianWidthURL = "https://unicode.org/Public/" + unicodeVersion + "/ucd/EastAsianWidth.txt"
	defaultLineBreakTestURL = "https://unicode.org/Public/" + unicodeVersion + "/ucd/auxiliary/LineBreakTest.txt"
	outputFilename          = "../../trie.go"
	outputTestFilename      = "../../linebreak_conformance_generated_test.go"
	cacheDir                = "cache"
)

var versionRE = regexp.MustCompile(`LineBreak-([0-9]+(?:\.[0-9]+)*)\.txt`)

type record struct {
	lo    rune
	hi    rune
	class string
}

type conformanceCase struct {
	lineNo       int
	input        []byte
	breakOffsets []int
	comment      string
}

func main() {
	var inputPath string
	var sourceURL string
	var categoryInputPath string
	var categoryURL string
	var eastAsianWidthInputPath string
	var eastAsianWidthURL string
	var testInputPath string
	var testURL string
	var refresh bool

	flag.StringVar(&inputPath, "input", "", "path to local LineBreak.txt file (optional)")
	flag.StringVar(&sourceURL, "url", defaultLineBreakURL, "LineBreak.txt URL")
	flag.StringVar(&categoryInputPath, "gcinput", "", "path to local DerivedGeneralCategory.txt file (optional)")
	flag.StringVar(&categoryURL, "gcurl", defaultGeneralCategoryURL, "DerivedGeneralCategory.txt URL")
	flag.StringVar(&eastAsianWidthInputPath, "eawinput", "", "path to local EastAsianWidth.txt file (optional)")
	flag.StringVar(&eastAsianWidthURL, "eawurl", defaultEastAsianWidthURL, "EastAsianWidth.txt URL")
	flag.StringVar(&testInputPath, "testinput", "", "path to local LineBreakTest.txt file (optional)")
	flag.StringVar(&testURL, "testurl", defaultLineBreakTestURL, "LineBreakTest.txt URL")
	flag.BoolVar(&refresh, "refresh", false, "refresh local cache from network")
	flag.Parse()

	content, sourceLabel, err := loadData(inputPath, sourceURL, cachePath("LineBreak.txt"), refresh)
	if err != nil {
		fail(err)
	}

	version := unicodeVersion
	if extracted := extractVersion(content); extracted != "unknown" && extracted != unicodeVersion {
		fail(fmt.Errorf("LineBreak.txt version mismatch: got %s, expected %s", extracted, unicodeVersion))
	}
	records, err := parseLineBreak(content)
	if err != nil {
		fail(err)
	}

	categoryContent, categorySourceLabel, err := loadData(categoryInputPath, categoryURL, cachePath("DerivedGeneralCategory.txt"), refresh)
	if err != nil {
		fail(err)
	}
	categoryRecords, err := parseLineBreak(categoryContent)
	if err != nil {
		fail(err)
	}
	quoteCategoryRecords := selectQuoteCategoryRecords(categoryRecords)

	eastAsianWidthContent, eastAsianWidthSourceLabel, err := loadData(eastAsianWidthInputPath, eastAsianWidthURL, cachePath("EastAsianWidth.txt"), refresh)
	if err != nil {
		fail(err)
	}
	eastAsianWidthRecords, err := parseLineBreak(eastAsianWidthContent)
	if err != nil {
		fail(err)
	}
	eastAsianRecords := selectEastAsianRecords(eastAsianWidthRecords)

	extPictUnassignedRecords := selectExtendedPictographicUnassignedRecords(categoryRecords)

	src, err := generateTrieSource(records, quoteCategoryRecords, eastAsianRecords, extPictUnassignedRecords, version, sourceLabel, categorySourceLabel, eastAsianWidthSourceLabel)
	if err != nil {
		fail(err)
	}
	formatted, err := format.Source(src)
	if err != nil {
		fail(fmt.Errorf("format trie file: %w", err))
	}
	if err := os.WriteFile(outputFilename, formatted, 0o644); err != nil {
		fail(fmt.Errorf("write %s: %w", outputFilename, err))
	}

	testContent, testSourceLabel, err := loadData(testInputPath, testURL, cachePath("LineBreakTest.txt"), refresh)
	if err != nil {
		fail(err)
	}
	tests, err := parseLineBreakTests(testContent)
	if err != nil {
		fail(err)
	}
	testSrc, err := generateConformanceTestsSource(tests, testSourceLabel)
	if err != nil {
		fail(err)
	}
	testFormatted, err := format.Source(testSrc)
	if err != nil {
		fail(fmt.Errorf("format conformance test file: %w", err))
	}
	if err := os.WriteFile(outputTestFilename, testFormatted, 0o644); err != nil {
		fail(fmt.Errorf("write %s: %w", outputTestFilename, err))
	}
}

func loadData(inputPath, sourceURL, cachedPath string, refresh bool) ([]byte, string, error) {
	if inputPath != "" {
		b, err := os.ReadFile(inputPath)
		if err != nil {
			return nil, "", fmt.Errorf("read input file: %w", err)
		}
		return b, inputPath, nil
	}

	if !refresh && cachedPath != "" {
		b, err := os.ReadFile(cachedPath)
		if err == nil {
			return b, cachedPath, nil
		}
		if !os.IsNotExist(err) {
			return nil, "", fmt.Errorf("read cache %s: %w", cachedPath, err)
		}
	}

	req, err := http.NewRequest(http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("download %s: %w", sourceURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download %s: status %s", sourceURL, resp.Status)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read response body: %w", err)
	}

	if cachedPath != "" {
		if err := os.MkdirAll(filepath.Dir(cachedPath), 0o755); err != nil {
			return nil, "", fmt.Errorf("create cache dir for %s: %w", cachedPath, err)
		}
		if err := os.WriteFile(cachedPath, b, 0o644); err != nil {
			return nil, "", fmt.Errorf("write cache %s: %w", cachedPath, err)
		}
	}

	return b, sourceURL, nil
}

func cachePath(filename string) string {
	return filepath.Join(cacheDir, unicodeVersion, filename)
}

func extractVersion(content []byte) string {
	m := versionRE.FindSubmatch(content)
	if len(m) < 2 {
		return "unknown"
	}
	return string(m[1])
}

func parseLineBreak(content []byte) ([]record, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)

	records := make([]record, 0, 4096)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		semi := strings.IndexRune(line, ';')
		if semi == -1 {
			return nil, fmt.Errorf("line %d: missing ';'", lineNo)
		}

		left := strings.TrimSpace(line[:semi])
		right := strings.TrimSpace(line[semi+1:])
		if i := strings.IndexRune(right, '#'); i >= 0 {
			right = strings.TrimSpace(right[:i])
		}

		if left == "" || right == "" {
			return nil, fmt.Errorf("line %d: malformed entry", lineNo)
		}

		lo, hi, err := parseRange(left)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}

		records = append(records, record{lo: lo, hi: hi, class: right})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan file: %w", err)
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].lo != records[j].lo {
			return records[i].lo < records[j].lo
		}
		if records[i].hi != records[j].hi {
			return records[i].hi < records[j].hi
		}
		return records[i].class < records[j].class
	})

	return records, nil
}

func parseLineBreakTests(content []byte) ([]conformanceCase, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 0, 4096), 8*1024*1024)

	tests := make([]conformanceCase, 0, 20000)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		raw := line
		comment := ""
		if i := strings.IndexByte(raw, '#'); i >= 0 {
			comment = strings.TrimSpace(raw[i+1:])
			raw = strings.TrimSpace(raw[:i])
		}
		if raw == "" {
			continue
		}

		fields := strings.Fields(raw)
		if len(fields) < 3 || len(fields)%2 == 0 {
			return nil, fmt.Errorf("line %d: invalid field layout", lineNo)
		}
		if fields[0] != "÷" && fields[0] != "×" {
			return nil, fmt.Errorf("line %d: invalid leading marker %q", lineNo, fields[0])
		}

		tc := conformanceCase{
			lineNo:       lineNo,
			input:        make([]byte, 0, len(fields)*2),
			breakOffsets: make([]int, 0, len(fields)/2+1),
			comment:      comment,
		}
		if fields[0] == "÷" {
			tc.breakOffsets = append(tc.breakOffsets, 0)
		}

		for i := 1; i < len(fields); i += 2 {
			r, err := parseHexRune(fields[i])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNo, err)
			}
			if !utf8.ValidRune(r) {
				return nil, fmt.Errorf("line %d: invalid rune %q", lineNo, fields[i])
			}
			tc.input = utf8.AppendRune(tc.input, r)

			marker := fields[i+1]
			if marker != "÷" && marker != "×" {
				return nil, fmt.Errorf("line %d: invalid marker %q", lineNo, marker)
			}
			if marker == "÷" {
				tc.breakOffsets = append(tc.breakOffsets, len(tc.input))
			}
		}

		tests = append(tests, tc)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan line break tests: %w", err)
	}
	return tests, nil
}

func parseRange(s string) (rune, rune, error) {
	if strings.Contains(s, "..") {
		parts := strings.SplitN(s, "..", 2)
		if len(parts) != 2 {
			return 0, 0, fmt.Errorf("invalid range %q", s)
		}
		lo, err := parseHexRune(parts[0])
		if err != nil {
			return 0, 0, err
		}
		hi, err := parseHexRune(parts[1])
		if err != nil {
			return 0, 0, err
		}
		if hi < lo {
			return 0, 0, fmt.Errorf("descending range %q", s)
		}
		return lo, hi, nil
	}

	r, err := parseHexRune(s)
	if err != nil {
		return 0, 0, err
	}
	return r, r, nil
}

func parseHexRune(s string) (rune, error) {
	u, err := strconv.ParseUint(strings.TrimSpace(s), 16, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid code point %q: %w", s, err)
	}
	if u > 0x10FFFF {
		return 0, fmt.Errorf("code point out of range %q", s)
	}
	return rune(u), nil
}

func generateTrieSource(records, quoteCategories, eastAsian, extPictUnassigned []record, unicodeVersion, sourceLabel, categorySourceLabel, eastAsianSourceLabel string) ([]byte, error) {
	allRecords := make([]record, 0, len(records)+len(quoteCategories)+len(eastAsian)+len(extPictUnassigned)+1)
	allRecords = append(allRecords, records...)
	allRecords = append(allRecords, quoteCategories...)
	allRecords = append(allRecords, eastAsian...)
	allRecords = append(allRecords, extPictUnassigned...)
	// LB28a treats dotted circle specially; keep this as a dedicated bit.
	allRecords = append(allRecords, record{lo: 0x25CC, hi: 0x25CC, class: "DC"})

	classes := uniqueClasses(allRecords)

	iotasByClass := map[string]uint64{}
	for i, c := range classes {
		iotasByClass[c] = 1 << i
	}

	trie := triegen.NewTrie("lineBreak")
	// Build and merge all per-rune property bits before writing the trie.
	runeValues := map[rune]uint64{}
	for _, rec := range allRecords {
		v := iotasByClass[rec.class]
		for r := rec.lo; r <= rec.hi; r++ {
			if r >= 0xD800 && r <= 0xDFFF {
				continue
			}
			runeValues[r] |= v
		}
	}
	for r, v := range runeValues {
		trie.Insert(r, v)
	}

	buf := bytes.Buffer{}
	fmt.Fprintln(&buf, "package uax14")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "// Code generated by internal/gen; DO NOT EDIT.")
	fmt.Fprintf(&buf, "// Source: %s\n", sourceLabel)
	fmt.Fprintf(&buf, "// Source: %s\n", categorySourceLabel)
	fmt.Fprintf(&buf, "// Source: %s\n", eastAsianSourceLabel)
	fmt.Fprintf(&buf, "// Unicode LineBreak version: %s\n\n", unicodeVersion)
	fmt.Fprintln(&buf, "type property uint64")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "const (")
	for i, c := range classes {
		if i == 0 {
			fmt.Fprintf(&buf, "\t_%s property = 1 << iota\n", c)
		} else {
			fmt.Fprintf(&buf, "\t_%s\n", c)
		}
	}
	fmt.Fprintln(&buf, ")")
	fmt.Fprintln(&buf)

	_, err := triegen.Gen(&buf, "lineBreak", []*triegen.Trie{trie})
	if err != nil {
		return nil, err
	}

	b := buf.Bytes()
	typename := "lineBreakTrie"
	b = bytes.ReplaceAll(b, []byte("type "+typename+" struct"), []byte("// type "+typename+" struct"))
	b = bytes.ReplaceAll(b, []byte("(t *"+typename+") lookup(s []byte)"), []byte("lookup[T ~string | ~[]byte](s T)"))
	b = bytes.ReplaceAll(b, []byte("(t *"+typename+") lookupValue"), []byte("lookupValue"))
	b = bytes.ReplaceAll(b, []byte("t.lookupValue("), []byte("lookupValue("))

	return b, nil
}

func generateConformanceTestsSource(tests []conformanceCase, sourceLabel string) ([]byte, error) {
	buf := bytes.Buffer{}
	fmt.Fprintln(&buf, "package uax14")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "// Code generated by internal/gen; DO NOT EDIT.")
	fmt.Fprintf(&buf, "// Source: %s\n\n", sourceLabel)
	fmt.Fprintln(&buf, "type generatedConformanceCase struct {")
	fmt.Fprintln(&buf, "\tlineNo       int")
	fmt.Fprintln(&buf, "\tinput        []byte")
	fmt.Fprintln(&buf, "\tbreakOffsets []int")
	fmt.Fprintln(&buf, "\tcomment      string")
	fmt.Fprintln(&buf, "}")
	fmt.Fprintln(&buf)
	fmt.Fprintf(&buf, "var lineBreakConformanceTests = [%d]generatedConformanceCase{\n", len(tests))
	for _, tc := range tests {
		fmt.Fprintf(&buf, "{lineNo: %d, input: %#v, breakOffsets: %#v, comment: %#v},\n", tc.lineNo, tc.input, tc.breakOffsets, tc.comment)
	}
	fmt.Fprintln(&buf, "}")

	b := buf.Bytes()
	b = bytes.ReplaceAll(b, []byte("[]uint8{0x"), []byte("{0x"))
	b = bytes.ReplaceAll(b, []byte("[]uint8{"), []byte("[]byte{"))
	return b, nil
}

func uniqueClasses(records []record) []string {
	m := map[string]struct{}{}
	for _, r := range records {
		m[r.class] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for c := range m {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

func selectQuoteCategoryRecords(records []record) []record {
	out := make([]record, 0, 32)
	for _, rec := range records {
		switch rec.class {
		case "Pi", "Pf":
			out = append(out, record{
				lo:    rec.lo,
				hi:    rec.hi,
				class: strings.ToUpper(rec.class), // PI/PF for generated constant names
			})
		}
	}
	return out
}

func selectEastAsianRecords(records []record) []record {
	out := make([]record, 0, 4096)
	for _, rec := range records {
		switch rec.class {
		case "F", "W", "H":
			out = append(out, record{
				lo:    rec.lo,
				hi:    rec.hi,
				class: "EA",
			})
		}
	}
	return out
}

func selectExtendedPictographicUnassignedRecords(records []record) []record {
	// UAX #44 specifies Extended_Pictographic default=Y for unassigned code points
	// in these ranges.
	extPictDefaultRanges := []struct {
		lo rune
		hi rune
	}{
		{lo: 0x1F000, hi: 0x1FAFF},
		{lo: 0x1FC00, hi: 0x1FFFD},
	}

	out := make([]record, 0, 2048)
	for _, rec := range records {
		if rec.class != "Cn" {
			continue
		}
		for _, r := range extPictDefaultRanges {
			lo := maxRune(rec.lo, r.lo)
			hi := minRune(rec.hi, r.hi)
			if lo <= hi {
				out = append(out, record{lo: lo, hi: hi, class: "EPU"})
			}
		}
	}
	return out
}

func maxRune(a, b rune) rune {
	if a > b {
		return a
	}
	return b
}

func minRune(a, b rune) rune {
	if a < b {
		return a
	}
	return b
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
