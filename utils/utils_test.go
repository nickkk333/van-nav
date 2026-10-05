package utils

import "testing"

// TestGetSuffixFromUrl 回归：url 不含点时老实现会 panic（LastIndex 返回 -1 直接切片）
func TestGetSuffixFromUrl(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"https://a.b/x.png", ".png"},
		{"https://a.b/x.png?v=1", ".png"},
		{"https://a.b/x.PNG", ".PNG"},
		{"https://example.com", ""},
		{"https://a.b/no-ext", ""},
		{"", ""},
		{"no-dot-at-all", ""},
		{".hidden", ""},
	}
	for _, c := range cases {
		if got := GetSuffixFromUrl(c.in); got != c.want {
			t.Fatalf("GetSuffixFromUrl(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
