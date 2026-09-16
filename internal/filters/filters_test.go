package filters

import (
	"testing"
)

// ---- ParseNumericRanges ----

func TestParseNumericRanges(t *testing.T) {
	cases := []struct {
		in    string
		count int
		first [2]int
	}{
		{"", 0, [2]int{}},
		{"200", 1, [2]int{200, 200}},
		{"200,300-399", 2, [2]int{200, 200}},
		{"100-200", 1, [2]int{100, 200}},
		{" 100 , 200 ", 2, [2]int{100, 100}},
	}
	for _, c := range cases {
		ranges, err := ParseNumericRanges(c.in)
		if err != nil {
			t.Fatalf("ParseNumericRanges(%q) error: %v", c.in, err)
		}
		if len(ranges) != c.count {
			t.Fatalf("ParseNumericRanges(%q) = %v, want %d ranges", c.in, ranges, c.count)
		}
		if c.count > 0 && (ranges[0].Min != c.first[0] || ranges[0].Max != c.first[1]) {
			t.Errorf("ParseNumericRanges(%q)[0] = %v", c.in, ranges[0])
		}
	}
}

func TestParseNumericRangesErrors(t *testing.T) {
	for _, in := range []string{"300-200", "abc", "1-,2", "200-", "1;2"} {
		if _, err := ParseNumericRanges(in); err == nil {
			t.Errorf("ParseNumericRanges(%q) should error", in)
		}
	}
}

// ---- ParseTimeFilters ----

func TestParseTimeFilters(t *testing.T) {
	filters, err := ParseTimeFilters(">100,<200,50")
	if err != nil {
		t.Fatal(err)
	}
	if len(filters) != 3 {
		t.Fatalf("filters = %v", filters)
	}
	if filters[0].Op != ">" || filters[0].Value != 100 {
		t.Errorf("filters[0] = %v", filters[0])
	}
	if filters[1].Op != "<" || filters[1].Value != 200 {
		t.Errorf("filters[1] = %v", filters[1])
	}
	if filters[2].Op != "=" || filters[2].Value != 50 {
		t.Errorf("filters[2] = %v", filters[2])
	}
	if _, err := ParseTimeFilters(""); err != nil {
		t.Error("empty should be allowed")
	}
	if _, err := ParseTimeFilters(">abc"); err == nil {
		t.Error("invalid should error")
	}
}

// ---- ParseSize ----

func TestParseSize(t *testing.T) {
	cases := []struct {
		in   string
		want int64
		err  bool
	}{
		{"", 0, false},
		{"1024", 1024, false},
		{"1KB", 1024, false},
		{"1kb", 1024, false},
		{"4 KB", 4096, false},
		{"1MB", 1024 * 1024, false},
		{"0B", 0, false},
		{"1TB", 1024 * 1024 * 1024 * 1024, false},
		{"abc", 0, true},
		{"1XB", 0, true},
		{"-5", 0, true},
	}
	for _, c := range cases {
		got, err := ParseSize(c.in)
		if c.err {
			if err == nil {
				t.Errorf("ParseSize(%q) should error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseSize(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseSize(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

// ---- ParseSizeList ----

func TestParseSizeList(t *testing.T) {
	sizes, err := ParseSizeList("0,0B,4KB")
	if err != nil {
		t.Fatal(err)
	}
	if len(sizes) != 2 {
		t.Errorf("sizes = %v, want 2 unique", sizes)
	}
	if _, ok := sizes[4096]; !ok {
		t.Errorf("4KB missing: %v", sizes)
	}
}

// ---- ValidateRegex ----

func TestValidateRegex(t *testing.T) {
	if err := ValidateRegex("", "--x"); err != nil {
		t.Error("empty pattern is valid")
	}
	if err := ValidateRegex("^404$", "--x"); err != nil {
		t.Errorf("valid regex: %v", err)
	}
	if err := ValidateRegex("([unclosed", "--exclude-regex"); err == nil {
		t.Error("invalid regex should error")
	} else if !containsStr(err.Error(), "--exclude-regex") {
		t.Errorf("error should mention option name: %v", err)
	}
}

// ---- MatchesNumericRanges / MatchesTimeFilters ----

func TestMatchesNumericRanges(t *testing.T) {
	ranges, _ := ParseNumericRanges("100-200,300")
	if !MatchesNumericRanges(150, ranges) {
		t.Error("150 should match 100-200")
	}
	if !MatchesNumericRanges(300, ranges) {
		t.Error("300 should match")
	}
	if MatchesNumericRanges(250, ranges) {
		t.Error("250 should not match")
	}
	if MatchesNumericRanges(99, ranges) {
		t.Error("99 should not match")
	}
}

func TestMatchesTimeFilters(t *testing.T) {
	filters, _ := ParseTimeFilters(">100,<200")
	if !MatchesTimeFilters(0.15, filters) {
		t.Error("150ms should match <200")
	}
	if !MatchesTimeFilters(0.25, filters) {
		t.Error("250ms should match >100")
	}
	if !MatchesTimeFilters(1.5, filters) {
		// dirsearch 语义：任一条件满足即命中（1500ms > 100）
		t.Error("1500ms matches >100, should return true")
	}

	// 任一条件都不满足时才返回 false（OR 语义）
	strict, _ := ParseTimeFilters(">100,>200")
	if MatchesTimeFilters(0.05, strict) {
		t.Error("50ms should not match >100 or >200")
	}
	if !MatchesTimeFilters(0.15, strict) {
		t.Error("150ms should match >100")
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
