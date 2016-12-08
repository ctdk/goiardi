package indexer

import "testing"

func TestUnescapeSpecials(t *testing.T) {
	specials := []string{
		`+`, `-`, `&&`, `||`,
		`!`, `(`, `)`, `{`, `}`,
		`[`, `]`, `^`, `"`, `~`,
		`*`, `?`, `:`, `\`,
	}
	for _, s := range specials {
		in := `\` + s
		if out := unescape(in); out != s {
			t.Errorf("Unescaping %s failed. expected = %s, got = %s", in, s, out)
		}
	}
}

func TestUnescapedStrings(t *testing.T) {
	tests := []struct {
		in, out string
	}{
		{`00\:01\:02`, `00:01:02`},
		{`foo \&& bar`, `foo && bar`},
		{`some\\file\\path`, `some\file\path`},
	}
	for _, tt := range tests {
		if out := unescape(tt.in); out != tt.out {
			t.Errorf("Unescaping %s failed. expected = %s, got = %s", tt.in, tt.out, out)
		}
	}
}
